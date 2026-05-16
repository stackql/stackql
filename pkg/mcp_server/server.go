package mcp_server //nolint:revive // fine for now

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
	"github.com/stackql/stackql/pkg/mcp_server/render"
)

const (
	serverTransportStdIO     = "stdio"
	serverTransportHTTP      = "http"
	serverTransportSSE       = "sse"
	DefaultHTTPServerAddress = "127.0.0.1:9876"
)

type MCPServer interface {
	Start(context.Context) error
	Stop() error
}

// simpleMCPServer implements the Model Context Protocol server for StackQL.
type simpleMCPServer struct {
	config    *Config
	backend   Backend
	logger    *logrus.Logger
	auditSink audit.Sink

	server *mcp.Server

	// Concurrency control
	requestSemaphore *semaphore.Weighted

	// Server state
	mu      sync.RWMutex
	running bool
	servers []io.Closer // Track all running servers for cleanup
}

func (s *simpleMCPServer) runHTTPServer(server *mcp.Server, config *Config) error {
	address := config.GetServerAddress()
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	handlerWithLogging := loggingHandler(handler, s.logger)

	s.logger.Debugf("MCP server listening on %s", address)

	//nolint:gosec // TODO: find viable alternative to http.ListenAndServe
	if config.Server.TLSCertFile != "" && config.Server.TLSKeyFile != "" {
		s.logger.Infof("Starting HTTPS server on %s", address)
		if err := http.ListenAndServeTLS(address, config.Server.TLSCertFile, config.Server.TLSKeyFile, handlerWithLogging); err != nil {
			s.logger.Errorf("HTTPS Server failed: %v", err)
			return err
		}
		return nil
	}
	//nolint:gosec // TODO: find viable alternative to http.ListenAndServe
	if err := http.ListenAndServe(address, handlerWithLogging); err != nil {
		s.logger.Errorf("Server failed: %v", err)
		return err
	}
	return nil
}

// addPromptIfEnabled registers a prompt only when cfg.IsPromptEnabled allows it.
func addPromptIfEnabled(s *mcp.Server, cfg *Config, p *mcp.Prompt, h mcp.PromptHandler) {
	if !cfg.IsPromptEnabled(p.Name) {
		return
	}
	s.AddPrompt(p, h)
}

func NewExampleBackendServer(config *Config, logger *logrus.Logger) (MCPServer, error) {
	backend := NewExampleBackend("example-connection-string")
	return newMCPServer(config, backend, logger)
}

func NewAgnosticBackendServer(backend Backend, config *Config, logger *logrus.Logger) (MCPServer, error) {
	return newMCPServer(config, backend, logger)
}

func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal: %v"}`, err)
	}
	return string(b)
}

// initAuditSink constructs the audit sink dictated by cfg.  When audit is
// disabled it returns a nop sink so the rest of the code can be uniform.
func initAuditSink(cfg *Config, logger *logrus.Logger) (audit.Sink, error) {
	if !cfg.IsAuditEnabled() {
		return audit.NewNopSink(), nil
	}
	switch cfg.Server.Audit.Sink {
	case "", "file":
		sink, err := audit.NewFileSink(cfg.Server.Audit.File)
		if err != nil {
			return nil, fmt.Errorf("audit file sink: %w", err)
		}
		return sink, nil
	default:
		logger.Warnf("unknown audit sink %q; falling back to file", cfg.Server.Audit.Sink)
		return audit.NewFileSink(cfg.Server.Audit.File)
	}
}

// NewMCPServer creates a new MCP server with the provided configuration and backend.
//
//nolint:gocognit,funlen // ok
func newMCPServer(config *Config, backend Backend, logger *logrus.Logger) (MCPServer, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if backend == nil {
		return nil, fmt.Errorf("backend is required")
	}
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	sink, err := initAuditSink(config, logger)
	if err != nil {
		return nil, err
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "stackql", Version: "v0.1.1"},
		nil,
	)

	registerTools(server, config, backend, logger, sink)
	registerPrompts(server, config)

	return &simpleMCPServer{
		config:           config,
		backend:          backend,
		logger:           logger,
		auditSink:        sink,
		server:           server,
		requestSemaphore: semaphore.NewWeighted(int64(config.Server.MaxConcurrentRequests)),
		servers:          make([]io.Closer, 0),
	}, nil
}

