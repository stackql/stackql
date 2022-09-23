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
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"
)

const (
	gen_id_col_name         string = "iql_generation_id"
	ssn_id_col_name         string = "iql_session_id"
	txn_id_col_name         string = "iql_txn_id"
	ins_id_col_name         string = "iql_insert_id"
	insert_endoded_col_name string = "iql_insert_encoded"
	latest_update_col_name  string = "iql_last_modified"
	gc_status_col_name      string = "iql_gc_status"
)

type DRM interface {
	DRMConfig
}

type DRMCoupling struct {
	RelationalType string
	GolangKind     reflect.Kind
}

type ColumnMetadata struct {
	Coupling DRMCoupling
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

func (cd ColumnMetadata) getTypeFromVal() string {
	switch cd.Column.Val.Type {
	case sqlparser.BitVal, sqlparser.HexNum, sqlparser.HexVal, sqlparser.StrVal:
		return "string"
	case sqlparser.FloatVal:
		return "float"
	case sqlparser.IntVal:
		return "int"
	default:
		return "string"
	}
}

func NewColDescriptor(col openapistackql.ColumnDescriptor, relTypeStr string) ColumnMetadata {
	return ColumnMetadata{
		Coupling: DRMCoupling{RelationalType: relTypeStr, GolangKind: reflect.String},
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
	}
}

func NewQueryOnlyPreparedStatementCtx(query string) *PreparedStatementCtx {
	return &PreparedStatementCtx{query: query, ctrlColumnRepeats: 0}
}

func (ps PreparedStatementCtx) GetGCHousekeepingQueries() string {
	templateQuery := `INSERT OR IGNORE INTO 
	  "__iql__.control.gc.txn_table_x_ref" (
			iql_generation_id, 
			iql_session_id, 
			iql_transaction_id, 
			table_name
		) values(%d, %d, %d, '%s')`
	var housekeepingQueries []string
	for _, table := range ps.TableNames {
		housekeepingQueries = append(housekeepingQueries, fmt.Sprintf(templateQuery, ps.txnCtrlCtrs.GenId, ps.txnCtrlCtrs.SessionId, ps.txnCtrlCtrs.TxnId, table))
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
	GetCurrentTable(*dto.HeirarchyIdentifiers, sqlengine.SQLEngine) (dto.DBTable, error)
	GetRelationalType(string) string
	GenerateDDL(util.AnnotatedTabulation, *openapistackql.OperationStore, int, bool) ([]string, error)
	GetGolangValue(string) interface{}
	GetGolangSlices([]ColumnMetadata) ([]interface{}, []string)
	GetInsControlColumn() string
	GetInsertEncodedControlColumn() string
	GetLastUpdatedControlColumn() string
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetParserTableName(*dto.HeirarchyIdentifiers, int) sqlparser.TableName
	GetSessionControlColumn() string
	GetTableName(*dto.HeirarchyIdentifiers, int) (string, error)
	GetTxnControlColumn() string
	GetGenerationControlColumn() string
	GenerateInsertDML(util.AnnotatedTabulation, *openapistackql.OperationStore, *dto.TxnControlCounters) (*PreparedStatementCtx, error)
	GenerateSelectDML(util.AnnotatedTabulation, *dto.TxnControlCounters, string, string) (*PreparedStatementCtx, error)
	ExecuteInsertDML(sqlengine.SQLEngine, *PreparedStatementCtx, map[string]interface{}, string) (sql.Result, error)
	QueryDML(sqlengine.SQLEngine, PreparedStatementParameterized) (*sql.Rows, error)
}

type StaticDRMConfig struct {
	typeMappings          map[string]DRMCoupling
	defaultRelationalType string
	defaultGolangKind     reflect.Kind
	defaultGolangValue    interface{}
	namespaceCollection   tablenamespace.TableNamespaceCollection
}

func (dc *StaticDRMConfig) getDefaultGolangValue() interface{} {
	return &sql.NullString{}
}

func (dc *StaticDRMConfig) getDefaultGolangKind() reflect.Kind {
	return dc.defaultGolangKind
}

func (dc *StaticDRMConfig) GetGolangSlices(nonControlColumns []ColumnMetadata) ([]interface{}, []string) {
	return dc.getGolangSlices(nonControlColumns)
}

func (dc *StaticDRMConfig) ExtractObjectFromSQLRows(r *sql.Rows, nonControlColumns []ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
	return dc.extractObjectFromSQLRows(r, nonControlColumns, stream)
}

func (dc *StaticDRMConfig) extractObjectFromSQLRows(r *sql.Rows, nonControlColumns []ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
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
		y := dc.getGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.Column.GetIdentifier())
		i++
	}
	return ifArr, keyArr
}

