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
	s.logger.Debugf("Available tool: cityTime (cities: nyc, sf, boston)")

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
		&mcp.Implementation{Name: "stackql", Version: "v0.1.0"},
		nil,
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "greet",
			Description: "Say hi.  A simple liveness check.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args GreetInput) (*mcp.CallToolResult, any, error) {
			greeting, greetingErr := backend.Greet(ctx, args)
			if greetingErr != nil {
				return nil, nil, greetingErr
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: greeting},
				},
			}, nil, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "server_info",
			Description: "Get server information",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, ServerInfoOutput, error) {
			rv, rvErr := backend.ServerInfo(ctx, args)
			if rvErr != nil {
				return nil, ServerInfoOutput{}, rvErr
			}
			return nil, rv, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "db_identity",
			Description: "get current database identity",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, map[string]any, error) {
			rv, rvErr := backend.DBIdentity(ctx, args)
			if rvErr != nil {
				return nil, nil, rvErr
			}
			return nil, rv, nil
		},
	)
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "query_v2",
			Description: "Execute a SQL query.  Please adhere to the expected parameters.  Returns a textual response",
			// Input and output schemas can be defined here if needed.
		},
		func(ctx context.Context, req *mcp.CallToolRequest, arg QueryInput) (*mcp.CallToolResult, any, error) {
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
			Name:        "query_json_v2",
			Description: "Execute a SQL query and return a JSON array of rows, as text.",
			// Input and output schemas can be defined here if needed.
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args QueryJSONInput) (*mcp.CallToolResult, any, error) {
			arr, err := backend.RunQueryJSON(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			bytesArr, marshalErr := json.Marshal(arr)
			if marshalErr != nil {
				return nil, nil, fmt.Errorf("failed to marshal query result to JSON: %w", marshalErr)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(bytesArr)},
				},
			}, nil, nil
		},
	)

	// mcp.AddTool(
	// 	server,
	// 	&mcp.Tool{
	// 		Name:        "list_table_resources",
	// 		Description: "List resource URIs for tables in a schema.",
	// 	},
	// 	func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
	// 		result, err := backend.ListTableResources(ctx, args)
	// 		if err != nil {
	// 			return nil, nil, err
	// 		}
	// 		return &mcp.CallToolResult{
	// 			Content: []mcp.Content{
	// 				&mcp.TextContent{Text: fmt.Sprintf("%v", result)},
	// 			},
	// 		}, result, nil
	// 	},
	// )

	// mcp.AddTool(
	// 	server,
	// 	&mcp.Tool{
	// 		Name:        "read_table_resource",
	// 		Description: "Read rows from a table resource.",
	// 	},
	// 	func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
	// 		result, err := backend.ReadTableResource(ctx, args)
	// 		if err != nil {
	// 			return nil, nil, err
	// 		}
	// 		return &mcp.CallToolResult{
	// 			Content: []mcp.Content{
	// 				&mcp.TextContent{Text: fmt.Sprintf("%v", result)},
	// 			},
	// 		}, result, nil
	// 	},
	// )

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "prompt_write_safe_select_tool",
			Description: "Prompt: guidelines for writing safe SELECT queries.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.PromptWriteSafeSelectTool(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
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
		func(ctx context.Context, req *mcp.CallToolRequest, args ListTablesInput) (*mcp.CallToolResult, any, error) {
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
		func(ctx context.Context, req *mcp.CallToolRequest, args ListTablesPageInput) (*mcp.CallToolResult, any, error) {
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
			Description: "List all schemas/providers in the database.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListProviders(ctx)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_services",
			Description: "List services for a provider.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListServices(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_resources",
			Description: "List resources for a service.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListResources(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "list_methods",
			Description: "List methods for a resource.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.ListMethods(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "describe_table",
			Description: "Get detailed information about a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.DescribeTable(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_foreign_keys",
			Description: "Get foreign key information for a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.GetForeignKeys(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "find_relationships",
			Description: "Find explicit and implied relationships for a table.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args HierarchyInput) (*mcp.CallToolResult, any, error) {
			result, err := backend.FindRelationships(ctx, args)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, result, nil
		},
	)

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
