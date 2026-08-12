package intrinsic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lib/pq/oid"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const methodPredicate = "method"

// endpointEnvVar retargets omnisdk at a local mock. Transport configuration, so
// it stays out of the query.
const endpointEnvVar = "STACKQL_PREVIEW_ENDPOINT"

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
	ctx queryContext, resourcePath string, params map[string]string) (*rowStream, error) {
	method, err := pickMethod(resourcePath, params)
	if err != nil {
		return nil, err
	}
	args := omnisdk.Args{
		Params:   params,
		Auth:     omnisdkAuth(ctx, resourcePath),
		Endpoint: os.Getenv(endpointEnvVar),
	}
	plan, err := omnisdk.Default().New(method.Path, args)
	if err != nil {
		return nil, err
	}
	rows, err := plan.Open(context.Background())
	if err != nil {
		return nil, err
	}
	return &rowStream{rows: rows, columns: schemaColumns(method.Schema)}, nil
}

type rowStream struct {
	rows    omnisdk.Rows
	columns []column
	table   sqldata.ISQLTable
	typCfg  columnFactory
	done    bool
}

type columnFactory interface {
	GetPlaceholderColumn(table sqldata.ISQLTable, colName string, colOID oid.Oid) sqldata.ISQLColumn
}

func (rs *rowStream) Read() (sqldata.ISQLResult, error) {
	if rs.done {
		return rs.result(nil), io.EOF
	}
	rs.done = true
	var batch []omnisdk.Row
	for rs.rows.Next() {
		batch = append(batch, rs.rows.Row())
	}
	if err := rs.rows.Err(); err != nil {
		return rs.result(nil), err
	}
	return rs.result(batch), io.EOF
}

func (rs *rowStream) result(batch []omnisdk.Row) sqldata.ISQLResult {
	if len(rs.columns) == 0 && len(batch) > 0 {
		for _, name := range sortedKeys(batch[0]) {
			rs.columns = append(rs.columns, column{name: name})
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
			values = append(values, textValue(row[col.name]))
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
	if !strings.EqualFold(tableName.Qualifier.GetRawVal(), auditService) ||
		!IsProvider(resolveProvider(tableName.QualifierSecond.GetRawVal(), currentProvider)) {
		return nil, false
	}
	resource, ok := lookupDataRelation(tableName.Name.GetRawVal())
	if !ok {
		return nil, false
	}
	params := equalityPredicates(node.Where)
	return func() internaldto.ExecutorOutput {
		stream, err := openStream(ctx, resource.Path, params)
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

func equalityPredicates(where *sqlparser.Where) map[string]string {
	params := map[string]string{}
	if where == nil {
		return params
	}
	var walk func(expr sqlparser.Expr)
	walk = func(expr sqlparser.Expr) {
		switch node := expr.(type) {
		case *sqlparser.AndExpr:
			walk(node.Left)
			walk(node.Right)
		case *sqlparser.ComparisonExpr:
			if node.Operator != sqlparser.EqualStr {
				return
			}
			col, isCol := node.Left.(*sqlparser.ColName)
			val, isVal := node.Right.(*sqlparser.SQLVal)
			if !isCol || !isVal {
				return
			}
			params[col.Name.GetRawVal()] = string(val.Val)
		}
	}
	walk(where.Expr)
	return params
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

func omnisdkAuth(ctx queryContext, resourcePath string) *omnisdk.Auth {
	cloud, _, _ := strings.Cut(resourcePath, ".")
	providerName, ok := cloudProviders[cloud]
	if !ok {
		return nil
	}
	authCtx, err := ctx.GetAuthContext(providerName)
	if err != nil || authCtx == nil {
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