func (dc *StaticDRMConfig) GetRelationalType(discoType string) string {
	rv, ok := dc.typeMappings[discoType]
	if ok {
		return rv.RelationalType
	}
	return dc.defaultRelationalType
}

func (dc *StaticDRMConfig) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return dc.namespaceCollection
}

func (dc *StaticDRMConfig) GetGolangValue(discoType string) interface{} {
	return dc.getGolangValue(discoType)
}

func (dc *StaticDRMConfig) getGolangValue(discoType string) interface{} {
	rv, ok := dc.typeMappings[discoType]
	if !ok {
		return dc.getDefaultGolangValue()
	}
	switch rv.GolangKind {
	case reflect.String:
		return &sql.NullString{}
	case reflect.Array:
		return &sql.NullString{}
	case reflect.Bool:
		return &sql.NullBool{}
	case reflect.Map:
		return &sql.NullString{}
	case reflect.Int:
		return &sql.NullInt64{}
	}
	return dc.getDefaultGolangValue()
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
	}
	return retVal
}

func (dc *StaticDRMConfig) GetGolangKind(discoType string) reflect.Kind {
	rv, ok := dc.typeMappings[discoType]
	if !ok {
		return dc.getDefaultGolangKind()
	}
	return rv.GolangKind
}

func (dc *StaticDRMConfig) GetGenerationControlColumn() string {
	return dc.getGenerationControlColumn()
}

func (dc *StaticDRMConfig) getGenerationControlColumn() string {
	return gen_id_col_name
}

func (dc *StaticDRMConfig) GetSessionControlColumn() string {
	return dc.getSessionControlColumn()
}

func (dc *StaticDRMConfig) GetGCStatusColumn() string {
	return dc.getGCStatusColumn()
}

func (dc *StaticDRMConfig) getSessionControlColumn() string {
	return ssn_id_col_name
}

func (dc *StaticDRMConfig) getGCStatusColumn() string {
	return gc_status_col_name
}

func (dc *StaticDRMConfig) GetTxnControlColumn() string {
	return dc.getTxnControlColumn()
}

func (dc *StaticDRMConfig) getTxnControlColumn() string {
	return txn_id_col_name
}

func (dc *StaticDRMConfig) GetInsControlColumn() string {
	return dc.getInsControlColumn()
}

func (dc *StaticDRMConfig) getInsControlColumn() string {
	return ins_id_col_name
}

func (dc *StaticDRMConfig) GetLastUpdatedControlColumn() string {
	return dc.getLastUpdatedControlColumn()
}

func (dc *StaticDRMConfig) getLastUpdatedControlColumn() string {
	return latest_update_col_name
}

func (dc *StaticDRMConfig) GetInsertEncodedControlColumn() string {
	return dc.getInsertEncodedControlColumn()
}

func (dc *StaticDRMConfig) getInsertEncodedControlColumn() string {
	return insert_endoded_col_name
}

func (dc *StaticDRMConfig) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers, dbEngine sqlengine.SQLEngine) (dto.DBTable, error) {
	tn := tableHeirarchyIDs.GetTableName()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tn) {
		return dto.NewDBTableAnalytics(tn, -1, tableHeirarchyIDs), nil
	}
	return dbEngine.GetCurrentTable(tableHeirarchyIDs)
}

func (dc *StaticDRMConfig) GetTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	return dc.getTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) getTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	unadornedTableName := hIds.GetTableName()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(unadornedTableName) {
		return dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(unadornedTableName)
	}
	return fmt.Sprintf("%s.generation_%d", hIds.GetTableName(), discoveryGenerationID), nil
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

func (dc *StaticDRMConfig) inferTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	return dc.getTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) generateDropTableStatement(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	tableName, err := dc.getTableName(hIds, discoveryGenerationID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`drop table if exists "%s"`, tableName), nil
}

