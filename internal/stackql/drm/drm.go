package drm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/relationaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"
)

type DRM interface {
	DRMConfig
}

type ColumnMetadata struct {
	Coupling dto.DRMCoupling
	Column   openapistackql.ColumnDescriptor
}

func (cd ColumnMetadata) GetName() string {
	return cd.Column.Name
}

func (cd ColumnMetadata) GetIdentifier() string {
	return cd.Column.GetIdentifier()
}

func (cd ColumnMetadata) GetType() string {
	if cd.Column.Schema != nil {
		return cd.Column.Schema.Type
	}
	return parserutil.ExtractStringRepresentationOfValueColumn(cd.Column.Val)
}

func NewColDescriptor(col openapistackql.ColumnDescriptor, relTypeStr string) ColumnMetadata {
	return ColumnMetadata{
		Coupling: dto.DRMCoupling{RelationalType: relTypeStr, GolangKind: reflect.String},
		Column:   col,
	}
}

type PreparedStatementCtx struct {
	query                   string
	kind                    string // string annotation applicable only in some cases eg UNION [ALL]
	genIdControlColName     string
	sessionIdControlColName string
	TableNames              []string
	txnIdControlColName     string
	insIdControlColName     string
	insEncodedColName       string
	nonControlColumns       []ColumnMetadata
	ctrlColumnRepeats       int
	txnCtrlCtrs             *dto.TxnControlCounters
	selectTxnCtrlCtrs       []*dto.TxnControlCounters
	namespaceCollection     tablenamespace.TableNamespaceCollection
	sqlDialect              sqldialect.SQLDialect
}

func (ps *PreparedStatementCtx) SetKind(kind string) {
	ps.kind = kind
}

func (ps *PreparedStatementCtx) GetQuery() string {
	return ps.query
}

func (ps *PreparedStatementCtx) GetGCCtrlCtrs() *dto.TxnControlCounters {
	return ps.txnCtrlCtrs
}

func (ps *PreparedStatementCtx) SetGCCtrlCtrs(tcc *dto.TxnControlCounters) {
	ps.txnCtrlCtrs = tcc
}

func (ps *PreparedStatementCtx) GetNonControlColumns() []ColumnMetadata {
	return ps.nonControlColumns
}

func (ps *PreparedStatementCtx) GetAllCtrlCtrs() []*dto.TxnControlCounters {
	var rv []*dto.TxnControlCounters
	rv = append(rv, ps.txnCtrlCtrs)
	rv = append(rv, ps.selectTxnCtrlCtrs...)
	return rv
}

func NewPreparedStatementCtx(
	query string,
	kind string,
	genIdControlColName string,
	sessionIdControlColName string,
	tableNames []string,
	txnIdControlColName string,
	insIdControlColName string,
	insEncodedColName string,
	nonControlColumns []ColumnMetadata,
	ctrlColumnRepeats int,
	txnCtrlCtrs *dto.TxnControlCounters,
	secondaryCtrs []*dto.TxnControlCounters,
	namespaceCollection tablenamespace.TableNamespaceCollection,
	sqlDialect sqldialect.SQLDialect,
) *PreparedStatementCtx {
	return &PreparedStatementCtx{
		query:                   query,
		kind:                    kind,
		genIdControlColName:     genIdControlColName,
		sessionIdControlColName: sessionIdControlColName,
		TableNames:              tableNames,
		txnIdControlColName:     txnIdControlColName,
		insIdControlColName:     insIdControlColName,
		insEncodedColName:       insEncodedColName,
		nonControlColumns:       nonControlColumns,
		ctrlColumnRepeats:       ctrlColumnRepeats,
		txnCtrlCtrs:             txnCtrlCtrs,
		selectTxnCtrlCtrs:       secondaryCtrs,
		namespaceCollection:     namespaceCollection,
		sqlDialect:              sqlDialect,
	}
}

func NewQueryOnlyPreparedStatementCtx(query string) *PreparedStatementCtx {
	return &PreparedStatementCtx{query: query, ctrlColumnRepeats: 0}
}

