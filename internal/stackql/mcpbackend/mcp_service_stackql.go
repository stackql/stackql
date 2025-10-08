package mcpbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/acid/tsm_physio"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/pkg/mcp_server"
	"github.com/stackql/stackql/pkg/presentation"
)

var (
	_ mcp_server.Backend = mcp_server.Backend(nil)
)

const (
	resultsFormatMarkdown     = "markdown"
	resultsFormatJSON         = "json"
	unlimitedRowLimit     int = -1
)

type StackqlInterrogator interface {
	// This struct is responsible for interrogating the StackQL engine.
	// Each method provides the requisite query string.

	GetShowProviders(mcp_server.HierarchyInput, string) (string, error)
	GetShowServices(mcp_server.HierarchyInput, string) (string, error)
	GetShowResources(mcp_server.HierarchyInput, string) (string, error)
	GetShowMethods(mcp_server.HierarchyInput) (string, error)
	// GetShowTables(mcp_server.HierarchyInput) (string, error)
	GetDescribeTable(mcp_server.HierarchyInput) (string, error)
	GetForeignKeys(mcp_server.HierarchyInput) (string, error)
	FindRelationships(mcp_server.HierarchyInput) (string, error)
	GetQuery(mcp_server.QueryInput) (string, error)
	GetQueryJSON(mcp_server.QueryJSONInput) (string, error)
	// GetListTableResources(mcp_server.HierarchyInput) (string, error)
	// GetReadTableResource(mcp_server.HierarchyInput) (string, error)
	GetPromptWriteSafeSelectTool() (string, error)
	// GetPromptExplainPlanTipsTool() (string, error)
	// GetListTablesJSON(mcp_server.ListTablesInput) (string, error)
	// GetListTablesJSONPage(mcp_server.ListTablesPageInput) (string, error)
}

type simpleStackqlInterrogator struct{}

func NewSimpleStackqlInterrogator() StackqlInterrogator {
	return &simpleStackqlInterrogator{}
}

