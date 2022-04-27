package drm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/vt/sqlparser"
)

const (
	gen_id_col_name string = "iql_generation_id"
	ssn_id_col_name string = "iql_session_id"
	txn_id_col_name string = "iql_txn_id"
	ins_id_col_name string = "iql_insert_id"
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
	kind                    string // string annotation applicale only in some cases eg UNION [ALL]
	genIdControlColName     string
	sessionIdControlColName string
	TableNames              []string
	txnIdControlColName     string
	insIdControlColName     string
	nonControlColumns       []ColumnMetadata
	ctrlColumnRepeats       int
	txnCtrlCtrs             *dto.TxnControlCounters
	selectTxnCtrlCtrs       []*dto.TxnControlCounters
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
	nonControlColumns []ColumnMetadata,
	ctrlColumnRepeats int,
	txnCtrlCtrs *dto.TxnControlCounters,
	secondaryCtrs []*dto.TxnControlCounters,
) *PreparedStatementCtx {
	return &PreparedStatementCtx{
		query:                   query,
		kind:                    kind,
		genIdControlColName:     genIdControlColName,
		sessionIdControlColName: sessionIdControlColName,
		TableNames:              tableNames,
		txnIdControlColName:     txnIdControlColName,
		insIdControlColName:     insIdControlColName,
		nonControlColumns:       nonControlColumns,
		ctrlColumnRepeats:       ctrlColumnRepeats,
		txnCtrlCtrs:             txnCtrlCtrs,
		selectTxnCtrlCtrs:       secondaryCtrs,
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
	GetCurrentTable(*dto.HeirarchyIdentifiers, sqlengine.SQLEngine) (dto.DBTable, error)
	GetRelationalType(string) string
	GenerateDDL(util.AnnotatedTabulation, int, bool) []string
	GetGolangValue(string) interface{}
	GetInsControlColumn() string
	GetParserTableName(*dto.HeirarchyIdentifiers, int) sqlparser.TableName
	GetSessionControlColumn() string
	GetTableName(*dto.HeirarchyIdentifiers, int) string
	GetTxnControlColumn() string
	GetGenerationControlColumn() string
	GenerateInsertDML(util.AnnotatedTabulation, *dto.TxnControlCounters) (*PreparedStatementCtx, error)
	GenerateSelectDML(util.AnnotatedTabulation, *dto.TxnControlCounters, string, string) (*PreparedStatementCtx, error)
	ExecuteInsertDML(sqlengine.SQLEngine, *PreparedStatementCtx, map[string]interface{}) (sql.Result, error)
	QueryDML(sqlengine.SQLEngine, PreparedStatementParameterized) (*sql.Rows, error)
}

type StaticDRMConfig struct {
	typeMappings          map[string]DRMCoupling
	defaultRelationalType string
	defaultGolangKind     reflect.Kind
	defaultGolangValue    interface{}
}

func (dc *StaticDRMConfig) getDefaultGolangValue() interface{} {
	return &sql.NullString{}
}

func (dc *StaticDRMConfig) getDefaultGolangKind() reflect.Kind {
	return dc.defaultGolangKind
}

func (dc *StaticDRMConfig) GetRelationalType(discoType string) string {
	rv, ok := dc.typeMappings[discoType]
	if ok {
		return rv.RelationalType
	}
	return dc.defaultRelationalType
}

func (dc *StaticDRMConfig) GetGolangValue(discoType string) interface{} {
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

func (dc *StaticDRMConfig) getSessionControlColumn() string {
	return ssn_id_col_name
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

func (dc *StaticDRMConfig) GetCurrentTable(tableHeirarchyIDs *dto.HeirarchyIdentifiers, dbEngine sqlengine.SQLEngine) (dto.DBTable, error) {
	return dbEngine.GetCurrentTable(tableHeirarchyIDs)
}

func (dc *StaticDRMConfig) GetTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) string {
	return dc.getTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) getTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) string {
	return fmt.Sprintf("%s.generation_%d", hIds.GetTableName(), discoveryGenerationID)
}

func (dc *StaticDRMConfig) GetParserTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) sqlparser.TableName {
	return dc.getParserTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) getParserTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) sqlparser.TableName {
	return sqlparser.TableName{
		Name:            sqlparser.NewTableIdent(fmt.Sprintf("generation_%d", discoveryGenerationID)),
		Qualifier:       sqlparser.NewTableIdent(hIds.ResourceStr),
		QualifierSecond: sqlparser.NewTableIdent(hIds.ServiceStr),
		QualifierThird:  sqlparser.NewTableIdent(hIds.ProviderStr),
	}
}