func (ps PreparedStatementCtx) GetGCHousekeepingQueries() string {
	var housekeepingQueries []string
	for _, table := range ps.TableNames {
		housekeepingQueries = append(housekeepingQueries, ps.sqlDialect.GetGCHousekeepingQuery(table, *ps.txnCtrlCtrs))
	}
	return strings.Join(housekeepingQueries, "; ")
}

type PreparedStatementParameterized struct {
	Ctx                 *PreparedStatementCtx
	args                map[string]interface{}
	controlArgsRequired bool
	requestEncoding     string
	children            map[int]PreparedStatementParameterized
}

func (ps PreparedStatementParameterized) AddChild(key int, val PreparedStatementParameterized) {
	ps.children[key] = val
}

type PreparedStatementArgs struct {
	query    string
	args     []interface{}
	children map[int]PreparedStatementArgs
}

func NewPreparedStatementArgs(query string) PreparedStatementArgs {
	return PreparedStatementArgs{
		query:    query,
		children: make(map[int]PreparedStatementArgs),
	}
}

func NewPreparedStatementParameterized(ctx *PreparedStatementCtx, args map[string]interface{}, controlArgsRequired bool) PreparedStatementParameterized {
	return PreparedStatementParameterized{
		Ctx:                 ctx,
		args:                args,
		controlArgsRequired: controlArgsRequired,
		children:            make(map[int]PreparedStatementParameterized),
	}
}

type DRMConfig interface {
	ExtractFromGolangValue(interface{}) interface{}
	ExtractObjectFromSQLRows(r *sql.Rows, nonControlColumns []ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{})
	GetCurrentTable(*dto.HeirarchyIdentifiers) (dto.DBTable, error)
	GetRelationalType(string) string
	GenerateDDL(util.AnnotatedTabulation, *openapistackql.OperationStore, int, bool) ([]string, error)
	GetControlAttributes() sqlcontrol.ControlAttributes
	GetGolangValue(string) interface{}
	GetGolangSlices([]ColumnMetadata) ([]interface{}, []string)
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetParserTableName(*dto.HeirarchyIdentifiers, int) sqlparser.TableName
	GetSQLDialect() sqldialect.SQLDialect
	GetTable(*dto.HeirarchyIdentifiers, int) (dto.DBTable, error)
	GenerateInsertDML(util.AnnotatedTabulation, *openapistackql.OperationStore, *dto.TxnControlCounters) (*PreparedStatementCtx, error)
	GenerateSelectDML(util.AnnotatedTabulation, *dto.TxnControlCounters, string, string) (*PreparedStatementCtx, error)
	ExecuteInsertDML(sqlengine.SQLEngine, *PreparedStatementCtx, map[string]interface{}, string) (sql.Result, error)
	QueryDML(sqlengine.SQLEngine, PreparedStatementParameterized) (*sql.Rows, error)
}

type StaticDRMConfig struct {
	namespaceCollection tablenamespace.TableNamespaceCollection
	controlAttributes   sqlcontrol.ControlAttributes
	sqlEngine           sqlengine.SQLEngine
	sqlDialect          sqldialect.SQLDialect
}

func (dc *StaticDRMConfig) GetSQLDialect() sqldialect.SQLDialect {
	return dc.sqlDialect
}

func (dc *StaticDRMConfig) GetTable(hids *dto.HeirarchyIdentifiers, discoveryID int) (dto.DBTable, error) {
	return dc.sqlDialect.GetTable(hids, discoveryID)
}

func (dc *StaticDRMConfig) GetControlAttributes() sqlcontrol.ControlAttributes {
	return dc.getControlAttributes()
}

func (dc *StaticDRMConfig) getControlAttributes() sqlcontrol.ControlAttributes {
	return dc.controlAttributes
}

func (dc *StaticDRMConfig) GetGolangSlices(nonControlColumns []ColumnMetadata) ([]interface{}, []string) {
	return dc.getGolangSlices(nonControlColumns)
}

