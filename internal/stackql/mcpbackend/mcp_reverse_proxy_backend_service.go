package mcpbackend

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/pkg/mcp_server"
)

type stackqlMCPReverseProxyService struct {
	isReadOnly   bool
	dsn          string
	handlerCtx   handler.HandlerContext
	logger       *logrus.Logger
	db           *sql.DB
	interrogator StackqlInterrogator
	renderer     resultsRenderer
}

func NewStackqlMCPReverseProxyService(
	isReadOnly bool,
	dsn string,
	db *sql.DB,
	handlerCtx handler.HandlerContext,
	logger *logrus.Logger,
) (mcp_server.Backend, error) {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}
	if handlerCtx == nil {
		return nil, fmt.Errorf("handler context is nil")
	}
	return &stackqlMCPReverseProxyService{
		dsn:          dsn,
		isReadOnly:   isReadOnly,
		interrogator: NewSimpleStackqlInterrogator(),
		logger:       logger,
		handlerCtx:   handlerCtx,
		db:           db,
		renderer:     NewResultsRenderer(),
	}, nil
}

func (b *stackqlMCPReverseProxyService) getDefaultFormat() string {
	return resultsFormatMarkdown
}

func (b *stackqlMCPReverseProxyService) Ping(ctx context.Context) error {
	return nil
}

func (b *stackqlMCPReverseProxyService) Close() error {
	return nil
}

// Server and environment info
func (b *stackqlMCPReverseProxyService) ServerInfo(ctx context.Context, args any) (mcp_server.ServerInfoOutput, error) {
	return mcp_server.ServerInfoOutput{
		Name:       "Stackql MCP Reverse Proxy Service",
		Info:       "This is the Stackql MCP Reverse Proxy Service.",
		IsReadOnly: b.isReadOnly,
	}, nil
}

// Current DB identity details
func (b *stackqlMCPReverseProxyService) DBIdentity(ctx context.Context, args any) (map[string]any, error) {
	return map[string]any{
		"identity": "stackql_mcp_reverse_proxy_service",
	}, nil
}

func (b *stackqlMCPReverseProxyService) Greet(ctx context.Context, args mcp_server.GreetInput) (string, error) {
	return "Hi " + args.Name, nil
}

