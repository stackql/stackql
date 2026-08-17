package intrinsic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq/oid"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const methodPredicate = "method"

const defaultBatchSize = 100

const defaultFlushInterval = 50 * time.Millisecond

func relationName(path string) string {
	return strings.ReplaceAll(path, ".", "_")
}

func requiredParamNames(method omnisdk.Method) []string {
	var names []string
	for _, param := range method.Params {
		if param.Required {
			names = append(names, param.Name)
		}
	}
	return names
}

func dataTables() ([]table, error) {
	resources, err := omnisdk.Default().Resources(".*")
	if err != nil {
		return nil, fmt.Errorf("intrinsic: cannot read omnisdk catalog: %w", err)
	}
	out := make([]table, 0, len(resources))
	for _, resource := range resources {
		out = append(out, dataTable(resource))
	}
	return out, nil
}

func dataTable(resource omnisdk.Resource) table {
	return table{
		service:     auditService,
		name:        relationName(resource.Path),
		description: resource.Summary,
		columns:     schemaColumns(resource.Schema),
	}
}

func schemaColumns(schema map[string]any) []column {
	required, ok := schema["required"].([]string)
	if !ok {
		return nil
	}
	properties, _ := schema["properties"].(map[string]any)
	cols := make([]column, 0, len(required))
	for _, name := range required {
		property, _ := properties[name].(map[string]any)
		cols = append(cols, column{name: name, dataType: schemaTypeOf(property)})
	}
	return cols
}

func lookupDataRelation(name string) (omnisdk.Resource, bool) {
	resources, err := omnisdk.Default().Resources(".*")
	if err != nil {
		return omnisdk.Resource{}, false
	}
	for _, resource := range resources {
		if strings.EqualFold(name, relationName(resource.Path)) {
			return resource, true
		}
	}
	return omnisdk.Resource{}, false
}

func pickMethod(resourcePath string, params map[string]string) (omnisdk.Method, error) {
	methods, err := omnisdk.Default().Methods(resourcePath)
	if err != nil {
		return omnisdk.Method{}, err
	}
	if explicit, ok := params[methodPredicate]; ok {
		for _, method := range methods {
			if strings.EqualFold(method.Path, explicit) || strings.EqualFold(lastSegment(method.Path), explicit) {
				return method, nil
			}
		}
		return omnisdk.Method{}, fmt.Errorf(
			"intrinsic: resource '%s' has no method '%s'", resourcePath, explicit)
	}
	var satisfied []omnisdk.Method
	for _, method := range methods {
		if satisfiedBy(method, params) {
			satisfied = append(satisfied, method)
		}
	}
	switch len(satisfied) {
	case 1:
		return satisfied[0], nil
	case 0:
		return omnisdk.Method{}, fmt.Errorf(
			"intrinsic: no method of '%s' has its required parameters supplied; run SHOW METHODS IN %s.%s.%s",
			resourcePath, ProviderName, auditService, relationName(resourcePath))
	default:
		return omnisdk.Method{}, fmt.Errorf(
			"intrinsic: several methods of '%s' are satisfiable (%s); disambiguate with %s = '<name>'",
			resourcePath, strings.Join(methodPaths(satisfied), ", "), methodPredicate)
	}
}

func satisfiedBy(method omnisdk.Method, params map[string]string) bool {
	for _, name := range requiredParamNames(method) {
		if params[name] == "" {
			return false
		}
	}
	return true
}

func methodPaths(methods []omnisdk.Method) []string {
	out := make([]string, 0, len(methods))
	for _, method := range methods {
		out = append(out, method.Path)
	}
	return out
}