func (dc *StaticDRMConfig) ExtractObjectFromSQLRows(r *sql.Rows, nonControlColumns []ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
	return dc.extractObjectFromSQLRows(r, nonControlColumns, stream)
}

func (dc *StaticDRMConfig) extractObjectFromSQLRows(r *sql.Rows, nonControlColumns []ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
	if r != nil {
		defer r.Close()
	}
	altKeys := make(map[string]map[string]interface{})
	rawRows := make(map[int]map[int]interface{})
	var ks []int
	ifArr, keyArr := dc.getGolangSlices(nonControlColumns)
	if r != nil {
		i := 0
		for r.Next() {
			errScan := r.Scan(ifArr...)
			if errScan != nil {
				logging.GetLogger().Infoln(fmt.Sprintf("%v", errScan))
			}
			for ord, val := range ifArr {
				logging.GetLogger().Infoln(fmt.Sprintf("col #%d '%s':  %v  type: %T", ord, nonControlColumns[ord].GetName(), val, val))
			}
			im := make(map[string]interface{})
			imRaw := make(map[int]interface{})
			for ord, key := range keyArr {
				val := ifArr[ord]
				ev := dc.ExtractFromGolangValue(val)
				im[key] = ev
				imRaw[ord] = ev
			}
			altKeys[strconv.Itoa(i)] = im
			stream.Write([]map[string]interface{}{im})
			rawRows[i] = imRaw
			ks = append(ks, i)
			i++
		}

		for ord := range ks {
			val := altKeys[strconv.Itoa(ord)]
			logging.GetLogger().Infoln(fmt.Sprintf("row #%d:  %v  type: %T", ord, val, val))
		}
	}
	return altKeys, rawRows
}

func (dc *StaticDRMConfig) getGolangSlices(nonControlColumns []ColumnMetadata) ([]interface{}, []string) {
	i := 0
	var keyArr []string
	var ifArr []interface{}
	for i < len(nonControlColumns) {
		x := nonControlColumns[i]
		y := dc.sqlDialect.GetGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.Column.GetIdentifier())
		i++
	}
	return ifArr, keyArr
}

func (dc *StaticDRMConfig) GetRelationalType(discoType string) string {
	return dc.sqlDialect.GetRelationalType(discoType)
}

func (dc *StaticDRMConfig) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return dc.namespaceCollection
}

func (dc *StaticDRMConfig) GetGolangValue(discoType string) interface{} {
	return dc.sqlDialect.GetGolangValue(discoType)
}

func (dc *StaticDRMConfig) ExtractFromGolangValue(val interface{}) interface{} {
	return dc.extractFromGolangValue(val)
}

func (dc *StaticDRMConfig) extractFromGolangValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}
	var retVal interface{}
	switch v := val.(type) {
	case *sql.NullString:
		retVal, _ = (*v).Value()
	case *sql.NullBool:
		retVal, _ = (*v).Value()
	case *sql.NullInt64:
		retVal, _ = (*v).Value()
	case *sql.NullFloat64:
		retVal, _ = (*v).Value()
	}
	return retVal
}

func (dc *StaticDRMConfig) GetGolangKind(discoType string) reflect.Kind {
	return dc.sqlDialect.GetGolangKind(discoType)
}

func (dc *StaticDRMConfig) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers) (dto.DBTable, error) {
	tn := tableHeirarchyIDs.GetTableName()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tn) {
		templatedName, err := dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(tn)
		if err != nil {
			return dto.NewDBTableAnalytics(templatedName, -1, tableHeirarchyIDs), err
		}
		return dto.NewDBTableAnalytics(templatedName, -1, tableHeirarchyIDs), nil
	}
	return dc.sqlDialect.GetCurrentTable(tableHeirarchyIDs)
}

func (dc *StaticDRMConfig) GetTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	return dc.getTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) getTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	tbl, err := dc.sqlDialect.GetTable(hIds, discoveryGenerationID)
	if err != nil {
		return "", err
	}
	unadornedTableName := tbl.GetNameStump()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(unadornedTableName) {
		return dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(unadornedTableName)
	}
	return tbl.GetName(), nil
}

