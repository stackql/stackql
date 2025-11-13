package mcpbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/acid/tsm_physio"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/pkg/mcp_server"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
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

type resultsRenderer interface {
	RenderAsMarkdown(results []map[string]interface{}) string
}

func NewResultsRenderer() resultsRenderer {
	return &simpleRenderer{}
}

type simpleRenderer struct{}

func (r *simpleRenderer) RenderAsMarkdown(results []map[string]interface{}) string {
	if len(results) == 0 {
		return "**no results**"
	}
	var sb strings.Builder
	headerRow := presentation.NewMarkdownRowFromMap(results[0])
	sb.WriteString(headerRow.HeaderString() + "\n")
	sb.WriteString(headerRow.SeparatorString() + "\n")
	for _, row := range results {
		markdownRow := presentation.NewMarkdownRowFromMap(row)
		sb.WriteString(markdownRow.RowString() + "\n")
	}
	return sb.String()
}

type StackqlInterrogator interface {
	// This struct is responsible for interrogating the StackQL engine.
	// Each method provides the requisite query string.

	GetShowProviders(dto.HierarchyInput, string) (string, error)
	GetShowServices(dto.HierarchyInput, string) (string, error)
	GetShowResources(dto.HierarchyInput, string) (string, error)
	GetShowMethods(dto.HierarchyInput) (string, error)
	// GetShowTables(dto.HierarchyInput) (string, error)
	GetDescribeTable(dto.HierarchyInput) (string, error)
	GetForeignKeys(dto.HierarchyInput) (string, error)
	FindRelationships(dto.HierarchyInput) (string, error)
	GetQuery(dto.QueryInput) (string, error)
	GetQueryJSON(dto.QueryJSONInput) (string, error)
	// GetListTableResources(dto.HierarchyInput) (string, error)
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

func (s *simpleStackqlInterrogator) GetShowProviders(_ dto.HierarchyInput, likeStr string) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("SHOW PROVIDERS")
	if likeStr != "" {
		sb.WriteString(" LIKE '")
		sb.WriteString(likeStr)
		sb.WriteString("'")
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetShowServices(hI dto.HierarchyInput, likeStr string) (string, error) {
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

func (s *simpleStackqlInterrogator) GetShowResources(hI dto.HierarchyInput, likeString string) (string, error) {
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

func (s *simpleStackqlInterrogator) GetShowMethods(hI dto.HierarchyInput) (string, error) {
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

func (s *simpleStackqlInterrogator) GetDescribeTable(hI dto.HierarchyInput) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("DESCRIBE ")
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

func (s *simpleStackqlInterrogator) GetForeignKeys(hI dto.HierarchyInput) (string, error) {
	return mcp_server.ExplainerForeignKeyStackql, nil
}

func (s *simpleStackqlInterrogator) FindRelationships(hI dto.HierarchyInput) (string, error) {
	return mcp_server.ExplainerFindRelationships, nil
}

func (s *simpleStackqlInterrogator) GetQuery(qI dto.QueryInput) (string, error) {
	if qI.SQL == "" {
		return "", fmt.Errorf("no SQL provided")
	}
	return qI.SQL, nil
}

func (s *simpleStackqlInterrogator) GetQueryJSON(qI dto.QueryJSONInput) (string, error) {
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
	renderer        resultsRenderer
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
		renderer:        NewResultsRenderer(),
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
func (b *stackqlMCPService) ServerInfo(ctx context.Context, args any) (dto.ServerInfoOutput, error) {
	return dto.ServerInfoOutput{
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

func (b *stackqlMCPService) Greet(ctx context.Context, args dto.GreetInput) (string, error) {
	return "Hi " + args.Name, nil
}

func (b *stackqlMCPService) RunQuery(ctx context.Context, args dto.QueryInput) (string, error) {
	q, qErr := b.interrogator.GetQuery(args)
	if qErr != nil {
		return "", qErr
	}
	rv := b.renderQueryResults(q, args.Format, args.RowLimit)
	return rv, nil
}

func (b *stackqlMCPService) RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error) {
	q := input.SQL
	if q == "" {
		return nil, fmt.Errorf("no SQL provided")
	}
	return b.runPreprocessedQueryJSON(ctx, q, input.RowLimit)
}

func (b *stackqlMCPService) runPreprocessedQueryJSON(ctx context.Context, query string, rowLimit int) ([]map[string]interface{}, error) {
	results, ok := b.extractQueryResults(query, rowLimit)
	if !ok {
		return nil, fmt.Errorf("failed to extract query results")
	}
	return results, nil
}

func (b *stackqlMCPService) ExecQuery(ctx context.Context, query string) (map[string]any, error) {
	return b.execQuery(ctx, query)
}

func (b *stackqlMCPService) ValidateQuery(ctx context.Context, query string) ([]map[string]any, error) {
	explainQuery := fmt.Sprintf("EXPLAIN %s", query)
	rows, err := b.runPreprocessedQueryJSON(ctx, explainQuery, unlimitedRowLimit)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (b *stackqlMCPService) execQuery(ctx context.Context, query string) (map[string]any, error) {
	rv := map[string]any{}
	r, ok := b.applyQuery(query)
	if !ok {
		return rv, fmt.Errorf("failed to extract query results")
	}
	messages := []string{}
	for _, resp := range r {
		messages = append(messages, resp.GetMessages()...)
	}
	rv["messages"] = messages
	oneLinerOutput := time.Now().Format("2006-01-02T15:04:05-07:00 MST")
	rv["timestamp"] = oneLinerOutput
	return rv, nil
}

// func (b *stackqlMCPService) ListTableResources(ctx context.Context, hI mcp_server.HierarchyInput) ([]string, error) {
// 	return []string{}, nil
// }

// func (b *stackqlMCPService) ReadTableResource(ctx context.Context, hI mcp_server.HierarchyInput) ([]map[string]interface{}, error) {
// 	return []map[string]interface{}{}, nil
// }

func (b *stackqlMCPService) PromptWriteSafeSelectTool(ctx context.Context, args dto.HierarchyInput) (string, error) {
	return b.interrogator.GetPromptWriteSafeSelectTool()
}

// func (b *stackqlMCPService) PromptExplainPlanTipsTool(ctx context.Context) (string, error) {
// 	return "stub", nil
// }

func (b *stackqlMCPService) ListTablesJSON(ctx context.Context, input dto.ListTablesInput) ([]map[string]interface{}, error) {
	hI := dto.HierarchyInput{}
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

func (b *stackqlMCPService) ListTablesJSONPage(ctx context.Context, input dto.ListTablesPageInput) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (b *stackqlMCPService) ListTables(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	return b.ListResources(ctx, hI)
}

func (b *stackqlMCPService) ListMethods(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowMethods(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
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
	return b.renderer.RenderAsMarkdown(results)
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

func (b *stackqlMCPService) DescribeTable(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetDescribeTable(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPService) GetForeignKeys(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("GetForeignKeys not implemented")
}

func (b *stackqlMCPService) FindRelationships(ctx context.Context, hI dto.HierarchyInput) (string, error) {
	return b.interrogator.FindRelationships(hI)
}

func (b *stackqlMCPService) ListProviders(ctx context.Context) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowProviders(dto.HierarchyInput{}, "")
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPService) ListServices(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowServices(hI, "")
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPService) ListResources(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowResources(hI, "")
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}