func (dc *StaticDRMConfig) inferTableName(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) string {
	return dc.getTableName(hIds, discoveryGenerationID)
}

func (dc *StaticDRMConfig) generateDropTableStatement(hIds *dto.HeirarchyIdentifiers, discoveryGenerationID int) string {
	return fmt.Sprintf(`drop table if exists "%s"`, dc.getTableName(hIds, discoveryGenerationID))
}

func (dc *StaticDRMConfig) GenerateDDL(tabAnn util.AnnotatedTabulation, discoveryGenerationID int, dropTable bool) []string {
	var colDefs, retVal []string
	var rv strings.Builder
	tableName := dc.getTableName(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
	rv.WriteString(fmt.Sprintf(`create table if not exists "%s" ( `, tableName))
	colDefs = append(colDefs, fmt.Sprintf(`"iql_%s_id" INTEGER PRIMARY KEY AUTOINCREMENT`, tableName))
	genIdColName := dc.getGenerationControlColumn()
	sessionIdColName := dc.getSessionControlColumn()
	txnIdColName := dc.getTxnControlColumn()
	insIdColName := dc.getInsControlColumn()
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, genIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, sessionIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, txnIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, insIdColName))
	for _, col := range tabAnn.GetTabulation().GetColumns() {
		var b strings.Builder
		b.WriteString(`"` + col.Name + `" `)
		b.WriteString(dc.GetRelationalType(col.Schema.Type))
		colDefs = append(colDefs, b.String())
	}
	rv.WriteString(strings.Join(colDefs, " , "))
	rv.WriteString(" ) ")
	if dropTable {
		retVal = append(retVal, dc.generateDropTableStatement(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID))
	}
	retVal = append(retVal, rv.String())
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIdColName, tableName, genIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIdColName, tableName, sessionIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIdColName, tableName, txnIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIdColName, tableName, insIdColName))
	return retVal
}

func (dc *StaticDRMConfig) GenerateInsertDML(tabAnnotated util.AnnotatedTabulation, tcc *dto.TxnControlCounters) (*PreparedStatementCtx, error) {
	// log.Infoln(fmt.Sprintf("%v", tabulation))
	var q strings.Builder
	var quotedColNames, vals []string
	var columns []ColumnMetadata
	tableName := dc.inferTableName(tabAnnotated.GetHeirarchyIdentifiers(), tcc.DiscoveryGenerationId)
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s" `, tableName))
	genIdColName := dc.getGenerationControlColumn()
	sessionIdColName := dc.getSessionControlColumn()
	txnIdColName := dc.getTxnControlColumn()
	insIdColName := dc.getInsControlColumn()
	quotedColNames = append(quotedColNames, `"`+genIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+sessionIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+txnIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+insIdColName+`" `)
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	for _, col := range tabAnnotated.GetTabulation().GetColumns() {
		columns = append(columns, NewColDescriptor(col, dc.GetRelationalType(col.Schema.Type)))
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
			columns,
			1,
			tcc,
			nil,
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
	q.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" %s WHERE `, strings.Join(quotedColNames, ", "), dc.getTableName(tabAnnotated.GetHeirarchyIdentifiers(), txnCtrlCtrs.DiscoveryGenerationId), aliasStr))
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
		columns,
		1,
		txnCtrlCtrs,
		nil,
	), nil
}