func (dc *StaticDRMConfig) GetParserTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) sqlparser.TableName {
	return dc.getParserTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) getParserTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) sqlparser.TableName {
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(hIds.GetTableName()) {
		return sqlparser.TableName{
			Name:            sqlparser.NewTableIdent(hIds.ResourceStr),
			Qualifier:       sqlparser.NewTableIdent(hIds.ServiceStr),
			QualifierSecond: sqlparser.NewTableIdent(hIds.ProviderStr),
		}
	}
	return sqlparser.TableName{
		Name:            sqlparser.NewTableIdent(fmt.Sprintf("generation_%d", discoveryGenerationID)),
		Qualifier:       sqlparser.NewTableIdent(hIds.ResourceStr),
		QualifierSecond: sqlparser.NewTableIdent(hIds.ServiceStr),
		QualifierThird:  sqlparser.NewTableIdent(hIds.ProviderStr),
	}
}

func (dc *StaticDRMConfig) inferColType(col util.Column) string {
	relationalType := "text"
	schema := col.GetSchema()
	if schema != nil && schema.Type != "" {
		relationalType = dc.GetRelationalType(schema.Type)
	}
	return relationalType
}

func (dc *StaticDRMConfig) genRelationalTable(tabAnn util.AnnotatedTabulation, m *openapistackql.OperationStore, discoveryGenerationID int) (relationaldto.RelationalTable, error) {
	tableName, err := dc.getTableName(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	relationalTable := relationaldto.NewRelationalTable(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID, tableName, tabAnn.GetInputTableName())
	schemaAnalyzer := util.NewTableSchemaAnalyzer(tabAnn.GetTabulation().GetSchema(), m)
	tableColumns := schemaAnalyzer.GetColumns()
	for _, col := range tableColumns {
		colName := col.GetName()
		colType := dc.inferColType(col)
		// relationalType := dc.GetRelationalType(colType)
		// TODO: add drm logic to infer / transform width as suplied by openapi doc
		colWidth := col.GetWidth()
		relationalColumn := relationaldto.NewRelationalColumn(colName, colType).WithWidth(colWidth)
		relationalTable.PushBackColumn(relationalColumn)
	}
	return relationalTable, nil
}

func (dc *StaticDRMConfig) GenerateDDL(tabAnn util.AnnotatedTabulation, m *openapistackql.OperationStore, discoveryGenerationID int, dropTable bool) ([]string, error) {
	relationalTable, err := dc.genRelationalTable(tabAnn, m, discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	return dc.sqlDialect.GenerateDDL(relationalTable, dropTable)
}

func (dc *StaticDRMConfig) GenerateInsertDML(tabAnnotated util.AnnotatedTabulation, method *openapistackql.OperationStore, tcc *dto.TxnControlCounters) (*PreparedStatementCtx, error) {
	var columns []ColumnMetadata
	tableName, err := dc.GetCurrentTable(tabAnnotated.GetHeirarchyIdentifiers())
	if err != nil {
		return nil, err
	}
	genIdColName := dc.controlAttributes.GetControlGenIdColumnName()
	sessionIdColName := dc.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := dc.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := dc.controlAttributes.GetControlInsIdColumnName()
	insEncodedColName := dc.controlAttributes.GetControlInsertEncodedIdColumnName()

	relationalTable := relationaldto.NewRelationalTable(tabAnnotated.GetHeirarchyIdentifiers(), tableName.GetDiscoveryID(), tableName.GetName(), tabAnnotated.GetInputTableName())
	schemaAnalyzer := util.NewTableSchemaAnalyzer(tabAnnotated.GetTabulation().GetSchema(), method)
	tableColumns := schemaAnalyzer.GetColumnDescriptors(tabAnnotated)
	for _, col := range tableColumns {
		relationalType := "text"
		schema := col.Schema
		if schema != nil && schema.Type != "" {
			relationalType = dc.GetRelationalType(schema.Type)
		}
		columns = append(columns, NewColDescriptor(col, relationalType))
		relationalColumn := relationaldto.NewRelationalColumn(col.Name, relationalType).WithParserNode(col.Node)
		relationalTable.PushBackColumn(relationalColumn)
	}
	queryString, err := dc.sqlDialect.GenerateInsertDML(relationalTable, tcc)
	if err != nil {
		return nil, err
	}
	return NewPreparedStatementCtx(
			queryString,
			"",
			genIdColName,
			sessionIdColName,
			[]string{tableName.GetName()},
			txnIdColName,
			insIdColName,
			insEncodedColName,
			columns,
			1,
			tcc,
			nil,
			dc.namespaceCollection,
			dc.sqlDialect,
		),
		nil
}

func (dc *StaticDRMConfig) GenerateSelectDML(tabAnnotated util.AnnotatedTabulation, txnCtrlCtrs *dto.TxnControlCounters, selectSuffix, rewrittenWhere string) (*PreparedStatementCtx, error) {
	var quotedColNames []string
	var columns []ColumnMetadata

	aliasStr := ""
	if tabAnnotated.GetAlias() != "" {
		aliasStr = fmt.Sprintf(` AS "%s" `, tabAnnotated.GetAlias())
	}
	tn, err := dc.GetCurrentTable(tabAnnotated.GetHeirarchyIdentifiers())
	if err != nil {
		return nil, err
	}
	relationalTable := relationaldto.NewRelationalTable(tabAnnotated.GetHeirarchyIdentifiers(), tn.GetDiscoveryID(), tn.GetName(), tabAnnotated.GetInputTableName()).WithAlias(aliasStr)
	for _, col := range tabAnnotated.GetTabulation().GetColumns() {
		var typeStr string
		if col.Schema != nil {
			typeStr = dc.GetRelationalType(col.Schema.Type)
		} else {
			if col.Val != nil {
				switch col.Val.Type {
				case sqlparser.BitVal:
				}
			}
		}
		columns = append(columns, NewColDescriptor(col, typeStr))
		// TODO: logic to infer column width
		relationalColumn := relationaldto.NewRelationalColumn(col.Name, typeStr).WithQualifier(col.Qualifier).WithParserNode(col.Node)
		if col.DecoratedCol == "" {
			if col.Alias != "" {
				relationalColumn = relationalColumn.WithAlias(col.Alias)
			}
		} else {
			relationalColumn = relationalColumn.WithDecorated(col.DecoratedCol)
		}
		relationalTable.PushBackColumn(relationalColumn)
		quotedColNames = append(quotedColNames, fmt.Sprintf("%s ", relationalColumn.CanonicalSelectionString()))
	}
	queryString, err := dc.sqlDialect.GenerateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)

	if err != nil {
		return nil, err
	}

	genIdColName := dc.controlAttributes.GetControlGenIdColumnName()
	sessionIDColName := dc.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := dc.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := dc.controlAttributes.GetControlInsIdColumnName()
	return NewPreparedStatementCtx(
		queryString,
		"",
		genIdColName,
		sessionIDColName,
		nil,
		txnIdColName,
		insIdColName,
		dc.controlAttributes.GetControlInsertEncodedIdColumnName(),
		columns,
		1,
		txnCtrlCtrs,
		nil,
		dc.namespaceCollection,
		dc.sqlDialect,
	), nil
}

func (dc *StaticDRMConfig) generateControlVarArgs(cp PreparedStatementParameterized, isInsert bool) ([]interface{}, error) {
	var varArgs []interface{}
	if cp.controlArgsRequired {
		ctrSlice := cp.Ctx.GetAllCtrlCtrs()
		for _, ctrs := range ctrSlice {
			if ctrs == nil {
				continue
			}
			varArgs = append(varArgs, ctrs.GenId)
			varArgs = append(varArgs, ctrs.SessionId)
			varArgs = append(varArgs, ctrs.TxnId)
			varArgs = append(varArgs, ctrs.InsertId)
			if isInsert {
				varArgs = append(varArgs, cp.requestEncoding)
			}
		}
	}
	return varArgs, nil
}

func (dc *StaticDRMConfig) generateVarArgs(cp PreparedStatementParameterized, isInsert bool) (PreparedStatementArgs, error) {
	retVal := NewPreparedStatementArgs(cp.Ctx.GetQuery())
	for i, child := range cp.children {
		chidRv, err := dc.generateVarArgs(child, isInsert)
		if err != nil {
			return retVal, err
		}
		retVal.children[i] = chidRv
	}
	varArgs, _ := dc.generateControlVarArgs(cp, isInsert)
	if cp.args != nil && len(cp.args) > 0 {
		for _, col := range cp.Ctx.GetNonControlColumns() {
			va, ok := cp.args[col.GetName()]
			if !ok {
				varArgs = append(varArgs, nil)
				continue
			}
			switch vt := va.(type) {
			case map[string]interface{}, []interface{}:
				b, err := json.Marshal(vt)
				if err != nil {
					return retVal, err
				}
				varArgs = append(varArgs, string(b))
			case string:
				varArgs = append(varArgs, va)
			default:
				if strings.ToLower(col.Coupling.RelationalType) == "text" && strings.ToLower(dc.sqlDialect.GetName()) == constants.SQLDialectPostgres {
					varArgs = append(varArgs, fmt.Sprintf("%v", va))
					continue
				}
				varArgs = append(varArgs, va)
			}
		}
	}
	retVal.args = varArgs
	return retVal, nil
}

func (dc *StaticDRMConfig) ExecuteInsertDML(dbEngine sqlengine.SQLEngine, ctx *PreparedStatementCtx, payload map[string]interface{}, requestEncoding string) (sql.Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cannot execute on nil PreparedStatementContext")
	}
	stmtArgs, err := dc.generateVarArgs(PreparedStatementParameterized{Ctx: ctx, args: payload, controlArgsRequired: true, requestEncoding: requestEncoding}, true)
	if err != nil {
		return nil, err
	}
	return dbEngine.Exec(stmtArgs.query, stmtArgs.args...)
}