func lastSegment(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func openStream(
	ctx queryContext, resourcePath string, params map[string]string,
	exprs sqlparser.SelectExprs) (*rowStream, error) {
	method, err := pickMethod(resourcePath, params)
	if err != nil {
		return nil, err
	}
	input := previewCfg
	args := omnisdk.Args{
		Params:                params,
		Auth:                  omnisdkAuth(providerAuthContext(ctx, resourcePath)),
		Endpoint:              input.getEndpoint(),
		InsecureSkipTLSVerify: input.getInsecureSkipTLSVerify(),
	}
	plan, err := omnisdk.Default().New(method.Path, args)
	if err != nil {
		return nil, err
	}
	rows, err := plan.Open(context.Background())
	if err != nil {
		return nil, err
	}
	selected, projectionErr := projection(exprs, schemaColumns(method.Schema))
	if projectionErr != nil {
		return nil, projectionErr
	}
	return &rowStream{
		rows:          rows,
		columns:       selected,
		batchSize:     input.getBatchSize(),
		flushInterval: input.getFlushInterval(),
	}, nil
}

type rowStream struct {
	rows          omnisdk.Rows
	batchSize     int
	flushInterval time.Duration
	produced      chan omnisdk.Row
	producerOnce  sync.Once
	columns       []column
	table         sqldata.ISQLTable
	projection    sqlparser.SelectExprs
	typCfg        columnFactory
	done          bool
}

type columnFactory interface {
	GetPlaceholderColumn(table sqldata.ISQLTable, colName string, colOID oid.Oid) sqldata.ISQLColumn
}

func (rs *rowStream) Read() (sqldata.ISQLResult, error) {
	if rs.done {
		return rs.result(nil), io.EOF
	}
	rs.startProducer()
	size := rs.batchSize
	if size < 1 {
		size = defaultBatchSize
	}
	batch := make([]omnisdk.Row, 0, size)
	// Block for the first row, then take whatever else has arrived within the
	// flush interval. A batch is therefore a cap, not a threshold: a result
	// smaller than the batch still reaches the caller promptly.
	row, ok := <-rs.produced
	if !ok {
		rs.done = true
		if err := rs.rows.Err(); err != nil {
			return rs.result(nil), err
		}
		return rs.result(nil), io.EOF
	}
	batch = append(batch, row)
	deadline := time.After(rs.flushIntervalOrDefault())
	for len(batch) < size {
		select {
		case next, more := <-rs.produced:
			if !more {
				rs.done = true
				if err := rs.rows.Err(); err != nil {
					return rs.result(batch), err
				}
				return rs.result(batch), io.EOF
			}
			batch = append(batch, next)
		case <-deadline:
			return rs.result(batch), nil
		}
	}
	return rs.result(batch), nil
}

func (rs *rowStream) flushIntervalOrDefault() time.Duration {
	if rs.flushInterval <= 0 {
		return defaultFlushInterval
	}
	return rs.flushInterval
}

// startProducer pulls the cursor on its own goroutine, so a read can bound how
// long it waits for a batch to fill without abandoning rows already produced.
func (rs *rowStream) startProducer() {
	rs.producerOnce.Do(func() {
		rs.produced = make(chan omnisdk.Row)
		go func() {
			defer close(rs.produced)
			for rs.rows.Next() {
				rs.produced <- rs.rows.Row()
			}
		}()
	})
}

func (rs *rowStream) result(batch []omnisdk.Row) sqldata.ISQLResult {
	if len(rs.columns) == 0 && len(batch) > 0 {
		for _, name := range sortedKeys(batch[0]) {
			rs.columns = append(rs.columns, column{name: name})
		}
		if len(rs.projection) > 0 {
			if selected, err := projection(rs.projection, rs.columns); err == nil {
				rs.columns = selected
			}
		}
	}
	columns := make([]sqldata.ISQLColumn, 0, len(rs.columns))
	for _, col := range rs.columns {
		columns = append(columns, rs.typCfg.GetPlaceholderColumn(rs.table, col.name, col.oid()))
	}
	rows := make([]sqldata.ISQLRow, 0, len(batch))
	for _, row := range batch {
		values := make([]interface{}, 0, len(rs.columns))
		for _, col := range rs.columns {
			values = append(values, textValue(row[col.sourceKey()]))
		}
		rows = append(rows, sqldata.NewSQLRow(values))
	}
	return sqldata.NewSQLResult(columns, uint64(len(rows)), 0, rows)
}

func (rs *rowStream) Write(sqldata.ISQLResult) error {
	return fmt.Errorf("intrinsic: omnisdk result stream is read-only")
}

func (rs *rowStream) Close() error {
	return rs.rows.Close()
}

func sortedKeys(row omnisdk.Row) []string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func selectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	if len(node.From) != 1 {
		return nil, false
	}
	aliased, ok := node.From[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return nil, false
	}
	tableName, ok := aliased.Expr.(sqlparser.TableName)
	if !ok {
		return nil, false
	}
	if bundle, isDoc := docProvider(
		resolveProvider(tableName.QualifierSecond.GetRawVal(), currentProvider)); isDoc {
		return docSelectFunc(ctx, node, bundle,
			tableName.Qualifier.GetRawVal(), tableName.Name.GetRawVal())
	}
	if !strings.EqualFold(tableName.Qualifier.GetRawVal(), auditService) ||
		!IsProvider(resolveProvider(tableName.QualifierSecond.GetRawVal(), currentProvider)) {
		return nil, false
	}
	resource, ok := lookupDataRelation(tableName.Name.GetRawVal())
	if !ok {
		return nil, false
	}
	if unsupported := unsupportedClauses(node); len(unsupported) > 0 {
		return func() internaldto.ExecutorOutput {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf(
				"relation '%s.%s.%s' streams its rows, so %s cannot be applied; remove %s from the query",
				ProviderName, auditService, relationName(resource.Path),
				strings.Join(unsupported, ", "), pluralClause(len(unsupported))))
		}, true
	}
	_, projectionErr := projection(node.SelectExprs, schemaColumns(resource.Schema))
	if projectionErr != nil {
		return func() internaldto.ExecutorOutput {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf(
				"relation '%s.%s.%s' streams its rows: %w",
				ProviderName, auditService, relationName(resource.Path), projectionErr))
		}, true
	}
	params, badPredicates := equalityPredicates(node.Where)
	if len(badPredicates) > 0 {
		return func() internaldto.ExecutorOutput {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf(
				"relation '%s.%s.%s' streams its rows, so only equality predicates are applied; "+
					"%s cannot be honoured",
				ProviderName, auditService, relationName(resource.Path),
				strings.Join(badPredicates, ", ")))
		}, true
	}
	return func() internaldto.ExecutorOutput {
		stream, err := openStream(ctx, resource.Path, params, node.SelectExprs)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		stream.table = sqldata.NewSQLTable(0, relationName(resource.Path))
		stream.typCfg = ctx.GetTypingConfig()
		primed, readErr := newPrimedStream(stream)
		if readErr != nil {
			return internaldto.NewErroneousExecutorOutput(readErr)
		}
		return internaldto.NewExecutorOutput(primed, nil, nil, nil, nil)
	}, true
}