func (s *simpleStackqlInterrogator) GetShowProviders(_ mcp_server.HierarchyInput, likeStr string) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("SHOW PROVIDERS")
	if likeStr != "" {
		sb.WriteString(" LIKE '")
		sb.WriteString(likeStr)
		sb.WriteString("'")
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetShowServices(hI mcp_server.HierarchyInput, likeStr string) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("SHOW SERVICES")
	if hI.Provider == "" {
		return "", fmt.Errorf("provider not specified")
	}
	sb.WriteString(" IN ")
	sb.WriteString(hI.Provider)
	if likeStr != "" {
		sb.WriteString(" LIKE '")
		sb.WriteString(likeStr)
		sb.WriteString("'")
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetShowResources(hI mcp_server.HierarchyInput, likeString string) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("SHOW RESOURCES")
	if hI.Provider == "" || hI.Service == "" {
		return "", fmt.Errorf("provider and / or service not specified")
	}
	sb.WriteString(" IN ")
	sb.WriteString(hI.Provider)
	if hI.Service != "" {
		sb.WriteString(".")
		sb.WriteString(hI.Service)
	}
	if likeString != "" {
		sb.WriteString(" LIKE '")
		sb.WriteString(likeString)
		sb.WriteString("'")
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetShowMethods(hI mcp_server.HierarchyInput) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("SHOW METHODS")
	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" {
		return "", fmt.Errorf("provider, service and / or resource not specified")
	}
	sb.WriteString(" IN ")
	sb.WriteString(hI.Provider)
	if hI.Service != "" {
		sb.WriteString(".")
		sb.WriteString(hI.Service)
	}
	if hI.Resource != "" {
		sb.WriteString(".")
		sb.WriteString(hI.Resource)
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetDescribeTable(hI mcp_server.HierarchyInput) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("DESCRIBE TABLE")
	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" {
		return "", fmt.Errorf("provider, service and / or resource not specified")
	}
	sb.WriteString(" ")
	sb.WriteString(hI.Provider)
	if hI.Service != "" {
		sb.WriteString(".")
		sb.WriteString(hI.Service)
	}
	if hI.Resource != "" {
		sb.WriteString(".")
		sb.WriteString(hI.Resource)
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetForeignKeys(hI mcp_server.HierarchyInput) (string, error) {
	return mcp_server.ExplainerForeignKeyStackql, nil
}

func (s *simpleStackqlInterrogator) FindRelationships(hI mcp_server.HierarchyInput) (string, error) {
	return mcp_server.ExplainerFindRelationships, nil
}

func (s *simpleStackqlInterrogator) GetQuery(qI mcp_server.QueryInput) (string, error) {
	if qI.SQL == "" {
		return "", fmt.Errorf("no SQL provided")
	}
	return qI.SQL, nil
}

func (s *simpleStackqlInterrogator) GetQueryJSON(qI mcp_server.QueryJSONInput) (string, error) {
	if qI.SQL == "" {
		return "", fmt.Errorf("no SQL provided")
	}
	return qI.SQL, nil
}

func (s *simpleStackqlInterrogator) GetPromptWriteSafeSelectTool() (string, error) {
	return mcp_server.ExplainerPromptWriteSafeSelectTool, nil
}

// func (s *simpleStackqlInterrogator) composeWhereClause(params map[string]any) (string, error) {
// 	sb := strings.Builder{}
// 	sb.WriteString(" WHERE ")
// 	for key, value := range params {
// 		sb.WriteString(fmt.Sprintf("%s = '%v' AND ", key, value))
// 	}
// 	// Remove the trailing " AND "
// 	whereClause := strings.TrimSuffix(sb.String(), " AND ")
// 	return whereClause, nil
// }

// func (s *simpleStackqlInterrogator) GetReadTableResource(hI mcp_server.HierarchyInput) (string, error) {
// 	sb := strings.Builder{}
// 	sb.WriteString("SELECT * FROM")
// 	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" {
// 		return "", fmt.Errorf("provider, service and / or resource not specified")
// 	}
// 	sb.WriteString(" ")
// 	sb.WriteString(hI.Provider)
// 	if hI.Service != "" {
// 		sb.WriteString(".")
// 		sb.WriteString(hI.Service)
// 	}
// 	if hI.Resource != "" {
// 		sb.WriteString(".")
// 		sb.WriteString(hI.Resource)
// 	}
// 	if len(hI.Parameters) > 0 {
// 		whereClause, err := s.composeWhereClause(hI.Parameters)
// 		if err != nil {
// 			return "", err
// 		}
// 		sb.WriteString(" " + whereClause)
// 	}
// 	return sb.String(), nil
// }

type stackqlMCPService struct {
	isReadOnly      bool
	txnOrchestrator tsm_physio.Orchestrator
	interrogator    StackqlInterrogator
	handlerCtx      handler.HandlerContext
	logger          *logrus.Logger
}

func NewStackqlMCPBackendService(
	isReadOnly bool,
	txnOrchestrator tsm_physio.Orchestrator,
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
	if txnOrchestrator == nil {
		return nil, fmt.Errorf("transaction orchestrator is nil")
	}
	return &stackqlMCPService{
		isReadOnly:      isReadOnly,
		txnOrchestrator: txnOrchestrator,
		interrogator:    NewSimpleStackqlInterrogator(),
		logger:          logger,
		handlerCtx:      handlerCtx,
	}, nil
}

func (b *stackqlMCPService) getDefaultFormat() string {
	return resultsFormatMarkdown
}

func (b *stackqlMCPService) Ping(ctx context.Context) error {
	return nil
}

func (b *stackqlMCPService) Close() error {
	return nil
}

// Server and environment info
func (b *stackqlMCPService) ServerInfo(ctx context.Context, args any) (mcp_server.ServerInfoOutput, error) {
	return mcp_server.ServerInfoOutput{
		Name:       "Stackql MCP Service",
		Info:       "This is the Stackql MCP Service.",
		IsReadOnly: b.isReadOnly,
	}, nil
}

// Current DB identity details
func (b *stackqlMCPService) DBIdentity(ctx context.Context, args any) (map[string]any, error) {
	return map[string]any{
		"identity": "stackql_mcp_service",
	}, nil
}

func (b *stackqlMCPService) Greet(ctx context.Context, args mcp_server.GreetInput) (string, error) {
	return "Hi " + args.Name, nil
}

func (b *stackqlMCPService) RunQuery(ctx context.Context, args mcp_server.QueryInput) (string, error) {
	q, qErr := b.interrogator.GetQuery(args)
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, args.Format, args.RowLimit)
	return rv, nil
}

func (b *stackqlMCPService) RunQueryJSON(ctx context.Context, input mcp_server.QueryJSONInput) ([]map[string]interface{}, error) {
	q := input.SQL
	if q == "" {
		return nil, fmt.Errorf("no SQL provided")
	}
	results, ok := b.extractQueryResults(q, input.RowLimit)
	if !ok {
		return nil, fmt.Errorf("failed to extract query results")
	}
	return results, nil
}

// func (b *stackqlMCPService) ListTableResources(ctx context.Context, hI mcp_server.HierarchyInput) ([]string, error) {
// 	return []string{}, nil
// }

// func (b *stackqlMCPService) ReadTableResource(ctx context.Context, hI mcp_server.HierarchyInput) ([]map[string]interface{}, error) {
// 	return []map[string]interface{}{}, nil
// }

func (b *stackqlMCPService) PromptWriteSafeSelectTool(ctx context.Context, args mcp_server.HierarchyInput) (string, error) {
	return b.interrogator.GetPromptWriteSafeSelectTool()
}

// func (b *stackqlMCPService) PromptExplainPlanTipsTool(ctx context.Context) (string, error) {
// 	return "stub", nil
// }

func (b *stackqlMCPService) ListTablesJSON(ctx context.Context, input mcp_server.ListTablesInput) ([]map[string]interface{}, error) {
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
	results, ok := b.extractQueryResults(q, input.RowLimit)
	if !ok {
		return nil, fmt.Errorf("failed to extract query results")
	}
	return results, nil
}

func (b *stackqlMCPService) ListTablesJSONPage(ctx context.Context, input mcp_server.ListTablesPageInput) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (b *stackqlMCPService) ListTables(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	return b.ListResources(ctx, hI)
}

func (b *stackqlMCPService) ListMethods(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetShowMethods(hI)
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, hI.Format, hI.RowLimit)
	return rv, nil
}

func (b *stackqlMCPService) getUpdatedHandlerCtx(query string) (handler.HandlerContext, error) {
	clonedCtx := b.handlerCtx.Clone()
	clonedCtx.SetRawQuery(query)
	return clonedCtx, nil
}

func (b *stackqlMCPService) applyQuery(query string) ([]internaldto.ExecutorOutput, bool) {
	updatedCtx, ctxErr := b.getUpdatedHandlerCtx(query)
	if ctxErr != nil {
		return nil, false
	}
	r, ok := b.txnOrchestrator.ProcessQueryOrQueries(updatedCtx)
	return r, ok
}

func (b *stackqlMCPService) extractQueryResults(query string, rowLimit int) ([]map[string]interface{}, bool) {
	r, ok := b.applyQuery(query)
	var rv []map[string]interface{}
	rowCount := 0
	for _, resp := range r {
		sqlRowStream := resp.GetSQLResult()
		if sqlRowStream == nil {
			ok = false
			break
		}
		for {
			row, err := sqlRowStream.Read()
			if err == io.EOF {
				rowArr := row.ToArr()
				rv = append(rv, rowArr...)
				break
			}
			if err != nil || row == nil {
				ok = false
				break
			}
			rowArr := row.ToArr()
			rv = append(rv, rowArr...)
			rowCount += len(rowArr)
			if rowLimit > 0 && rowCount >= rowLimit {
				break
			}
		}
	}
	return rv, (ok && len(rv) > 0)
}

func (b *stackqlMCPService) renderQueryResultsAsMarkdown(results []map[string]interface{}) string {
	if len(results) == 0 {
		return "**no results**"
	}
	var sb strings.Builder
	headerRow := presentation.NewMarkdownRowFromMap(results[0])
	sb.WriteString(headerRow.HeaderString() + "\n")
	sb.WriteString(headerRow.SeparatorString() + "\n")
	for _, row := range results[1:] {
		markdownRow := presentation.NewMarkdownRowFromMap(row)
		sb.WriteString(markdownRow.RowString() + "\n")
	}
	return sb.String()
}

func (b *stackqlMCPService) renderQueryResultsAsJSON(results []map[string]interface{}) string {
	if len(results) == 0 {
		return `{"error": "**no results**"}`
	}
	jsonData, err := json.Marshal(results)
	if err != nil {
		return fmt.Sprintf(`{"error": "%v"}`, err)
	}
	return string(jsonData)
}

func (b *stackqlMCPService) renderQueryResults(query string, format string, rowLimit int) string {
	results, ok := b.extractQueryResults(query, rowLimit)
	if format == "" {
		format = b.getDefaultFormat()
	}
	switch format {
	case resultsFormatMarkdown:
		if !ok || len(results) == 0 {
			return "**no results**"
		}
		return b.renderQueryResultsAsMarkdown(results)
	case resultsFormatJSON:
		if !ok || len(results) == 0 {
			return `{"error": "**no results**"}`
		}
		return b.renderQueryResultsAsJSON(results)
	default:
		return fmt.Sprintf("unsupported format: %s", format)
	}
}

func (b *stackqlMCPService) DescribeTable(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetDescribeTable(hI)
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, hI.Format, hI.RowLimit)
	return rv, nil
}

func (b *stackqlMCPService) GetForeignKeys(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	return b.interrogator.GetForeignKeys(hI)
}

func (b *stackqlMCPService) FindRelationships(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	return b.interrogator.FindRelationships(hI)
}

func (b *stackqlMCPService) ListProviders(ctx context.Context) (string, error) {
	q, qErr := b.interrogator.GetShowProviders(mcp_server.HierarchyInput{}, "")
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, "", unlimitedRowLimit)
	return rv, nil
}

func (b *stackqlMCPService) ListServices(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetShowServices(hI, "")
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, hI.Format, hI.RowLimit)
	return rv, nil
}

func (b *stackqlMCPService) ListResources(ctx context.Context, hI mcp_server.HierarchyInput) (string, error) {
	q, qErr := b.interrogator.GetShowResources(hI, "")
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, hI.Format, hI.RowLimit)
	return rv, nil
}