func (dc *StaticDRMConfig) QueryDML(dbEngine sqlengine.SQLEngine, ctxParameterized PreparedStatementParameterized) (*sql.Rows, error) {
	if ctxParameterized.Ctx == nil {
		return nil, fmt.Errorf("cannot execute based upon nil PreparedStatementContext")
	}
	rootArgs, err := dc.generateVarArgs(ctxParameterized, false)
	if err != nil {
		return nil, err
	}
	var varArgs []interface{}
	j := 0
	query := rootArgs.query
	var childQueryStrings []interface{} // dunno why
	var keys []int
	for i := range rootArgs.children {
		keys = append(keys, i)
	}
	sort.Ints(keys)
	for _, k := range keys {
		cp := rootArgs.children[k]
		logging.GetLogger().Infoln(fmt.Sprintf("adding child query = %s", cp.query))
		childQueryStrings = append(childQueryStrings, cp.query)
		if len(rootArgs.args) >= k {
			varArgs = append(varArgs, rootArgs.args[j:k]...)
		}
		varArgs = append(varArgs, cp.args...)
		j = k
	}
	logging.GetLogger().Infoln(fmt.Sprintf("raw query = %s", query))
	if len(childQueryStrings) > 0 {
		query = fmt.Sprintf(rootArgs.query, childQueryStrings...)
	}
	if len(rootArgs.args) >= j {
		varArgs = append(varArgs, rootArgs.args[j:]...)
	}
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	return dbEngine.Query(query, varArgs...)
}

func GetDRMConfig(sqlDialect sqldialect.SQLDialect, namespaceCollection tablenamespace.TableNamespaceCollection, controlAttributes sqlcontrol.ControlAttributes) (DRMConfig, error) {
	rv := &StaticDRMConfig{
		namespaceCollection: namespaceCollection,
		controlAttributes:   controlAttributes,
		sqlEngine:           sqlDialect.GetSQLEngine(),
		sqlDialect:          sqlDialect,
	}
	return rv, nil
}