// selectGate is the toolGate shape for SELECT/metadata tools that take no SQL.
// Decision is Allow under every mode, so the gate is effectively pass-through;
// the audit record still gets written.
func selectGate(name string) toolGate {
	return toolGate{
		toolName:     name,
		defaultClass: policy.QueryClassSelect,
		extractArgs:  extractArgsFromHierarchy,
	}
}

// queryGate is the toolGate shape for tools that take a SQL string as input.
// The classifier inspects args.SQL.
func queryGate(name string) toolGate {
	return toolGate{
		toolName:     name,
		defaultClass: policy.QueryClassUnknown,
		extractSQL:   extractSQLFromQueryInput,
		extractArgs:  extractArgsFromQueryInput,
	}
}

//nolint:funlen,gocognit // tool registrations are inherently long and branchy
func registerTools(server *mcp.Server, cfg *Config, backend Backend, logger *logrus.Logger, sink audit.Sink) {
	addToolWithGate(
		server, cfg, sink, selectGate("server_info"),
		&mcp.Tool{
			Name:        "server_info",
			Description: "Get server identity and runtime: stackql version, backing SQL engine, provider registry location, mode, read-only flag. Call once at session start.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args any) (*mcp.CallToolResult, dto.ServerInfoDTO, error) {
			rv, err := backend.ServerInfo(ctx, args)
			if err != nil {
				return nil, dto.ServerInfoDTO{}, err
			}
			out := dto.ServerInfoDTO{
				Version:          rv.Version,
				Commit:           rv.Commit,
				BuildDate:        rv.BuildDate,
				Platform:         rv.Platform,
				Transport:        rv.Transport,
				SQLBackend:       rv.SQLBackend,
				ProviderRegistry: rv.ProviderRegistry,
				Mode:             rv.Mode,
				ReadOnly:         rv.ReadOnly,
			}
			rec := []map[string]any{{
				"version":           out.Version,
				"commit":            out.Commit,
				"build_date":        out.BuildDate,
				"platform":          out.Platform,
				"transport":         out.Transport,
				"sql_backend":       out.SQLBackend,
				"provider_registry": out.ProviderRegistry,
				"mode":              out.Mode,
				"is_read_only":      out.ReadOnly,
			}}
			text := render.RenderKV("Server Info", rec)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, selectGate("list_providers"),
		&mcp.Tool{
			Name:        "list_providers",
			Description: "Available cloud/SaaS providers (top of the hierarchy). No inputs.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.ListProviders(ctx)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderTable(rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, selectGate("list_services"),
		&mcp.Tool{
			Name:        "list_services",
			Description: "Services under a provider. Requires provider.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.ListServices(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderTable(rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, selectGate("list_resources"),
		&mcp.Tool{
			Name:        "list_resources",
			Description: "Resources under a provider.service. Requires provider and service.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.ListResources(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderTable(rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, selectGate("list_methods"),
		&mcp.Tool{
			Name:        "list_methods",
			Description: "Access methods (HTTP operations) for a resource. Call before writing any query. Requires provider, service, resource.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.ListMethods(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderTable(rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, selectGate("describe_resource"),
		&mcp.Tool{
			Name:        "describe_resource",
			Description: "Output fields for a resource's primary read method. Requires provider, service, resource.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.DescribeResource(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderKV("Resource", rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, selectGate("describe_method"),
		&mcp.Tool{
			Name:        "describe_method",
			Description: "Full I/O contract for one method. Requires provider, service, resource, method.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.DescribeMethod(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderKV("Method", rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink,
		toolGate{
			toolName:     "validate_select_query",
			defaultClass: policy.QueryClassSelect, // validation is read-only by definition.
			extractArgs:  extractArgsFromQueryInput,
		},
		&mcp.Tool{
			Name:        "validate_select_query",
			Description: "Parse and plan a SELECT without executing. Returns {valid, errors}. SELECT only.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, dto.ValidationResultDTO, error) {
			rowsBack, err := backend.ValidateQuery(ctx, args.SQL)
			isValid := err == nil
			var errs []string
			if err != nil {
				errs = append(errs, err.Error())
			}
			out := dto.ValidationResultDTO{Valid: isValid, Errors: errs}
			rec := []map[string]any{{
				"valid":  out.Valid,
				"errors": mustMarshal(out.Errors),
			}}
			if !isValid {
				rec[0]["explain_output"] = mustMarshal(rowsBack)
			}
			text := render.RenderKV("Validation Result", rec)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, queryGate("run_select_query"),
		&mcp.Tool{
			Name:        "run_select_query",
			Description: "Execute a SELECT. Returns {rows}. Reads only.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			logger.Debugf("run_select_query: %s", args.SQL)
			rows, err := backend.RunQueryJSON(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: render.RenderTable(rows)}}}, out, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, queryGate("run_mutation_query"),
		&mcp.Tool{
			Name:        "run_mutation_query",
			Description: "Execute INSERT/UPDATE/REPLACE/DELETE against the provider. Real side effects. Returns {messages, timestamp}. Gated by server mode.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, map[string]any, error) {
			res, err := backend.ExecQuery(ctx, args.SQL)
			if err != nil {
				return nil, nil, err
			}
			text := render.RenderKV("Mutation Result", []map[string]any{res})
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, res, nil
		},
	)

	addToolWithGate(
		server, cfg, sink, queryGate("run_lifecycle_operation"),
		&mcp.Tool{
			Name:        "run_lifecycle_operation",
			Description: "Execute a stackql EXEC lifecycle operation. Returns {messages, timestamp}. Gated by server mode.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, map[string]any, error) {
			res, err := backend.ExecQuery(ctx, args.SQL)
			if err != nil {
				return nil, nil, err
			}
			text := render.RenderKV("Lifecycle Result", []map[string]any{res})
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, res, nil
		},
	)
}

func registerPrompts(server *mcp.Server, config *Config) {
	addPromptIfEnabled(
		server,
		config,
		&mcp.Prompt{
			Name:        "write_safe_select",
			Description: "Guidance for writing safe SELECT queries against stackql resources.",
		},
		func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return &mcp.GetPromptResult{
				Messages: []*mcp.PromptMessage{{
					Role:    "user",
					Content: &mcp.TextContent{Text: ExplainerPromptWriteSafeSelectTool},
				}},
			}, nil
		},
	)
}

// Start starts the MCP server with all configured transports.
//
//nolint:errcheck // ok for now
func (s *simpleMCPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer func() {
		s.mu.Unlock()
		s.running = false
	}()
	if s.running {
		return fmt.Errorf("server is already running")
	}
	s.running = true
	return s.run(ctx)
}

// Synchronous server run.
func (s *simpleMCPServer) run(ctx context.Context) error {
	switch s.config.GetServerTransport() {
	case serverTransportHTTP:
		return s.runHTTPServer(s.server, s.config)
	case serverTransportSSE:
		return fmt.Errorf("SSE transport obsoleted; use streamable HTTP transport instead")
	case serverTransportStdIO:
		return s.server.Run(ctx, &mcp.StdioTransport{})
	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Server.Transport)
	}
}

// Stop gracefully stops the MCP server and all transports.
func (s *simpleMCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	var errs []error
	for _, server := range s.servers {
		if err := server.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := s.backend.Close(); err != nil {
		errs = append(errs, err)
	}
	if s.auditSink != nil {
		if err := s.auditSink.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	s.running = false
	s.servers = s.servers[:0]

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	s.logger.Printf("MCP server stopped")
	return nil
}