func (dc *StaticDRMConfig) inferColType(col util.Column) string {
	relationalType := "text"
	schema := col.GetSchema()
	if schema != nil && schema.Type != "" {
		relationalType = dc.GetRelationalType(schema.Type)
	}
	return relationalType
}

func (dc *StaticDRMConfig) GenerateDDL(tabAnn util.AnnotatedTabulation, m *openapistackql.OperationStore, discoveryGenerationID int, dropTable bool) ([]string, error) {
	var colDefs, retVal []string
	var rv strings.Builder
	tableName, err := dc.getTableName(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	rv.WriteString(fmt.Sprintf(`create table if not exists "%s" ( `, tableName))
	colDefs = append(colDefs, fmt.Sprintf(`"iql_%s_id" INTEGER PRIMARY KEY AUTOINCREMENT`, tableName))
	genIdColName := dc.getGenerationControlColumn()
	sessionIdColName := dc.getSessionControlColumn()
	txnIdColName := dc.getTxnControlColumn()
	insIdColName := dc.getInsControlColumn()
	lastUpdateColName := dc.getLastUpdatedControlColumn()
	insertEncodedColName := dc.getInsertEncodedControlColumn()
	gcStatusColName := dc.getGCStatusColumn()
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, genIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, sessionIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, txnIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, insIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" TEXT `, insertEncodedColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" DateTime NOT NULL DEFAULT CURRENT_TIMESTAMP `, lastUpdateColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" SMALLINT NOT NULL DEFAULT %d `, gcStatusColName, constants.GCBlack))
	schemaAnalyzer := util.NewTableSchemaAnalyzer(tabAnn.GetTabulation().GetSchema(), m)
	tableColumns := schemaAnalyzer.GetColumns()
	for _, col := range tableColumns {
		var b strings.Builder
		colName := col.GetName()
		colType := dc.inferColType(col)
		b.WriteString(`"` + colName + `" `)
		b.WriteString(dc.GetRelationalType(colType))
		colDefs = append(colDefs, b.String())
	}
	rv.WriteString(strings.Join(colDefs, " , "))
	rv.WriteString(" ) ")
	if dropTable {
		dts, err := dc.generateDropTableStatement(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, dts)
	}
	retVal = append(retVal, rv.String())
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIdColName, tableName, genIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIdColName, tableName, sessionIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIdColName, tableName, txnIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIdColName, tableName, insIdColName))
	return retVal, nil
}

func (dc *StaticDRMConfig) GenerateInsertDML(tabAnnotated util.AnnotatedTabulation, method *openapistackql.OperationStore, tcc *dto.TxnControlCounters) (*PreparedStatementCtx, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("%v", tabulation))
	var q strings.Builder
	var quotedColNames, vals []string
	var columns []ColumnMetadata
	tableName, err := dc.inferTableName(tabAnnotated.GetHeirarchyIdentifiers(), tcc.DiscoveryGenerationId)
	if err != nil {
		return nil, err
	}
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s" `, tableName))
	genIdColName := dc.getGenerationControlColumn()
	sessionIdColName := dc.getSessionControlColumn()
	txnIdColName := dc.getTxnControlColumn()
	insIdColName := dc.getInsControlColumn()
	insEncodedColName := dc.getInsertEncodedControlColumn()
	quotedColNames = append(quotedColNames, `"`+genIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+sessionIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+txnIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+insIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+insEncodedColName+`" `)
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	schemaAnalyzer := util.NewTableSchemaAnalyzer(tabAnnotated.GetTabulation().GetSchema(), method)
	tableColumns := schemaAnalyzer.GetColumnDescriptors(tabAnnotated)
	for _, col := range tableColumns {
		relationalType := "text"
		schema := col.Schema
		if schema != nil && schema.Type != "" {
			relationalType = dc.GetRelationalType(schema.Type)
		}
		columns = append(columns, NewColDescriptor(col, relationalType))
		quotedColNames = append(quotedColNames, `"`+col.Name+`" `)
		vals = append(vals, "?")
	}
	q.WriteString(fmt.Sprintf(" (%s) ", strings.Join(quotedColNames, ", ")))
	q.WriteString(fmt.Sprintf(" VALUES (%s) ", strings.Join(vals, ", ")))
	return NewPreparedStatementCtx(
			q.String(),
			"",
			genIdColName,
			sessionIdColName,
			[]string{tableName},
			txnIdColName,
			insIdColName,
			insEncodedColName,
			columns,
			1,
			tcc,
			nil,
			dc.namespaceCollection,
		),
		nil
}

