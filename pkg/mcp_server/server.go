package mcp_server //nolint:revive // fine for now

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
	"github.com/stackql/stackql/pkg/mcp_server/render"
	"github.com/stackql/stackql/pkg/sink"
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
	auditSink sink.Sink

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
	}, &mcp.StreamableHTTPOptions{Stateless: config.Server.Stateless})

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

// resolveRenderFormat returns the effective text render format for a tool
// call: a legal per-call `format` argument wins, otherwise the server-level
// default applies (issue #669).  An unrecognised per-call value is an error
// so machine callers fail fast rather than silently getting markdown.
func resolveRenderFormat(cfg *Config, requested string) (string, error) {
	if !render.IsLegalFormat(requested) {
		return "", fmt.Errorf("invalid format %q (legal: markdown, json)", requested)
	}
	if requested != "" {
		return requested, nil
	}
	return cfg.GetRender(), nil
}

// textForFormat renders the text content for a tool result.  In JSON mode the
// value mirrored into structuredContent is rendered as compact JSON (row sets
// have their database/sql nullable wrappers unwrapped first); otherwise the
// supplied markdown renderer runs.
func textForFormat(format string, v any, markdown func() string) string {
	if format == render.FormatJSON {
		if q, isQueryResult := v.(dto.QueryResultDTO); isQueryResult {
			v = dto.QueryResultDTO{Rows: render.UnwrapRows(q.Rows)}
		}
		return render.JSONValue(v)
	}
	return markdown()
}

// mcpDefaultAuditFilename returns the default audit log filename the MCP
// server uses when the operator did not configure an explicit path.  The
// naming convention is the one shipped with PR2: stackql_mcp_server_<UTC>.log
// in cwd.  The generic sink package picks this up via FileConfig.DefaultFilename.
func mcpDefaultAuditFilename(t time.Time) string {
	return fmt.Sprintf("stackql_mcp_server_%s.log", t.UTC().Format("20060102T150405Z"))
}