// unsupportedClauses names the parts of a select that the streaming path cannot
// honour. Rows never reach the SQL backend, so anything the backend would have
// applied has to be refused rather than quietly dropped.
func unsupportedClauses(node *sqlparser.Select) []string {
	var out []string
	if len(node.OrderBy) > 0 {
		out = append(out, "ORDER BY")
	}
	if len(node.GroupBy) > 0 {
		out = append(out, "GROUP BY")
	}
	if node.Having != nil {
		out = append(out, "HAVING")
	}
	if node.Distinct {
		out = append(out, "DISTINCT")
	}
	if node.Limit != nil {
		out = append(out, "LIMIT")
	}
	return out
}

// projection resolves the select list against the relation's columns. A star
// selects them all; named columns are emitted in the order asked for. Anything
// else - an aggregate, a function, a literal - needs the SQL backend, which
// streamed rows never reach, so it is refused rather than quietly dropped.
func projection(exprs sqlparser.SelectExprs, available []column) ([]column, error) {
	if len(exprs) == 1 {
		if _, isStar := exprs[0].(*sqlparser.StarExpr); isStar {
			return available, nil
		}
	}
	byName := make(map[string]column, len(available))
	for _, col := range available {
		byName[strings.ToLower(col.name)] = col
	}
	out := make([]column, 0, len(exprs))
	for _, expr := range exprs {
		aliased, isAliased := expr.(*sqlparser.AliasedExpr)
		if !isAliased {
			return nil, fmt.Errorf("'%s' cannot be applied to a streamed relation", sqlparser.String(expr))
		}
		colName, isCol := aliased.Expr.(*sqlparser.ColName)
		if !isCol {
			return nil, fmt.Errorf("'%s' cannot be applied to a streamed relation", sqlparser.String(expr))
		}
		found, ok := byName[strings.ToLower(colName.Name.GetRawVal())]
		if !ok {
			return nil, fmt.Errorf("column '%s' does not exist", colName.Name.GetRawVal())
		}
		if alias := aliased.As.GetRawVal(); alias != "" {
			found.name = alias
			found.sourceName = colName.Name.GetRawVal()
		}
		out = append(out, found)
	}
	return out, nil
}