//nolint:gocognit,funlen // acceptable
func (b *stackqlMCPReverseProxyService) query(ctx context.Context, query string, rowLimit int) ([]map[string]any, error) {
	r, sqlErr := b.db.Query(query)
	if sqlErr != nil {
		return nil, sqlErr
	}
	rowsErr := r.Err()
	if rowsErr != nil {
		return nil, rowsErr
	}
	defer r.Close() //nolint:errcheck // TODO: investigate
	columnTypes, err := r.ColumnTypes()
	if err != nil {
		return nil, err
	}

	count := len(columnTypes)
	var finalRows []map[string]any

	rowCount := 0
	for r.Next() {
		if rowLimit > 0 && rowCount >= rowLimit {
			break
		}
		rowCount++
		scanArgs := make([]interface{}, count)

		for i, v := range columnTypes {
			switch v.DatabaseTypeName() {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
				break
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
				break
			case "INT4":
				scanArgs[i] = new(sql.NullInt64)
				break
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		scanErr := r.Scan(scanArgs...)

		if scanErr != nil {
			return nil, scanErr
		}

		masterData := map[string]any{}

		for i, v := range columnTypes {
			if z, ok := (scanArgs[i]).(*sql.NullBool); ok {
				masterData[v.Name()] = z.Bool
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				masterData[v.Name()] = z.String
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
				masterData[v.Name()] = z.Int64
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				masterData[v.Name()] = z.Float64
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullInt32); ok {
				masterData[v.Name()] = z.Int32
				continue
			}
			masterData[v.Name()] = scanArgs[i]
		}
		finalRows = append(finalRows, masterData)
	}
	return finalRows, nil
}

func (b *stackqlMCPReverseProxyService) renderQueryResults(query string, format string, rowLimit int) (string, error) {
	if format == "" {
		format = b.getDefaultFormat()
	}
	ctx := context.Background()
	rows, err := b.query(ctx, query, rowLimit)
	if err != nil {
		return "", err
	}
	switch format {
	case resultsFormatMarkdown:
		return b.renderer.RenderAsMarkdown(rows), nil
	case resultsFormatJSON:
		jsonStr, jsonErr := json.Marshal(rows)
		if jsonErr != nil {
			return "", jsonErr
		}
		return string(jsonStr), nil
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

func (b *stackqlMCPReverseProxyService) RunQuery(ctx context.Context, args mcp_server.QueryInput) (string, error) {
	if args.Format == "" {
		args.Format = b.getDefaultFormat()
	}
	rows, err := b.query(ctx, args.SQL, args.RowLimit)
	if err != nil {
		return "", err
	}
	switch args.Format {
	case resultsFormatMarkdown:
		return b.renderer.RenderAsMarkdown(rows), nil
	case resultsFormatJSON:
		jsonStr, jsonErr := json.Marshal(rows)
		if jsonErr != nil {
			return "", jsonErr
		}
		return string(jsonStr), nil
	default:
		return "", fmt.Errorf("unknown format: %s", args.Format)
	}
}

func (b *stackqlMCPReverseProxyService) RunQueryJSON(ctx context.Context, input mcp_server.QueryJSONInput) ([]map[string]interface{}, error) {
	return b.query(ctx, input.SQL, input.RowLimit)
}

// func (b *stackqlMCPReverseProxyService) ListTableResources(ctx context.Context, hI mcp_server.HierarchyInput) ([]string, error) {

// TODO: implement the remaining methods, using the db connection as sole sql data source

// 	return []string{}, nil
// }

func (b *stackqlMCPReverseProxyService) ReadTableResource(ctx context.Context, hI mcp_server.HierarchyInput) ([]map[string]interface{}, error) {
	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" {
		return nil, fmt.Errorf("provider, service, and resource must be specified")
	}
	query := fmt.Sprintf("SELECT * FROM %s.%s", hI.Service, hI.Resource)
	return b.query(ctx, query, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) PromptWriteSafeSelectTool(ctx context.Context, args mcp_server.HierarchyInput) (string, error) {
	return mcp_server.ExplainerPromptWriteSafeSelectTool, nil
}

// func (b *stackqlMCPReverseProxyService) PromptExplainPlanTipsTool(ctx context.Context) (string, error) {
// 	return "stub", nil
// }

func (b *stackqlMCPReverseProxyService) DescribeTable(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetDescribeTable(hI)
	if qErr != nil {
		return "", qErr
	}
	return b.renderQueryResults(q, hI.Format, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) GetForeignKeys(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	return b.interrogator.GetForeignKeys(hI)
}

func (b *stackqlMCPReverseProxyService) FindRelationships(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	return b.interrogator.FindRelationships(hI)
}

func (b *stackqlMCPReverseProxyService) ListProviders(ctx context.Context) (string, error) {
	q, qErr := b.interrogator.GetShowProviders(mcp_server.HierarchyInput{}, "")
	if qErr != nil {
		return "", qErr
	}
	return b.renderQueryResults(q, "", unlimitedRowLimit)
}

func (b *stackqlMCPReverseProxyService) ListServices(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetShowServices(hI, "")
	if qErr != nil {
		return "", qErr
	}
	return b.renderQueryResults(q, hI.Format, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) ListResources(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetShowResources(hI, "")
	if qErr != nil {
		return "", qErr
	}
	return b.renderQueryResults(q, hI.Format, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) ListTablesJSON(ctx context.Context, input mcp_server.ListTablesInput) ([]map[string]interface{}, error) {
	hI := mcp_server.HierarchyInput{}
	likeStr := ""
	if input.Hierarchy != nil {
		hI = *input.Hierarchy
	}
	if input.NameLike != nil {
		likeStr = *input.NameLike
	}
	q, qErr := b.interrogator.GetShowResources(hI, likeStr)
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, input.RowLimit)
}

func (b *stackqlMCPReverseProxyService) ListTablesJSONPage(ctx context.Context, input mcp_server.ListTablesPageInput) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (b *stackqlMCPReverseProxyService) ListTables(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	return b.ListResources(ctx, hI)
}

func (b *stackqlMCPReverseProxyService) ListMethods(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetShowMethods(hI)
	if qErr != nil {
		return "", qErr
	}
	return b.renderQueryResults(q, hI.Format, hI.RowLimit)
}
