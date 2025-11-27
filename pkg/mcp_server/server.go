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
	"github.com/stackql/stackql/pkg/mcp_server/dto"
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
	config  *Config
	backend Backend
	logger  *logrus.Logger

	server *mcp.Server

	// Concurrency control
	requestSemaphore *semaphore.Weighted

	// Server state
	mu      sync.RWMutex
	running bool
	servers []io.Closer // Track all running servers for cleanup
}

func (s *simpleMCPServer) runHTTPServer(server *mcp.Server, config *Config) error {
	// Create the streamable HTTP handler.
	address := config.GetServerAddress()
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	handlerWithLogging := loggingHandler(handler, s.logger)

	s.logger.Debugf("MCP server listening on %s", address)
	// s.logger.Debugf("Available tool: cityTime (cities: nyc, sf, boston)")

	// Start the HTTP server with logging handler.
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

func NewExampleBackendServer(config *Config, logger *logrus.Logger) (MCPServer, error) {
	backend := NewExampleBackend("example-connection-string")
	return newMCPServer(config, backend, logger)
}

func NewAgnosticBackendServer(backend Backend, config *Config, logger *logrus.Logger) (MCPServer, error) {
	return newMCPServer(config, backend, logger)
}

// func NewExampleHTTPBackendServer(config *Config, logger *logrus.Logger) (MCPServer, error) {
// 	backend := NewExampleBackend("example-connection-string")
// 	if config == nil {
// 		config = DefaultHTTPConfig()
// 	}
// 	return NewMCPServer(config, backend, logger)
// }

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
		// logger.SetOutput(io.Discard)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "stackql", Version: "v0.1.1"},
		nil,
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "greet",
			Description: "Say hi.  A simple liveness check.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.GreetInput) (*mcp.CallToolResult, any, error) {
			greeting, greetingErr := backend.Greet(ctx, args)
			if greetingErr != nil {
				return nil, nil, greetingErr
			}
			out := dto.GreetDTO{Greeting: greeting}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "server_info",
			Description: "Get server information",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, dto.ServerInfoDTO, error) {
			rv, rvErr := backend.ServerInfo(ctx, args)
			if rvErr != nil {
				return nil, dto.ServerInfoDTO{}, rvErr
			}
			out := dto.ServerInfoDTO{Name: rv.Name, Info: rv.Info, IsReadOnly: rv.IsReadOnly}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "db_identity",
			Description: "get current database identity",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, dto.DBIdentityDTO, error) {
			rv, rvErr := backend.DBIdentity(ctx, args)
			if rvErr != nil {
				return nil, dto.DBIdentityDTO{}, rvErr
			}
			out := dto.DBIdentityDTO{Identity: fmt.Sprintf("%v", rv["identity"])}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query_v2",
			Description: "Execute a SQL query.  Please adhere to the expected parameters.  Returns a textual response",
			// Input and output schemas can be defined here if needed.
		},
		func(ctx context.Context, req *mcp.CallToolRequest, arg dto.QueryInput) (*mcp.CallToolResult, any, error) {
			logger.Warnf("Received query: %s", arg.SQL)
			rv, rvErr := backend.RunQuery(ctx, arg)
			if rvErr != nil {
				return nil, nil, rvErr
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: rv},
				},
			}, nil, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query_v3",
			Description: "Execute a SQL query.  Returns a DTO with rows or raw text.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, arg dto.QueryInput) (*mcp.CallToolResult, any, error) {
			logger.Warnf("Received query: %s", arg.SQL)
			raw, rvErr := backend.RunQuery(ctx, arg)
			if rvErr != nil {
				return nil, nil, rvErr
			}
			out := dto.QueryResultDTO{Format: arg.Format, Raw: raw}
			if arg.Format == "json" {
				var rows []map[string]any
				if json.Unmarshal([]byte(raw), &rows) == nil {
					out.Rows = rows
					out.RowCount = len(rows)
					out.Raw = ""
				}
			}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query_json_v2",
			Description: "Execute a SQL query and return a JSON array of rows, as text plus DTO.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, any, error) {
			arr, err := backend.RunQueryJSON(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: arr, RowCount: len(arr), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "validate_query_json_v2",
			Description: "Validate a SQL SELECT query ahead of time and return a JSON object expressing success, or else an error.  Supply the query exactly as you would execute it, no qualifying keywords.  Only works for SELECT at this time.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, any, error) {
			arr, err := backend.ValidateQuery(ctx, args.SQL)
			isValid := err == nil
			message := "Query validation succeeded."
			var errorsToPublish []string
			if err != nil {
				errorsToPublish = append(errorsToPublish, err.Error())
				arrBytes, _ := json.Marshal(arr)
				message = fmt.Sprintf("Query validation failed, returned data: %s", string(arrBytes))
			}
			out := dto.ValidationResultDTO{IsValid: isValid, Errors: errorsToPublish, Message: message}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "exec_query_json_v2",
			Description: "Exec query pattern; for non-read operations. Tread carefully!!! These are almost always mutations!  Execute a SQL query and return an optional JSON object, describing the effect(s).",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, any, error) {
			res, err := backend.ExecQuery(ctx, args.SQL)
			if err != nil {
				return nil, nil, err
			}
			bytesOut, _ := json.Marshal(res)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, res, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query_validate_json_v2",
			Description: "Run an ahead of time (AOT) check for query tractability.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.QueryJSONInput) (*mcp.CallToolResult, any, error) {
			res, err := backend.ExecQuery(ctx, args.SQL)
			if err != nil {
				return nil, nil, err
			}
			bytesOut, _ := json.Marshal(res)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, res, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "prompt_write_safe_select_tool",
			Description: "Prompt: guidelines for writing safe SELECT queries.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.PromptWriteSafeSelectTool(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.SimpleTextDTO{Text: result}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	// mcp.AddTool(
	// 	server,
	// 	&mcp.Tool{
	// 		Name:        "prompt_explain_plan_tips_tool",
	// 		Description: "Prompt: tips for reading EXPLAIN ANALYZE output.",
	// 	},
	// 	func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	// 		result, err := backend.PromptExplainPlanTipsTool(ctx)
	// 		if err != nil {
	// 			return nil, nil, err
	// 		}
	// 		return &mcp.CallToolResult{
	// 			Content: []mcp.Content{
	// 				&mcp.TextContent{Text: result},
	// 			},
	// 		}, result, nil
	// 	},
	// )

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_tables_json",
			Description: "List tables in a schema and return JSON rows.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.ListTablesInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListTablesJSON(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			bytesArr, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				return nil, nil, fmt.Errorf("failed to marshal result to JSON: %w", marshalErr)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(bytesArr)},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_tables_json_page",
			Description: "List tables with pagination and filters, returns JSON.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.ListTablesPageInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListTablesJSONPage(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			bytesArr, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				return nil, nil, fmt.Errorf("failed to marshal result to JSON: %w", marshalErr)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(bytesArr)},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_providers",
			Description: "List available providers.  This is the top level of the stackql hierarchy.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListProviders(ctx)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_services",
			Description: "List services. **must** supply <provider>.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListServices(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_resources",
			Description: "List available resources. **must** supply <provider>, <service>.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListResources(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_methods",
			Description: "List access methods for a resource.  Interrogating this is almost compulsory before executing a CRUD query; you will need to infer requireed WHERE parameters. **must** supply <provider>, <service>, <resource>.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListMethods(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "describe_table",
			Description: "Get detailed information about a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.DescribeTable(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_foreign_keys",
			Description: "Get foreign key information for a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.GetForeignKeys(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "find_relationships",
			Description: "Find explicit and implied relationships for a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.FindRelationships(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.SimpleTextDTO{Text: result}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	// --- new: register namespaced meta.* and query.* tools ---
	registerNamespacedTools(server, backend, logger)
	// ---------------------------------------------------------

	return &simpleMCPServer{
		config:           config,
		backend:          backend,
		logger:           logger,
		server:           server,
		requestSemaphore: semaphore.NewWeighted(int64(config.Server.MaxConcurrentRequests)),
		servers:          make([]io.Closer, 0),
	}, nil
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
		// Default to stdio transport
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

	// Close all servers
	var errs []error
	for _, server := range s.servers {
		if err := server.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close backend
	if err := s.backend.Close(); err != nil {
		errs = append(errs, err)
	}

	s.running = false
	s.servers = s.servers[:0]

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	s.logger.Printf("MCP server stopped")
	return nil
}

// registerNamespacedTools adds meta.* and query.* tools (namespaced variants).
//
//nolint:gocognit,funlen // ok for now
func registerNamespacedTools(server *mcp.Server, backend Backend, logger *logrus.Logger) {
	// meta.server_info
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "meta.server_info",
			Description: "Namespaced: Get server information.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, dto.ServerInfoDTO, error) {
			info, err := backend.ServerInfo(ctx, nil)
			if err != nil {
				return nil, dto.ServerInfoDTO{}, err
			}
			out := dto.ServerInfoDTO{Name: info.Name, Info: info.Info, IsReadOnly: info.IsReadOnly}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	// meta.db_identity
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "meta.db_identity",
			Description: "Namespaced: Get current database identity.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, dto.DBIdentityDTO, error) {
			id, err := backend.DBIdentity(ctx, nil)
			if err != nil {
				return nil, dto.DBIdentityDTO{}, err
			}
			out := dto.DBIdentityDTO{Identity: fmt.Sprintf("%v", id["identity"])}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query.exec_text",
			Description: "Namespaced: Execute SQL returning textual result.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, arg dto.QueryInput) (*mcp.CallToolResult, any, error) {
			logger.Infof("query.exec_text SQL: %s", arg.SQL)
			rawText, err := backend.RunQuery(ctx, arg)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Raw: rawText, Format: "text"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query.exec_json",
			Description: "Namespaced: Execute SQL returning JSON array as text.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, arg dto.QueryJSONInput) (*mcp.CallToolResult, any, error) {
			rows, err := backend.RunQueryJSON(ctx, arg)
			if err != nil {
				return nil, nil, err
			}
			dtObj := dto.QueryResultDTO{
				Rows:     rows,
				RowCount: len(rows),
				Format:   "json",
			}
			bytesOut, _ := json.Marshal(dtObj)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, dtObj, nil
		},
	)

	// meta_describe_table
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "meta_describe_table",
			Description: "Describe a stackql relation.  This publishes the bullk of the columns returned from a SELECT.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.DescribeTable(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	// meta.get_foreign_keys
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "meta.get_foreign_keys",
			Description: "Namespaced: Get foreign keys for a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.GetForeignKeys(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.QueryResultDTO{Rows: result, RowCount: len(result), Format: "json"}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)

	// meta.find_relationships
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "meta.find_relationships",
			Description: "Namespaced: Find relationships for a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args dto.HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.FindRelationships(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			out := dto.SimpleTextDTO{Text: result}
			bytesOut, _ := json.Marshal(out)
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(bytesOut)}}}, out, nil
		},
	)
}