func pluralClause(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}

func equalityPredicates(where *sqlparser.Where) (map[string]string, []string) {
	params := map[string]string{}
	var bad []string
	if where == nil {
		return params, bad
	}
	var walk func(expr sqlparser.Expr)
	walk = func(expr sqlparser.Expr) {
		switch node := expr.(type) {
		case *sqlparser.AndExpr:
			walk(node.Left)
			walk(node.Right)
		case *sqlparser.ComparisonExpr:
			col, isCol := node.Left.(*sqlparser.ColName)
			val, isVal := node.Right.(*sqlparser.SQLVal)
			if !isCol || !isVal || node.Operator != sqlparser.EqualStr {
				bad = append(bad, sqlparser.String(expr))
				return
			}
			params[col.Name.GetRawVal()] = string(val.Val)
		default:
			bad = append(bad, sqlparser.String(expr))
		}
	}
	walk(where.Expr)
	return params, bad
}

type relationMethod struct {
	name           string
	description    string
	requiredParams []string
}

func (t table) methods() []relationMethod {
	viewMethod := []relationMethod{{
		name:        selectMethodName,
		description: "select-only intrinsic method",
	}}
	resource, ok := lookupDataRelation(t.name)
	if !ok {
		return viewMethod
	}
	sdkMethods, err := omnisdk.Default().Methods(resource.Path)
	if err != nil {
		return viewMethod
	}
	out := make([]relationMethod, 0, len(sdkMethods))
	for _, method := range sdkMethods {
		out = append(out, relationMethod{
			name:           lastSegment(method.Path),
			description:    method.Summary,
			requiredParams: requiredParamNames(method),
		})
	}
	return out
}

func (t table) methodColumns(methodName string) ([]column, bool) {
	resource, ok := lookupDataRelation(t.name)
	if !ok {
		return nil, false
	}
	sdkMethods, err := omnisdk.Default().Methods(resource.Path)
	if err != nil {
		return nil, false
	}
	for _, method := range sdkMethods {
		if strings.EqualFold(lastSegment(method.Path), methodName) ||
			strings.EqualFold(method.Path, methodName) {
			return schemaColumns(method.Schema), true
		}
	}
	return nil, false
}

var cloudProviders = map[string]string{ //nolint:gochecknoglobals // fixed mapping
	"aws":    "aws",
	"google": "google",
	"gcp":    "google",
	"azure":  "azure",
}

// providerAuthContext is the stackql auth context for the cloud behind a
// resource. It carries both the credentials and the tuning values.
func providerAuthContext(ctx queryContext, resourcePath string) *dto.AuthCtx {
	cloud, _, _ := strings.Cut(resourcePath, ".")
	providerName, ok := cloudProviders[cloud]
	if !ok {
		return nil
	}
	authCtx, err := ctx.GetAuthContext(providerName)
	if err != nil {
		return nil
	}
	return authCtx
}

func omnisdkAuth(authCtx *dto.AuthCtx) *omnisdk.Auth {
	if authCtx == nil {
		return nil
	}
	auth := &omnisdk.Auth{
		Type:        authCtx.Type,
		ValuePrefix: authCtx.ValuePrefix,
		Name:        authCtx.Name,
		Scopes:      authCtx.Scopes,
		TokenURL:    authCtx.GetTokenURL(),
	}
	if credentials, credErr := authCtx.GetCredentialsBytes(); credErr == nil {
		auth.SecretAccessKey = string(credentials)
		auth.Credentials = string(credentials)
	}
	if keyID, keyErr := authCtx.GetKeyIDString(); keyErr == nil {
		auth.AccessKeyID = keyID
	}
	if sessionToken, tokErr := authCtx.GetAwsSessionTokenString(); tokErr == nil {
		auth.SessionToken = sessionToken
	}
	if clientID, idErr := authCtx.GetClientID(); idErr == nil {
		auth.ClientID = clientID
	}
	if clientSecret, secErr := authCtx.GetClientSecret(); secErr == nil {
		auth.ClientSecret = clientSecret
	}
	return auth
}

type primedStream struct {
	inner    *rowStream
	first    sqldata.ISQLResult
	firstErr error
	replayed bool
}

