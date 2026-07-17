package mcpbackend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/acid/tsm_physio"
	"github.com/stackql/stackql/internal/stackql/buildinfo"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/pkg/mcp_server"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

var (
	_ mcp_server.Backend = mcp_server.Backend(nil)
)

const (
	unlimitedRowLimit int = -1
	// forbiddenRegistryCharacters mirrors the CLI registry command's guard
	// (see internal/stackql/cmd/registry.go).  Interrogator methods that
	// interpolate user-supplied registry identifiers reject these characters
	// rather than substituting / escaping them, matching CLI semantics.
	forbiddenRegistryCharacters string = ` ;\`
)

// serverBuildInfo carries the runtime + build-time metadata reported by the
// server_info MCP tool. The interface and concrete struct are both unexported;
// callers construct via NewServerBuildInfo and pass the value through.
type serverBuildInfo interface {
	version() string
	commit() string
	buildDate() string
	platform() string
	transport() string
	sqlBackend() string
	providerRegistry() string
	mode() string
}

type immutableServerBuildInfo struct {
	versionStr          string
	commitStr           string
	buildDateStr        string
	platformStr         string
	transportStr        string
	sqlBackendStr       string
	providerRegistryStr string
	modeStr             string
}

func (s *immutableServerBuildInfo) version() string          { return s.versionStr }
func (s *immutableServerBuildInfo) commit() string           { return s.commitStr }
func (s *immutableServerBuildInfo) buildDate() string        { return s.buildDateStr }
func (s *immutableServerBuildInfo) platform() string         { return s.platformStr }
func (s *immutableServerBuildInfo) transport() string        { return s.transportStr }
func (s *immutableServerBuildInfo) sqlBackend() string       { return s.sqlBackendStr }
func (s *immutableServerBuildInfo) providerRegistry() string { return s.providerRegistryStr }
func (s *immutableServerBuildInfo) mode() string             { return s.modeStr }

// NewServerBuildInfo composes build-time identifiers and runtime values into a
// single value carried by the backend. Fields are written exactly once here.
// The returned interface is unexported; callers pass it straight into the
// backend constructors and never read individual fields outside this package.
//
// `mode` is the server's mode string (read_only, safe, delete_safe, full_access).
// The back-compat `is_read_only` flag on server_info is derived by ServerInfo()
// from this value.
func NewServerBuildInfo(
	bi buildinfo.BuildInfo,
	transport, sqlBackend, providerRegistry, mode string,
) serverBuildInfo { //nolint:revive // intentional: unexported return for data-carrier rule
	return &immutableServerBuildInfo{
		versionStr:          bi.GetSemVersion(),
		commitStr:           bi.GetShortCommitSHA(),
		buildDateStr:        bi.GetDate(),
		platformStr:         bi.GetPlatform(),
		transportStr:        transport,
		sqlBackendStr:       sqlBackend,
		providerRegistryStr: providerRegistry,
		modeStr:             mode,
	}
}

// StackqlInterrogator builds StackQL SQL strings for the metadata tools.
type StackqlInterrogator interface {
	GetShowProviders(dto.HierarchyInput, string) (string, error)
	GetShowServices(dto.HierarchyInput, string) (string, error)
	GetShowResources(dto.HierarchyInput, string) (string, error)
	GetShowMethods(dto.HierarchyInput) (string, error)
	GetDescribeResource(dto.HierarchyInput) (string, error)
	GetDescribeMethod(dto.HierarchyInput) (string, error)
	GetQueryJSON(dto.QueryJSONInput) (string, error)
	GetRegistryList(provider string) (string, error)
	GetRegistryPull(provider, version string) (string, error)
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
	if hI.Provider == "" {
		return "", fmt.Errorf("provider not specified")
	}
	sb := strings.Builder{}
	sb.WriteString("SHOW SERVICES IN ")
	sb.WriteString(hI.Provider)
	if likeStr != "" {
		sb.WriteString(" LIKE '")
		sb.WriteString(likeStr)
		sb.WriteString("'")
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetShowResources(hI dto.HierarchyInput, likeString string) (string, error) {
	if hI.Provider == "" || hI.Service == "" {
		return "", fmt.Errorf("provider and / or service not specified")
	}
	sb := strings.Builder{}
	sb.WriteString("SHOW RESOURCES IN ")
	sb.WriteString(hI.Provider)
	sb.WriteString(".")
	sb.WriteString(hI.Service)
	if likeString != "" {
		sb.WriteString(" LIKE '")
		sb.WriteString(likeString)
		sb.WriteString("'")
	}
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetShowMethods(hI dto.HierarchyInput) (string, error) {
	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" {
		return "", fmt.Errorf("provider, service and / or resource not specified")
	}
	sb := strings.Builder{}
	sb.WriteString("SHOW METHODS IN ")
	sb.WriteString(hI.Provider)
	sb.WriteString(".")
	sb.WriteString(hI.Service)
	sb.WriteString(".")
	sb.WriteString(hI.Resource)
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetDescribeResource(hI dto.HierarchyInput) (string, error) {
	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" {
		return "", fmt.Errorf("provider, service and / or resource not specified")
	}
	sb := strings.Builder{}
	sb.WriteString("DESCRIBE ")
	sb.WriteString(hI.Provider)
	sb.WriteString(".")
	sb.WriteString(hI.Service)
	sb.WriteString(".")
	sb.WriteString(hI.Resource)
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetDescribeMethod(hI dto.HierarchyInput) (string, error) {
	if hI.Provider == "" || hI.Service == "" || hI.Resource == "" || hI.Method == "" {
		return "", fmt.Errorf("provider, service, resource and / or method not specified")
	}
	sb := strings.Builder{}
	sb.WriteString("DESCRIBE METHOD EXTENDED ")
	sb.WriteString(hI.Provider)
	sb.WriteString(".")
	sb.WriteString(hI.Service)
	sb.WriteString(".")
	sb.WriteString(hI.Resource)
	sb.WriteString(".")
	sb.WriteString(hI.Method)
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetQueryJSON(qI dto.QueryJSONInput) (string, error) {
	if qI.SQL == "" {
		return "", fmt.Errorf("no SQL provided")
	}
	return qI.SQL, nil
}

func (s *simpleStackqlInterrogator) GetRegistryList(provider string) (string, error) {
	if provider != "" && strings.ContainsAny(provider, forbiddenRegistryCharacters) {
		return "", fmt.Errorf("forbidden characters in provider")
	}
	sb := strings.Builder{}
	sb.WriteString("REGISTRY LIST")
	if provider != "" {
		sb.WriteString(" ")
		sb.WriteString(provider)
	}
	sb.WriteString(";")
	return sb.String(), nil
}

func (s *simpleStackqlInterrogator) GetRegistryPull(provider, version string) (string, error) {
	if provider == "" {
		return "", fmt.Errorf("provider not specified")
	}
	if strings.ContainsAny(provider, forbiddenRegistryCharacters) ||
		strings.ContainsAny(version, forbiddenRegistryCharacters) {
		return "", fmt.Errorf("forbidden characters in provider or version")
	}
	sb := strings.Builder{}
	sb.WriteString("REGISTRY PULL ")
	sb.WriteString(provider)
	if version != "" {
		sb.WriteString(" ")
		sb.WriteString(version)
	}
	sb.WriteString(";")
	return sb.String(), nil
}

type stackqlMCPService struct {
	txnOrchestrator tsm_physio.Orchestrator
	interrogator    StackqlInterrogator
	handlerCtx      handler.HandlerContext
	logger          *logrus.Logger
	serverInfo      serverBuildInfo
	// envFile is the dotenv-style file (mcp.config server.env_file) that the
	// reload_credentials tool (re)sources into the process environment
	// (issue #688).  Empty means no mutable credential source is configured.
	envFile string
}

func NewStackqlMCPBackendService(
	txnOrchestrator tsm_physio.Orchestrator,
	handlerCtx handler.HandlerContext,
	logger *logrus.Logger,
	serverInfo serverBuildInfo,
	envFile string,
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
	// Source the env file once at startup so credentials already present on
	// disk work without a reload_credentials call.  A missing file is fine
	// (it may be written mid-session); a malformed path is surfaced loudly.
	if _, _, sourceErr := sourceEnvFile(envFile); sourceErr != nil {
		return nil, fmt.Errorf("failed to source env file '%s': %w", envFile, sourceErr)
	}
	return &stackqlMCPService{
		txnOrchestrator: txnOrchestrator,
		interrogator:    NewSimpleStackqlInterrogator(),
		logger:          logger,
		handlerCtx:      handlerCtx,
		serverInfo:      serverInfo,
		envFile:         envFile,
	}, nil
}

// modeReadOnly is the on-wire mode string equivalent to the historical
// `read_only: true` flag.  Duplicated here (rather than importing
// pkg/mcp_server/policy) to keep this internal package free of cross-package
// dependencies on the public MCP surface.
const modeReadOnly = "read_only"

func (b *stackqlMCPService) Ping(_ context.Context) error {
	return nil
}

func (b *stackqlMCPService) Close() error {
	return nil
}

func (b *stackqlMCPService) ServerInfo(_ context.Context, _ any) (dto.ServerInfoOutput, error) {
	mode := b.serverInfo.mode()
	return dto.ServerInfoOutput{
		Version:          b.serverInfo.version(),
		Commit:           b.serverInfo.commit(),
		BuildDate:        b.serverInfo.buildDate(),
		Platform:         b.serverInfo.platform(),
		Transport:        b.serverInfo.transport(),
		SQLBackend:       b.serverInfo.sqlBackend(),
		ProviderRegistry: b.serverInfo.providerRegistry(),
		Mode:             mode,
		ReadOnly:         mode == modeReadOnly,
	}, nil
}

func (b *stackqlMCPService) RunQueryJSON(ctx context.Context, input dto.QueryJSONInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetQueryJSON(input)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, input.RowLimit)
}

func (b *stackqlMCPService) runPreprocessedQueryJSON(_ context.Context, query string, rowLimit int) ([]map[string]interface{}, error) {
	results, extractErr := b.extractQueryResults(query, rowLimit)
	if extractErr != nil {
		return nil, extractErr
	}
	return results, nil
}

// upstreamStatusCodeRegex matches the status code fragments any-sdk embeds in
// upstream response error strings, eg "HTTP response error.  Status code 403.",
// "Response error.  Status code 429." and "http response status code: 503".
var upstreamStatusCodeRegex = regexp.MustCompile(`(?i)status code:?\s*(\d{3})`) //nolint:gochecknoglobals // compiled once

// isRetryableHTTPStatus classifies upstream statuses for MCP clients:
// 408 / 429 / 5xx are worth retrying, other 4xx are not (issue #670).
func isRetryableHTTPStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

// isCredentialResolutionError matches the error shapes any-sdk emits when
// credential material cannot be resolved from the process environment or a
// key file (eg "credentials error: credentialsenvvar references empty
// string").  These fail before any HTTP request is made, so they carry no
// upstream status code.
func isCredentialResolutionError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "credentials error") ||
		strings.Contains(msg, "references empty string")
}

// classifyBackendError decorates a statement error for the MCP surface.
// When the error carries an upstream HTTP status, the returned error text is
// prefixed with a machine-readable classification ({"http_status": ...,
// "retryable": ...}); credential resolution failures get an agent-actionable
// hint pointing at the reload_credentials tool (issue #688); anything else
// keeps the historical "failed to extract query results" prefix with the
// underlying detail appended (issue #670).
func classifyBackendError(err error) error {
	match := upstreamStatusCodeRegex.FindStringSubmatch(err.Error())
	if match == nil {
		if isCredentialResolutionError(err) {
			return fmt.Errorf(
				"credential resolution failed (hint: update credentials at their source, "+
					"then call the reload_credentials tool and retry): %w",
				err,
			)
		}
		return fmt.Errorf("failed to extract query results: %w", err)
	}
	status, convErr := strconv.Atoi(match[1])
	if convErr != nil {
		return fmt.Errorf("failed to extract query results: %w", err)
	}
	return fmt.Errorf(
		`upstream http error: {"http_status": %d, "retryable": %t}: %w`,
		status,
		isRetryableHTTPStatus(status),
		err,
	)
}

// ExecQuery returns {messages, timestamp}: messages is the orchestrator's
// per-statement message list (eg "The operation was despatched successfully"),
// timestamp is the wall-clock dispatch time.  The reverse-proxy backend
// returns a different shape ({timestamp, rows_affected?, last_insert_id?})
// because it goes through database/sql Exec instead of the orchestrator.
// Robot assertions that target both backends must rely only on `timestamp`.
func (b *stackqlMCPService) ExecQuery(_ context.Context, query string) (map[string]any, error) {
	return b.execQuery(query)
}

func (b *stackqlMCPService) ValidateQuery(ctx context.Context, query string) ([]map[string]any, error) {
	explainQuery := fmt.Sprintf("EXPLAIN %s", query)
	return b.runPreprocessedQueryJSON(ctx, explainQuery, unlimitedRowLimit)
}

func (b *stackqlMCPService) execQuery(query string) (map[string]any, error) {
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
	rv["timestamp"] = nowTimestamp()
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

func (b *stackqlMCPService) extractQueryResults(query string, rowLimit int) ([]map[string]interface{}, error) {
	r, ok := b.applyQuery(query)
	if !ok {
		return nil, fmt.Errorf("failed to extract query results")
	}
	// Initialise as empty (not nil) so a zero-row result survives downstream
	// JSON-array schema validation on QueryResultDTO.Rows.  This pairs with
	// fix 1 (returning ok regardless of len(rv)) so empty results render as
	// "**no results**" rather than failing extraction.
	rv := []map[string]interface{}{}
	rowCount := 0
	for _, resp := range r {
		if respErr := resp.GetError(); respErr != nil {
			// Propagate the statement error (with upstream HTTP status
			// classification where available) rather than collapsing it
			// into an indistinct empty result (issue #670).
			return nil, classifyBackendError(respErr)
		}
		// PrepareResultSet emits a nil SQLResult when RowMap is empty (eg
		// REGISTRY LIST against an empty registry).  That's a zero-row
		// result, not an extraction failure: skip the stream and continue.
		sqlRowStream := resp.GetSQLResult()
		if sqlRowStream == nil {
			continue
		}
		var drainOK bool
		rv, rowCount, drainOK = drainSQLRowStream(sqlRowStream, rv, rowCount, rowLimit)
		if !drainOK {
			return nil, fmt.Errorf("failed to extract query results")
		}
	}
	return rv, nil
}

// drainSQLRowStream reads `stream` to EOF (or until rowLimit is reached),
// appending each row's payload to `rv`.  The returned bool is false when the
// stream surfaces a read error or a nil row outside of EOF; that maps onto
// extractQueryResults' error failure mode.
func drainSQLRowStream(
	stream sqldata.ISQLResultStream,
	rv []map[string]interface{},
	rowCount, rowLimit int,
) ([]map[string]interface{}, int, bool) {
	for {
		row, err := stream.Read()
		if err == io.EOF {
			if row != nil {
				rv = append(rv, row.ToArr()...)
			}
			return rv, rowCount, true
		}
		if err != nil || row == nil {
			return rv, rowCount, false
		}
		rowArr := row.ToArr()
		rv = append(rv, rowArr...)
		rowCount += len(rowArr)
		if rowLimit > 0 && rowCount >= rowLimit {
			return rv, rowCount, true
		}
	}
}

func (b *stackqlMCPService) DescribeResource(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetDescribeResource(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPService) DescribeMethod(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetDescribeMethod(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
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

func (b *stackqlMCPService) ListMethods(ctx context.Context, hI dto.HierarchyInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetShowMethods(hI)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPService) ListRegistry(ctx context.Context, input dto.RegistryInput) ([]map[string]interface{}, error) {
	q, qErr := b.interrogator.GetRegistryList(input.Provider)
	if qErr != nil {
		return nil, qErr
	}
	return b.runPreprocessedQueryJSON(ctx, q, unlimitedRowLimit)
}

func (b *stackqlMCPService) PullProvider(ctx context.Context, input dto.RegistryInput) (map[string]any, error) {
	q, qErr := b.interrogator.GetRegistryPull(input.Provider, input.Version)
	if qErr != nil {
		return nil, qErr
	}
	return b.ExecQuery(ctx, q)
}
