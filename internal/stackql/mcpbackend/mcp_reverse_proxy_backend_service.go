package mcpbackend

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/pkg/mcp_server"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

type stackqlMCPReverseProxyService struct {
	isReadOnly   bool
	dsn          string
	handlerCtx   handler.HandlerContext
	logger       *logrus.Logger
	db           *sql.DB
	interrogator StackqlInterrogator
	serverInfo   serverBuildInfo
}

func NewStackqlMCPReverseProxyService(
	isReadOnly bool,
	dsn string,
	db *sql.DB,
	handlerCtx handler.HandlerContext,
	logger *logrus.Logger,
	serverInfo serverBuildInfo,
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
		serverInfo:   serverInfo,
	}, nil
}

func (b *stackqlMCPReverseProxyService) Ping(_ context.Context) error {
	return nil
}

func (b *stackqlMCPReverseProxyService) Close() error {
	return nil
}

func (b *stackqlMCPReverseProxyService) ServerInfo(_ context.Context, _ any) (dto.ServerInfoOutput, error) {
	return dto.ServerInfoOutput{
		Version:          b.serverInfo.version(),
		Commit:           b.serverInfo.commit(),
		BuildDate:        b.serverInfo.buildDate(),
		Platform:         b.serverInfo.platform(),
		Transport:        b.serverInfo.transport(),
		SQLBackend:       b.serverInfo.sqlBackend(),
		ProviderRegistry: b.serverInfo.providerRegistry(),
		ReadOnly:         b.isReadOnly,
	}, nil
}

func (b *stackqlMCPReverseProxyService) ExecQuery(_ context.Context, query string) (map[string]any, error) {
	r, sqlErr := b.db.Exec(query)
	if sqlErr != nil {
		return nil, sqlErr
	}
	rowsAffected, rowsAffectedErr := r.RowsAffected()
	lastInsertID, lastInsertIDErr := r.LastInsertId()
	rv := map[string]any{}
	if rowsAffectedErr == nil {
		rv["rows_affected"] = rowsAffected
	}
	if lastInsertIDErr == nil {
		rv["last_insert_id"] = lastInsertID
	}
	rv["timestamp"] = nowTimestamp()
	return rv, nil
}

func (b *stackqlMCPReverseProxyService) ValidateQuery(ctx context.Context, query string) ([]map[string]any, error) {
	explainQuery := fmt.Sprintf("EXPLAIN %s", query)
	return b.query(ctx, explainQuery, unlimitedRowLimit)
}

//nolint:gocognit,funlen // acceptable
func (b *stackqlMCPReverseProxyService) query(_ context.Context, query string, rowLimit int) ([]map[string]any, error) {
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
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT4":
				scanArgs[i] = new(sql.NullInt64)
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

func (b *stackqlMCPReverseProxyService) RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error) {
	return b.query(ctx, input.SQL, input.RowLimit)
}

func (b *stackqlMCPReverseProxyService) DescribeResource(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetDescribeResource(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) DescribeMethod(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetDescribeMethod(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPReverseProxyService) ListProviders(ctx context.Context) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowProviders(dto.HierarchyInput{}, "")
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPReverseProxyService) ListServices(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowServices(hI, "")
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) ListResources(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowResources(hI, "")
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, hI.RowLimit)
}

func (b *stackqlMCPReverseProxyService) ListMethods(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowMethods(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.query(ctx, q, hI.RowLimit)
}