func (dc *StaticDRMConfig) GenerateSelectDML(tabAnnotated util.AnnotatedTabulation, txnCtrlCtrs *dto.TxnControlCounters, selectSuffix, rewrittenWhere string) (*PreparedStatementCtx, error) {
	var q strings.Builder
	var quotedColNames, quotedWhereColNames []string
	var columns []ColumnMetadata
	// var vals []interface{}
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
		var colEntry strings.Builder
		if col.DecoratedCol == "" {
			colEntry.WriteString(fmt.Sprintf(`"%s" `, col.Name))
			if col.Alias != "" {
				colEntry.WriteString(fmt.Sprintf(` AS "%s"`, col.Alias))
			}
		} else {
			colEntry.WriteString(fmt.Sprintf("%s ", col.DecoratedCol))
		}
		quotedColNames = append(quotedColNames, fmt.Sprintf("%s ", colEntry.String()))

	}
	genIdColName := dc.getGenerationControlColumn()
	sessionIDColName := dc.getSessionControlColumn()
	txnIdColName := dc.getTxnControlColumn()
	insIdColName := dc.getInsControlColumn()
	quotedWhereColNames = append(quotedWhereColNames, `"`+genIdColName+`" `)
	quotedWhereColNames = append(quotedWhereColNames, `"`+txnIdColName+`" `)
	quotedWhereColNames = append(quotedWhereColNames, `"`+insIdColName+`" `)
	aliasStr := ""
	if tabAnnotated.GetAlias() != "" {
		aliasStr = fmt.Sprintf(` AS "%s" `, tabAnnotated.GetAlias())
	}
	tn, err := dc.getTableName(tabAnnotated.GetHeirarchyIdentifiers(), txnCtrlCtrs.DiscoveryGenerationId)
	if err != nil {
		return nil, err
	}
	q.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" %s WHERE `, strings.Join(quotedColNames, ", "), tn, aliasStr))
	q.WriteString(fmt.Sprintf(`( "%s" = ? AND "%s" = ? AND "%s" = ? AND "%s" = ? ) `, genIdColName, sessionIDColName, txnIdColName, insIdColName))
	if strings.TrimSpace(rewrittenWhere) != "" {
		q.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
	}
	q.WriteString(selectSuffix)

	return NewPreparedStatementCtx(
		q.String(),
		"",
		genIdColName,
		sessionIDColName,
		nil,
		txnIdColName,
		insIdColName,
		dc.getInsertEncodedControlColumn(),
		columns,
		1,
		txnCtrlCtrs,
		nil,
		dc.namespaceCollection,
	), nil
}

func (dc *StaticDRMConfig) generateControlVarArgs(cp PreparedStatementParameterized, isInsert bool) ([]interface{}, error) {
	// logging.GetLogger().Infoln(fmt.Sprintf("%v", ctx))
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
			default:
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

func GetGoogleV1SQLiteConfig(namespaceCollection tablenamespace.TableNamespaceCollection) (DRMConfig, error) {
	rv := &StaticDRMConfig{
		typeMappings: map[string]DRMCoupling{
			"array":   {RelationalType: "text", GolangKind: reflect.Slice},
			"boolean": {RelationalType: "boolean", GolangKind: reflect.Bool},
			"int":     {RelationalType: "integer", GolangKind: reflect.Int},
			"integer": {RelationalType: "integer", GolangKind: reflect.Int},
			"object":  {RelationalType: "text", GolangKind: reflect.Map},
			"string":  {RelationalType: "text", GolangKind: reflect.String},
		},
		defaultRelationalType: "text",
		defaultGolangKind:     reflect.String,
		defaultGolangValue:    sql.NullString{}, // string is default
		namespaceCollection:   namespaceCollection,
	}
	return rv, nil
}