func newPrimedStream(inner *rowStream) (sqldata.ISQLResultStream, error) {
	first, err := inner.Read()
	if err != nil && !errors.Is(err, io.EOF) {
		inner.Close() //nolint:errcheck // the read error is the one worth reporting
		return nil, err
	}
	return &primedStream{inner: inner, first: first, firstErr: err}, nil
}

func (ps *primedStream) Read() (sqldata.ISQLResult, error) {
	if !ps.replayed {
		ps.replayed = true
		return ps.first, ps.firstErr
	}
	return ps.inner.Read()
}

func (ps *primedStream) Write(sqldata.ISQLResult) error {
	return fmt.Errorf("intrinsic: omnisdk result stream is read-only")
}

func (ps *primedStream) Close() error {
	return ps.inner.Close()
}

func (c column) reportedType() string {
	switch c.dataType {
	case "":
		return columnType
	case "boolean":
		return "bool"
	default:
		return c.dataType
	}
}

func (c column) oid() oid.Oid {
	return oid.T_text
}

func schemaTypeOf(property map[string]any) string {
	if lossless, ok := property["x-omnisdk-lossless"].(bool); ok && lossless {
		switch format, _ := property["format"].(string); format {
		case "int32", "int64":
			return "integer"
		case "float", "double":
			return "number"
		}
	}
	switch declared := property["type"].(type) {
	case string:
		return declared
	case []any:
		for _, member := range declared {
			if name, isString := member.(string); isString && name != "null" {
				return name
			}
		}
	}
	return ""
}

func textValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

// backendInput carries the tunables for a streaming run. It is read once, at
// construction, and only read thereafter.
type backendInput interface {
	getBatchSize() int
	getEndpoint() string
	getFlushInterval() time.Duration
	getInsecureSkipTLSVerify() bool
}

type standardBackendInput struct {
	batchSize             int
	endpoint              string
	flushInterval         time.Duration
	insecureSkipTLSVerify bool
}

// previewCfg is the parsed --preview argument. Cobra binds the raw string in
// internal/stackql/cmd, which calls Init exactly once; nothing else writes it.
//
//nolint:gochecknoglobals // set once from the CLI, read thereafter
var previewCfg = newBackendInput(previewCfgDTO{})

// CfgRawKey is the CLI argument that configures this provider's backend.
const CfgRawKey = "preview"

// previewCfgDTO is the wire shape of the --preview argument. Endpoint accepts
// either form omnisdk does: a base URL for every service, or an object of
// service to override. Both ride through as the string omnisdk parses.
type previewCfgDTO struct {
	BatchSize             int             `json:"batchSize"`
	FlushInterval         string          `json:"flushInterval"`
	Endpoint              json.RawMessage `json:"endpoint"`
	InsecureSkipTLSVerify bool            `json:"insecureSkipTLSVerify"`
}

func (c previewCfgDTO) endpoint() string {
	if len(c.Endpoint) == 0 {
		return ""
	}
	var asURL string
	if err := json.Unmarshal(c.Endpoint, &asURL); err == nil {
		return asURL
	}
	return string(c.Endpoint)
}

// Init records the --preview argument. It is called once, from cmd, before any
// query runs.
func Init(raw string) {
	var cfg previewCfgDTO
	if strings.TrimSpace(raw) != "" {
		//nolint:errcheck // a malformed argument leaves the defaults in place
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	previewCfg = newBackendInput(cfg)
}

func newBackendInput(cfg previewCfgDTO) backendInput {
	rv := &standardBackendInput{
		batchSize:             defaultBatchSize,
		endpoint:              cfg.endpoint(),
		flushInterval:         defaultFlushInterval,
		insecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
	}
	if cfg.BatchSize > 0 {
		rv.batchSize = cfg.BatchSize
	}
	if parsed, err := time.ParseDuration(cfg.FlushInterval); err == nil && parsed > 0 {
		rv.flushInterval = parsed
	}
	return rv
}

func (b *standardBackendInput) getBatchSize() int { return b.batchSize }

func (b *standardBackendInput) getEndpoint() string { return b.endpoint }

func (b *standardBackendInput) getFlushInterval() time.Duration { return b.flushInterval }

func (b *standardBackendInput) getInsecureSkipTLSVerify() bool { return b.insecureSkipTLSVerify }

// sourceKey is the row key a column reads from: its own name, unless an alias
// renamed it.
func (c column) sourceKey() string {
	if c.sourceName != "" {
		return c.sourceName
	}
	return c.name
}