// initAuditSink constructs the audit sink dictated by cfg.  When audit is
// disabled it returns a nop sink so the rest of the code can be uniform.
//
// The MCP server takes responsibility for the "where do logs land?" default:
// when neither Path nor Dir is supplied in mcp.config, we default Dir to cwd
// (".") so existing operators see the same behaviour as PR2.  The generic
// pkg/sink package itself refuses to silently pick a directory.
//
// The otel format wraps the same destination with the OTLP/JSON encoder
// (issue #729); serviceVersion lands in the resource attributes.
func initAuditSink(cfg *Config, logger *logrus.Logger, serviceVersion string) (sink.Sink, error) {
	if !cfg.IsAuditEnabled() {
		return sink.NewNopSink(), nil
	}
	fileCfg := cfg.Server.Audit.File
	if fileCfg.Path == "" && fileCfg.Dir == "" {
		fileCfg.Dir = "."
	}
	if fileCfg.DefaultFilename == nil {
		fileCfg.DefaultFilename = mcpDefaultAuditFilename
	}
	if cfg.Server.Audit.Sink != "" && cfg.Server.Audit.Sink != "file" {
		logger.Warnf("unknown audit sink %q; falling back to file", cfg.Server.Audit.Sink)
	}
	s, err := sink.NewFileSink(fileCfg)
	if err != nil {
		return nil, fmt.Errorf("audit file sink: %w", err)
	}
	if cfg.Server.Audit.GetFormat() == audit.FormatOTel {
		return audit.NewOTelSink(s, serviceVersion), nil
	}
	return s, nil
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

	serverInfo, _ := backend.ServerInfo(context.Background(), nil)
	sink, err := initAuditSink(config, logger, serverInfo.Version)
	if err != nil {
		return nil, err
	}

	serverOpts := &mcp.ServerOptions{}
	if !config.DisableInstructions {
		instructions, instrErr := loadEmbeddedInstructions()
		if instrErr != nil {
			return nil, fmt.Errorf("embedded instructions: %w", instrErr)
		}
		serverOpts.Instructions = instructions
	}
	server := mcp.NewServer(
		&mcp.Implementation{Name: "stackql", Version: "v0.1.1"},
		serverOpts,
	)

	if toolsErr := registerTools(server, config, backend, logger, sink); toolsErr != nil {
		return nil, toolsErr
	}
	if promptsErr := registerEmbeddedPrompts(server, config); promptsErr != nil {
		return nil, promptsErr
	}
	if resourcesErr := registerEmbeddedResources(server, config); resourcesErr != nil {
		return nil, resourcesErr
	}

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

// registryGate is the toolGate for list_registry and pull_provider.  Both are
// classified as QueryClassSelect so they Allow under every mode; pulling a
// provider writes only to the local approot cache (no cloud control/data
// plane effect) per the issue's "not a cloud mutation" rationale.  The audit
// record still gets written.
func registryGate(name string) toolGate {
	return toolGate{
		toolName:     name,
		defaultClass: policy.QueryClassSelect,
		extractArgs:  extractArgsFromRegistryInput,
	}
}

//nolint:funlen,gocognit // tool registrations are inherently long and branchy
func registerTools(server *mcp.Server, cfg *Config, backend Backend, logger *logrus.Logger, auditSink sink.Sink) error {
	descriptions, descriptionsErr := loadEmbeddedToolDescriptions()
	if descriptionsErr != nil {
		return fmt.Errorf("embedded tool descriptions: %w", descriptionsErr)
	}
	var errs []error
	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("server_info"),
		&mcp.Tool{Name: "server_info"},
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
			text := textForFormat(cfg.GetRender(), out, func() string { return render.RenderKV("Server Info", rec) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("list_providers"),
		&mcp.Tool{Name: "list_providers"},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			rows, err := backend.ListProviders(ctx)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(cfg.GetRender(), out, func() string { return render.RenderTable(rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("list_services"),
		&mcp.Tool{Name: "list_services"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.ListServices(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderTable(rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("list_resources"),
		&mcp.Tool{Name: "list_resources"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.ListResources(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderTable(rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("list_methods"),
		&mcp.Tool{Name: "list_methods"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.ListMethods(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderTable(rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("describe_resource"),
		&mcp.Tool{Name: "describe_resource"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.DescribeResource(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderKV("Resource", rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, selectGate("describe_method"),
		&mcp.Tool{Name: "describe_method"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.DescribeMethod(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderKV("Method", rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions,
		toolGate{
			toolName:     "validate_select_query",
			defaultClass: policy.QueryClassSelect, // validation is read-only by definition.
			extractArgs:  extractArgsFromQueryInput,
		},
		&mcp.Tool{Name: "validate_select_query"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, dto.ValidationResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.ValidationResultDTO{}, formatErr
			}
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
			text := textForFormat(format, out, func() string { return render.RenderKV("Validation Result", rec) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, queryGate("run_select_query"),
		&mcp.Tool{Name: "run_select_query"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			logger.Debugf("run_select_query: %s", args.SQL)
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.RunQueryJSON(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderTable(rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, registerExecQueryTool(
		server, cfg, backend, auditSink, descriptions, "run_mutation_query", "Mutation Result"))

	errs = append(errs, registerExecQueryTool(
		server, cfg, backend, auditSink, descriptions, "run_lifecycle_operation", "Lifecycle Result"))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, registryGate("list_registry"),
		&mcp.Tool{Name: "list_registry"},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.RegistryInput) (*mcp.CallToolResult, dto.QueryResultDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryResultDTO{}, formatErr
			}
			rows, err := backend.ListRegistry(ctx, args)
			if err != nil {
				return nil, dto.QueryResultDTO{}, err
			}
			out := dto.QueryResultDTO{Rows: rows}
			text := textForFormat(format, out, func() string { return render.RenderTable(rows) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	))

	errs = append(errs, registerReloadCredentialsTool(server, cfg, backend, auditSink, descriptions))

	errs = append(errs, addToolWithGate(
		server, cfg, auditSink, descriptions, registryGate("pull_provider"),
		&mcp.Tool{
			Name: "pull_provider",
			// Writes local cache state, so not read-only despite the select-class gate.
			Annotations: &mcp.ToolAnnotations{IdempotentHint: true, DestructiveHint: boolPtr(false)},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.RegistryInput) (*mcp.CallToolResult, map[string]any, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, nil, formatErr
			}
			res, err := backend.PullProvider(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			text := textForFormat(format, res, func() string { return render.RenderKV("Pull Result", []map[string]any{res}) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, res, nil
		},
	))

	errs = append(errs, registerQueryLibraryTools(server, cfg, auditSink, descriptions))
	return errors.Join(errs...)
}

// registerExecQueryTool registers a mutation-shaped tool (mutation or
// lifecycle): SQL in, {messages, timestamp} out, explicitly destructive.
func registerExecQueryTool(
	server *mcp.Server, cfg *Config, backend Backend, auditSink sink.Sink,
	descriptions toolDescriptions, name, kvTitle string,
) error {
	return addToolWithGate(
		server, cfg, auditSink, descriptions, queryGate(name),
		&mcp.Tool{
			Name:        name,
			Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, map[string]any, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, nil, formatErr
			}
			res, err := backend.ExecQuery(ctx, args.SQL)
			if err != nil {
				return nil, nil, err
			}
			text := textForFormat(format, res, func() string { return render.RenderKV(kvTitle, []map[string]any{res}) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, res, nil
		},
	)
}

// registerReloadCredentialsTool publishes reload_credentials (issue #688);
// classified as a select so it is allowed in every mode, read_only included.
func registerReloadCredentialsTool(
	server *mcp.Server, cfg *Config, backend Backend, auditSink sink.Sink, descriptions toolDescriptions,
) error {
	return addToolWithGate(
		server, cfg, auditSink, descriptions,
		toolGate{
			toolName:     "reload_credentials",
			defaultClass: policy.QueryClassSelect,
			extractArgs: func(args any) map[string]any {
				v, ok := args.(dto.CredentialsReloadInput)
				if !ok || v.Provider == "" {
					return nil
				}
				return map[string]any{"provider": v.Provider}
			},
		},
		&mcp.Tool{
			Name: "reload_credentials",
			// Mutates process env, so not read-only despite the select-class gate.
			Annotations: &mcp.ToolAnnotations{IdempotentHint: true, DestructiveHint: boolPtr(false)},
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.CredentialsReloadInput) (*mcp.CallToolResult, dto.CredentialsReloadDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.CredentialsReloadDTO{}, formatErr
			}
			out, err := backend.ReloadCredentials(ctx, args)
			if err != nil {
				return nil, dto.CredentialsReloadDTO{}, err
			}
			rec := make([]map[string]any, 0, len(out.Providers))
			for _, p := range out.Providers {
				rec = append(rec, map[string]any{
					"provider":     p.Provider,
					"auth_type":    p.AuthType,
					"sourced_from": p.SourcedFrom,
					"status":       p.Status,
					"changed":      p.Changed,
					"detail":       p.Detail,
				})
			}
			text := textForFormat(format, out, func() string { return render.RenderTable(rec) })
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
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
		// Handles CRLF framing (issue #668) and malformed frames (issue
		// #701); see resilientStdioTransport.
		return s.server.Run(ctx, newResilientStdioTransport(s.logger))
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