func (dc *StaticDRMConfig) generateControlVarArgs(cp PreparedStatementParameterized) ([]interface{}, error) {
	// log.Infoln(fmt.Sprintf("%v", ctx))
	var varArgs []interface{}
	if cp.controlArgsRequired {
		ctrSlice := cp.Ctx.GetAllCtrlCtrs()
		for _, ctrs := range ctrSlice {
			varArgs = append(varArgs, ctrs.GenId)
			varArgs = append(varArgs, ctrs.SessionId)
			varArgs = append(varArgs, ctrs.TxnId)
			varArgs = append(varArgs, ctrs.InsertId)
		}
	}
	return varArgs, nil
}

func (dc *StaticDRMConfig) generateVarArgs(cp PreparedStatementParameterized) (PreparedStatementArgs, error) {
	retVal := NewPreparedStatementArgs(cp.Ctx.GetQuery())
	for i, child := range cp.children {
		chidRv, err := dc.generateVarArgs(child)
		if err != nil {
			return retVal, err
		}
		retVal.children[i] = chidRv
	}
	varArgs, _ := dc.generateControlVarArgs(cp)
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

func (dc *StaticDRMConfig) ExecuteInsertDML(dbEngine sqlengine.SQLEngine, ctx *PreparedStatementCtx, payload map[string]interface{}) (sql.Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cannot execute on nil PreparedStatementContext")
	}
	stmtArgs, err := dc.generateVarArgs(PreparedStatementParameterized{Ctx: ctx, args: payload, controlArgsRequired: true})
	if err != nil {
		return nil, err
	}
	return dbEngine.Exec(stmtArgs.query, stmtArgs.args...)
}

func (dc *StaticDRMConfig) QueryDML(dbEngine sqlengine.SQLEngine, ctxParameterized PreparedStatementParameterized) (*sql.Rows, error) {
	if ctxParameterized.Ctx == nil {
		return nil, fmt.Errorf("cannot execute based upon nil PreparedStatementContext")
	}
	rootArgs, err := dc.generateVarArgs(ctxParameterized)
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
		log.Infoln(fmt.Sprintf("adding child query = %s", cp.query))
		childQueryStrings = append(childQueryStrings, cp.query)
		if len(rootArgs.args) >= k {
			varArgs = append(varArgs, rootArgs.args[j:k]...)
		}
		varArgs = append(varArgs, cp.args...)
		j = k
	}
	log.Infoln(fmt.Sprintf("raw query = %s", query))
	if len(childQueryStrings) > 0 {
		query = fmt.Sprintf(rootArgs.query, childQueryStrings...)
	}
	if len(rootArgs.args) >= j {
		varArgs = append(varArgs, rootArgs.args[j:]...)
	}
	log.Infoln(fmt.Sprintf("query = %s", query))
	return dbEngine.Query(query, varArgs...)
}

func GetGoogleV1SQLiteConfig() DRMConfig {
	return &StaticDRMConfig{
		typeMappings: map[string]DRMCoupling{
			"array":   DRMCoupling{RelationalType: "text", GolangKind: reflect.Slice},
			"boolean": DRMCoupling{RelationalType: "boolean", GolangKind: reflect.Bool},
			"int":     DRMCoupling{RelationalType: "integer", GolangKind: reflect.Int},
			"integer": DRMCoupling{RelationalType: "integer", GolangKind: reflect.Int},
			"object":  DRMCoupling{RelationalType: "text", GolangKind: reflect.Map},
			"string":  DRMCoupling{RelationalType: "text", GolangKind: reflect.String},
		},
		defaultRelationalType: "text",
		defaultGolangKind:     reflect.String,
		defaultGolangValue:    sql.NullString{}, // string is default
	}
}

type GoogleV1DRM struct {
}
