package sqldialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/relationaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"vitess.io/vitess/go/vt/sqlparser"
)

func newPostgresDialect(sqlEngine sqlengine.SQLEngine, analyticsNamespaceLikeString string, controlAttributes sqlcontrol.ControlAttributes, formatter sqlparser.NodeFormatter, sqlCfg dto.SQLBackendCfg) (SQLDialect, error) {
	catalogName, err := sqlCfg.GetDatabaseName()
	if err != nil {
		return nil, err
	}
	tableSchemaName := sqlCfg.GetTableSchemaName()
	if tableSchemaName == "" {
		tableSchemaName = "public"
	}
	rv := &postgresDialect{
		defaultGolangKind:     reflect.String,
		defaultRelationalType: "text",
		typeMappings: map[string]internaldto.DRMCoupling{
			"array":   internaldto.NewDRMCoupling("text", reflect.Slice),
			"boolean": internaldto.NewDRMCoupling("boolean", reflect.Bool),
			"int":     internaldto.NewDRMCoupling("bigint", reflect.Int64),
			"integer": internaldto.NewDRMCoupling("bigint", reflect.Int64),
			"object":  internaldto.NewDRMCoupling("text", reflect.Map),
			"string":  internaldto.NewDRMCoupling("text", reflect.String),
			"number":  internaldto.NewDRMCoupling("numeric", reflect.Float64),
		},
		controlAttributes:            controlAttributes,
		analyticsNamespaceLikeString: analyticsNamespaceLikeString,
		sqlEngine:                    sqlEngine,
		formatter:                    formatter,
		tableSchema:                  tableSchemaName,
		tableCatalog:                 catalogName,
	}
	viewSchemataEnabled, err := rv.inferViewSchemataEnabled(sqlCfg.Schemata)
	if err != nil {
		return nil, err
	}
	if viewSchemataEnabled {
		rv.viewSchemataEnabled = viewSchemataEnabled
		rv.tableSchema = sqlCfg.GetTableSchemaName()
		rv.opsViewSchema = sqlCfg.GetOpsViewSchemaName()
		rv.intelViewSchema = sqlCfg.GetIntelViewSchemaName()
	}
	err = rv.initPostgresEngine()
	if err != nil {
		return nil, err
	}
	return rv, nil
}

func (eng *postgresDialect) inferViewSchemataEnabled(schemataCfg dto.SQLBackendSchemata) (bool, error) {
	if schemataCfg.TableSchema == "" || schemataCfg.OpsViewSchema == "" || schemataCfg.IntelViewSchema == "" {
		return false, nil
	}
	return true, nil
}

type postgresDialect struct {
	controlAttributes            sqlcontrol.ControlAttributes
	analyticsNamespaceLikeString string
	sqlEngine                    sqlengine.SQLEngine
	formatter                    sqlparser.NodeFormatter
	typeMappings                 map[string]internaldto.DRMCoupling
	defaultRelationalType        string
	defaultGolangKind            reflect.Kind
	tableSchema                  string
	viewSchemataEnabled          bool
	opsViewSchema                string
	intelViewSchema              string
	tableCatalog                 string
}

func (eng *postgresDialect) initPostgresEngine() error {
	_, err := eng.sqlEngine.Exec(postgresEngineSetupDDL)
	return err
}

func (eng *postgresDialect) generateDropTableStatement(relationalTable relationaldto.RelationalTable) (string, error) {
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`drop table if exists "%s"`, tableName), nil
}

func (eng *postgresDialect) GetFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return eng.getFullyQualifiedTableName(unqualifiedTableName)
}

func (eng *postgresDialect) getFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return fmt.Sprintf(`"%s"."%s"`, eng.tableSchema, unqualifiedTableName), nil
}

func (sl *postgresDialect) GetASTFormatter() sqlparser.NodeFormatter {
	return sl.formatter
}

func (sl *postgresDialect) GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter {
	return astfuncrewrite.GetPostgresASTFuncRewriter()
}

func (eng *postgresDialect) GenerateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
	return eng.generateDDL(relationalTable, dropTable)
}

func (eng *postgresDialect) SanitizeQueryString(queryString string) (string, error) {
	return eng.sanitizeQueryString(queryString)
}

func (eng *postgresDialect) sanitizeQueryString(queryString string) (string, error) {
	return strings.ReplaceAll(queryString, "`", `"`), nil
}

func (eng *postgresDialect) SanitizeWhereQueryString(queryString string) (string, error) {
	return eng.sanitizeWhereQueryString(queryString)
}

func (eng *postgresDialect) sanitizeWhereQueryString(queryString string) (string, error) {
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(queryString, "`", `"`),
			"||", "OR",
		),
		"|", "||",
	), nil
}

func (eng *postgresDialect) generateViewDDL(srcSchemaName string, destSchemaName string, relationalTable relationaldto.RelationalTable) ([]string, error) {
	var colNames, retVal []string
	var createViewBuilder strings.Builder
	retVal = append(retVal, fmt.Sprintf(`drop view if exists "%s"."%s" ; `, destSchemaName, relationalTable.GetBaseName()))
	createViewBuilder.WriteString(fmt.Sprintf(`create or replace view "%s"."%s" AS `, destSchemaName, relationalTable.GetBaseName()))
	for _, col := range relationalTable.GetColumns() {
		var b strings.Builder
		colName := col.GetName()
		b.WriteString(`"` + colName + `" `)
		colNames = append(colNames, b.String())
	}
	tableName, err := relationalTable.GetName()
	if err != nil {
		return nil, err
	}
	createViewBuilder.WriteString(fmt.Sprintf(`select %s from "%s"."%s" ;`, strings.Join(colNames, ", "), srcSchemaName, tableName))
	retVal = append(retVal, createViewBuilder.String())
	return retVal, nil
}

func (eng *postgresDialect) generateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
	var colDefs, retVal []string
	if dropTable {
		dt, err := eng.generateDropTableStatement(relationalTable)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, dt)
	}
	var rv strings.Builder
	tableName, err := relationalTable.GetName()
	if err != nil {
		return nil, err
	}
	rv.WriteString(fmt.Sprintf(`create table if not exists "%s"."%s" ( `, eng.tableSchema, tableName))
	colDefs = append(colDefs, fmt.Sprintf(`"iql_%s_id" BIGSERIAL PRIMARY KEY`, tableName))
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	sessionIdColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	maxTxnIdColName := eng.controlAttributes.GetControlMaxTxnColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	lastUpdateColName := eng.controlAttributes.GetControlLatestUpdateColumnName()
	insertEncodedColName := eng.controlAttributes.GetControlInsertEncodedIdColumnName()
	gcStatusColName := eng.controlAttributes.GetControlGCStatusColumnName()
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, genIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, sessionIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, txnIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, maxTxnIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, insIdColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" TEXT `, insertEncodedColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP `, lastUpdateColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" SMALLINT NOT NULL DEFAULT %d `, gcStatusColName, constants.GCBlack))
	for _, col := range relationalTable.GetColumns() {
		var b strings.Builder
		colName := col.GetName()
		colType := col.GetType()
		b.WriteString(`"` + colName + `" `)
		b.WriteString(colType)
		colDefs = append(colDefs, b.String())
	}
	rv.WriteString(strings.Join(colDefs, " , "))
	rv.WriteString(" ) ")
	retVal = append(retVal, rv.String())
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIdColName, eng.tableSchema, tableName, genIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIdColName, eng.tableSchema, tableName, sessionIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIdColName, eng.tableSchema, tableName, txnIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIdColName, eng.tableSchema, tableName, insIdColName))
	rawViewDDL, err := eng.generateViewDDL(eng.tableSchema, eng.tableSchema, relationalTable)
	if err != nil {
		return nil, err
	}
	retVal = append(retVal, rawViewDDL...)
	if eng.viewSchemataEnabled {
		intelViewDDL, err := eng.generateViewDDL(eng.tableSchema, eng.intelViewSchema, relationalTable)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, intelViewDDL...)
	}
	return retVal, nil
}

func (eng *postgresDialect) CreateView(viewName string, rawDDL string, translatedDDL string) error {
	return eng.createView(viewName, rawDDL, translatedDDL)
}

func (eng *postgresDialect) createView(viewName string, rawDDL string, translatedDDL string) error {
	return nil
}

func (eng *postgresDialect) ViewExists(viewName string) bool {
	return eng.viewExists(viewName)
}

func (eng *postgresDialect) viewExists(viewName string) bool {
	q := `SELECT count(*) AS view_count FROM "__iql__.views" WHERE view_name = $1 and deleted_dttm IS NULL`
	row := eng.sqlEngine.QueryRow(q, viewName)
	if row != nil {
		var viewCount int
		err := row.Scan(&viewCount)
		if err != nil {
			return false
		}
		if viewCount == 1 {
			return true
		}
	}
	return false
}

func (eng *postgresDialect) GetGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	return eng.getGCHousekeepingQuery(tableName, tcc)
}

func (eng *postgresDialect) getGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	templateQuery := `INSERT INTO 
	  "__iql__.control.gc.txn_table_x_ref" (
			iql_generation_id, 
			iql_session_id, 
			iql_transaction_id, 
			table_name
		) values(%d, %d, %d, '%s')
		ON CONFLICT (iql_generation_id, iql_session_id, iql_transaction_id, table_name) DO NOTHING
		`
	return fmt.Sprintf(templateQuery, tcc.GetGenID(), tcc.GetSessionID(), tcc.GetTxnID(), tableName)
}

func (eng *postgresDialect) DelimitGroupByColumn(term string) string {
	return eng.quoteWrapTerm(term)
}

func (eng *postgresDialect) DelimitOrderByColumn(term string) string {
	return eng.quoteWrapTerm(term)
}

func (eng *postgresDialect) quoteWrapTerm(term string) string {
	return fmt.Sprintf(`"%s"`, term)
}

func (eng *postgresDialect) ComposeSelectQuery(columns []relationaldto.RelationalColumn, tableAliases []string, fromString string, rewrittenWhere string, selectSuffix string) (string, error) {
	return eng.composeSelectQuery(columns, tableAliases, fromString, rewrittenWhere, selectSuffix)
}

func (eng *postgresDialect) composeSelectQuery(columns []relationaldto.RelationalColumn, tableAliases []string, fromString string, rewrittenWhere string, selectSuffix string) (string, error) {
	var q strings.Builder
	var quotedColNames []string
	for _, col := range columns {
		quotedColNames = append(quotedColNames, col.DelimitedSelectionString(`"`))
	}
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	var wq strings.Builder
	var controlWhereComparisons []string
	i := 0
	for _, alias := range tableAliases {
		j := i * 4
		if alias != "" {
			gIDcn := fmt.Sprintf(`"%s"."%s"`, alias, genIdColName)
			sIDcn := fmt.Sprintf(`"%s"."%s"`, alias, sessionIDColName)
			tIDcn := fmt.Sprintf(`"%s"."%s"`, alias, txnIdColName)
			iIDcn := fmt.Sprintf(`"%s"."%s"`, alias, insIdColName)
			controlWhereComparisons = append(controlWhereComparisons, fmt.Sprintf(`%s = $%d AND %s = $%d AND %s = $%d AND %s = $%d`, gIDcn, j+1, sIDcn, j+2, tIDcn, j+3, iIDcn, j+4))
		} else {
			gIDcn := fmt.Sprintf(`"%s"`, genIdColName)
			sIDcn := fmt.Sprintf(`"%s"`, sessionIDColName)
			tIDcn := fmt.Sprintf(`"%s"`, txnIdColName)
			iIDcn := fmt.Sprintf(`"%s"`, insIdColName)
			controlWhereComparisons = append(controlWhereComparisons, fmt.Sprintf(`%s = $%d AND %s = $%d AND %s = $%d AND %s = $%d`, gIDcn, j+1, sIDcn, j+2, tIDcn, j+3, iIDcn, j+4))
		}
		i++
	}
	if len(controlWhereComparisons) > 0 {
		controlWhereSubClause := fmt.Sprintf("( %s )", strings.Join(controlWhereComparisons, " AND "))
		wq.WriteString(controlWhereSubClause)
	}

	if strings.TrimSpace(rewrittenWhere) != "" {
		if len(controlWhereComparisons) > 0 {
			wq.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
		} else {
			wq.WriteString(fmt.Sprintf(" ( %s ) ", rewrittenWhere))
		}
	}
	whereExprsStr := wq.String()

	q.WriteString(fmt.Sprintf(`SELECT %s FROM `, strings.Join(quotedColNames, ", ")))
	q.WriteString(fromString)
	if whereExprsStr != "" {
		q.WriteString(" WHERE ")
		q.WriteString(whereExprsStr)
	}
	q.WriteString(selectSuffix)

	query := q.String()

	return eng.sanitizeQueryString(query)
}

func (eng *postgresDialect) GenerateInsertDML(relationalTable relationaldto.RelationalTable, tcc internaldto.TxnControlCounters) (string, error) {
	return eng.generateInsertDML(relationalTable, tcc)
}

func (eng *postgresDialect) generateInsertDML(relationalTable relationaldto.RelationalTable, tcc internaldto.TxnControlCounters) (string, error) {
	var q strings.Builder
	var quotedColNames, vals []string
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s"."%s" `, eng.tableSchema, tableName))
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	sessionIdColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	insEncodedColName := eng.controlAttributes.GetControlInsertEncodedIdColumnName()
	quotedColNames = append(quotedColNames, `"`+genIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+sessionIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+txnIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+insIdColName+`" `)
	quotedColNames = append(quotedColNames, `"`+insEncodedColName+`" `)
	vals = append(vals, "$1")
	vals = append(vals, "$2")
	vals = append(vals, "$3")
	vals = append(vals, "$4")
	vals = append(vals, "$5")
	i := 1
	for _, col := range relationalTable.GetColumns() {
		quotedColNames = append(quotedColNames, `"`+col.GetName()+`" `)
		if strings.ToLower(col.GetType()) != "text" {
			vals = append(vals, fmt.Sprintf("$%d", 5+i))
		} else {
			vals = append(vals, fmt.Sprintf("CAST($%d AS TEXT)", 5+i))
		}
		i++
	}
	q.WriteString(fmt.Sprintf(" (%s) ", strings.Join(quotedColNames, ", ")))
	q.WriteString(fmt.Sprintf(" VALUES (%s) ", strings.Join(vals, ", ")))
	return q.String(), nil
}

func (eng *postgresDialect) GenerateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
	return eng.generateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)
}

func (eng *postgresDialect) generateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
	var q strings.Builder
	var quotedColNames []string
	for _, col := range relationalTable.GetColumns() {
		var colEntry strings.Builder
		if col.GetDecorated() == "" {
			colEntry.WriteString(fmt.Sprintf(`"%s" `, col.GetName()))
			if col.GetAlias() != "" {
				colEntry.WriteString(fmt.Sprintf(` AS "%s"`, col.GetAlias()))
			}
		} else {
			colEntry.WriteString(fmt.Sprintf("%s ", col.GetDecorated()))
		}
		quotedColNames = append(quotedColNames, fmt.Sprintf("%s ", colEntry.String()))

	}
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	aliasStr := ""
	if relationalTable.GetAlias() != "" {
		aliasStr = fmt.Sprintf(` AS "%s" `, relationalTable.GetAlias())
	}
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	q.WriteString(fmt.Sprintf(`SELECT %s FROM "%s"."%s" %s WHERE `, strings.Join(quotedColNames, ", "), eng.tableCatalog, tableName, aliasStr))
	q.WriteString(fmt.Sprintf(`( "%s" = $1 AND "%s" = $2 AND "%s" = $3 AND "%s" = $4 ) `, genIdColName, sessionIDColName, txnIdColName, insIdColName))
	if strings.TrimSpace(rewrittenWhere) != "" {
		q.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
	}
	q.WriteString(selectSuffix)

	return q.String(), nil
}

func (sl *postgresDialect) GCAdd(tableName string, parentTcc, lockableTcc internaldto.TxnControlCounters) error {
	maxTxnColName := sl.controlAttributes.GetControlMaxTxnColumnName()
	q := fmt.Sprintf(
		`
		UPDATE "%s" 
		SET "%s" = r.current_value
		FROM (
			SELECT *
			FROM
				"__iql__.control.gc.rings"
		) AS r
		WHERE 
			"%s" = $1 
			AND 
			"%s" = $2 
			AND
			r.ring_name = 'transaction_id'
			AND
			"%s" < CASE 
			   WHEN ("%s" - r.current_offset) < 0
				 THEN CAST(pow(2, r.width_bits) + ("%s" - r.current_offset)  AS int)
				 ELSE "%s" - r.current_offset
				 END
		`,
		tableName,
		maxTxnColName,
		sl.controlAttributes.GetControlTxnIdColumnName(),
		sl.controlAttributes.GetControlInsIdColumnName(),
		maxTxnColName,
		maxTxnColName,
		maxTxnColName,
		maxTxnColName,
	)
	_, err := sl.sqlEngine.Exec(q, lockableTcc.GetTxnID(), lockableTcc.GetInsertID())
	return err
}

func (sl *postgresDialect) GCCollectObsoleted(minTransactionID int) error {
	return sl.gCCollectObsoleted(minTransactionID)
}

func (sl *postgresDialect) gCCollectObsoleted(minTransactionID int) error {
	maxTxnColName := sl.controlAttributes.GetControlMaxTxnColumnName()
	obtainQuery := fmt.Sprintf(
		`
		SELECT
			'DELETE FROM "%s"."' || table_name || '" WHERE "%s" < %d ; '
		from 
			information_schema.tables 
		where 
			table_type = 'BASE TABLE' 
			and 
			table_catalog = $1
			and 
			table_schema = $2
		  and
			table_name not like '__iql__%%'
		`,
		sl.tableSchema,
		maxTxnColName,
		minTransactionID,
	)
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.tableCatalog, sl.tableSchema)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *postgresDialect) GCCollectAll() error {
	return sl.gCCollectAll()
}

func (sl *postgresDialect) GetOperatorOr() string {
	return "OR"
}

func (sl *postgresDialect) GetOperatorStringConcat() string {
	return "||"
}

func (sl *postgresDialect) gCCollectAll() error {
	obtainQuery := fmt.Sprintf(`
		SELECT
			'DELETE FROM "%s"."' || table_name || '"  ; '
		from 
			information_schema.tables 
		where 
			table_type = 'BASE TABLE' 
			and 
			table_catalog = $1
			and 
			table_schema = $2
		  and
			table_name not like '__iql__%%'
		`,
		sl.tableSchema,
	)
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.tableCatalog, sl.tableSchema)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *postgresDialect) GCControlTablesPurge() error {
	return sl.gcControlTablesPurge()
}

func (eng *postgresDialect) IsTablePresent(tableName string, requestEncoding string, colName string) bool {
	rows, err := eng.sqlEngine.Query(fmt.Sprintf(`SELECT count(*) as ct FROM "%s"."%s" WHERE iql_insert_encoded = $1 `, eng.tableSchema, tableName), requestEncoding)
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var ct int
			rows.Scan(&ct)
			if ct > 0 {
				return true
			}
		}
	}
	return false
}

// In Postgres, `Timestamp with time zone` objects are timezone-aware.
func (eng *postgresDialect) TableOldestUpdateUTC(tableName string, requestEncoding string, updateColName string, requestEncodingColName string) (time.Time, internaldto.TxnControlCounters) {
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	ssnIdColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	rows, err := eng.sqlEngine.Query(fmt.Sprintf("SELECT min(%s) as oldest_update, %s, %s, %s, %s FROM \"%s\".\"%s\" WHERE %s = '%s' GROUP BY %s, %s, %s, %s;", updateColName, genIdColName, ssnIdColName, txnIdColName, insIdColName, eng.tableSchema, tableName, requestEncodingColName, requestEncoding, genIdColName, ssnIdColName, txnIdColName, insIdColName))
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var oldestTime time.Time
			var genID, sessionID, txnID, insertID int
			err = rows.Scan(&oldestTime, &genID, &sessionID, &txnID, &insertID)
			if err == nil {
				tcc := internaldto.NewTxnControlCountersFromVals(genID, sessionID, txnID, insertID)
				tcc.SetTableName(tableName)
				return oldestTime, tcc
			}
		}
	}
	return time.Time{}, nil
}

func (sl *postgresDialect) gcControlTablesPurge() error {
	obtainQuery := fmt.Sprintf(`
		SELECT
		  'DELETE FROM "%s"."' || table_name || '" ; '
			from 
			information_schema.tables 
		where 
			table_type = 'BASE TABLE' 
			and 
			table_catalog = $1
			and 
			table_schema = $2
		  and
			table_name like '__iql__%%'
		`,
		sl.tableSchema,
	)
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.tableCatalog, sl.tableSchema)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *postgresDialect) GCPurgeEphemeral() error {
	return sl.gcPurgeEphemeral()
}

func (sl *postgresDialect) GCPurgeCache() error {
	return sl.gcPurgeCache()
}

func (sl *postgresDialect) GetName() string {
	return constants.SQLDialectPostgres
}

func (sl *postgresDialect) gcPurgeCache() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || table_name || '" ; ' 
	from 
		information_schema.tables 
	where 
		table_type = 'BASE TABLE' 
		and 
		table_catalog = $1
		and 
		table_schema = $2
		and 
		table_name like $3
	`
	rows, err := sl.sqlEngine.Query(query, sl.tableCatalog, sl.tableSchema, sl.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(rows)
}

func (sl *postgresDialect) gcPurgeEphemeral() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || table_name || '" ; ' 
	from 
		information_schema.tables 
	where 
		table_type = 'BASE TABLE' 
		and 
		table_catalog = $1
		and 
		table_schema = $2
		and 
		table_name NOT like $3
		and 
		table_name not like '__iql__%' 
	`
	rows, err := sl.sqlEngine.Query(query, sl.tableCatalog, sl.tableSchema, sl.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(rows)
}

func (sl *postgresDialect) PurgeAll() error {
	return sl.purgeAll()
}

func (sl *postgresDialect) GetSQLEngine() sqlengine.SQLEngine {
	return sl.sqlEngine
}

func (sl *postgresDialect) purgeAll() error {
	obtainQuery := `
		SELECT
			'DROP TABLE IF EXISTS "' || table_name || '" ; '
		from 
			information_schema.tables 
		where 
			table_type = 'BASE TABLE' 
			and 
			table_catalog = $1 
			and 
			table_schema = $2
		  AND
			table_name NOT LIKE '__iql__%'
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.tableCatalog, sl.tableSchema)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *postgresDialect) readExecGeneratedQueries(queryResultSet *sql.Rows) error {
	defer queryResultSet.Close()
	var queries []string
	for {
		hasNext := queryResultSet.Next()
		if !hasNext {
			break
		}
		var s string
		err := queryResultSet.Scan(&s)
		if err != nil {
			return err
		}
		queries = append(queries, s)
	}
	err := sl.sqlEngine.ExecInTxn(queries)
	return err
}

func (eng *postgresDialect) GetRelationalType(discoType string) string {
	return eng.getRelationalType(discoType)
}

func (eng *postgresDialect) getRelationalType(discoType string) string {
	rv, ok := eng.typeMappings[discoType]
	if ok {
		return rv.GetRelationalType()
	}
	return eng.defaultRelationalType
}

func (eng *postgresDialect) GetGolangValue(discoType string) interface{} {
	return eng.getGolangValue(discoType)
}

func (eng *postgresDialect) getGolangValue(discoType string) interface{} {
	rv, ok := eng.typeMappings[discoType]
	if !ok {
		return eng.getDefaultGolangValue()
	}
	switch rv.GetGolangKind() {
	case reflect.String:
		return &sql.NullString{}
	case reflect.Array:
		return &sql.NullString{}
	case reflect.Bool:
		return &sql.NullBool{}
	case reflect.Map:
		return &sql.NullString{}
	case reflect.Int, reflect.Int64:
		return &sql.NullInt64{}
	case reflect.Float64:
		return &sql.NullFloat64{}
	}
	return eng.getDefaultGolangValue()
}

func (eng *postgresDialect) getDefaultGolangValue() interface{} {
	return &sql.NullString{}
}

func (eng *postgresDialect) GetGolangKind(discoType string) reflect.Kind {
	rv, ok := eng.typeMappings[discoType]
	if !ok {
		return eng.getDefaultGolangKind()
	}
	return rv.GetGolangKind()
}

func (eng *postgresDialect) getDefaultGolangKind() reflect.Kind {
	return eng.defaultGolangKind
}

func (eng *postgresDialect) QueryNamespaced(colzString string, actualTableName string, requestEncodingColName string, requestEncoding string) (*sql.Rows, error) {
	return eng.sqlEngine.Query(fmt.Sprintf(`SELECT %s FROM "%s"."%s" WHERE "%s" = $1`, colzString, eng.tableSchema, actualTableName, requestEncodingColName), requestEncoding)
}

func (se *postgresDialect) GetTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers, discoveryId int) (internaldto.DBTable, error) {
	return se.getTable(tableHeirarchyIDs, discoveryId)
}

func (se *postgresDialect) getTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers, discoveryId int) (internaldto.DBTable, error) {
	tableNameStump, err := se.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableName := fmt.Sprintf("%s.generation_%d", tableNameStump, discoveryId)
	return internaldto.NewDBTable(tableName, tableNameStump, tableHeirarchyIDs.GetTableName(), discoveryId, tableHeirarchyIDs), err
}

func (se *postgresDialect) GetCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

// In postgres, 63 chars is default length for IDs such as table names
// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS
func (se *postgresDialect) getTableNameStump(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (string, error) {
	rawTableName := tableHeirarchyIDs.GetTableName()
	maxRawTableNameWidth := constants.PostgresIDMaxWidth - (len(".generation_") + constants.MaxDigits32BitUnsigned)
	if len(rawTableName) > maxRawTableNameWidth {
		return rawTableName[:maxRawTableNameWidth], nil
	}
	return rawTableName, nil
}

func (se *postgresDialect) getCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	var tableName string
	var discoID int
	tableNameStump, err := se.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableNameStump)
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableNameStump)
	res := se.sqlEngine.QueryRow(`
	select 
		table_name, 
		CAST(REPLACE(table_name, $1, '') AS INTEGER) 
	from 
		information_schema.tables 
	where 
		table_type = 'BASE TABLE'
	  and 
		table_name like $2 
	ORDER BY table_name DESC 
	limit 1
	`, tableNameLHSRemove, tableNamePattern)
	err = res.Scan(&tableName, &discoID)
	if err != nil {
		logging.GetLogger().Errorln(fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'", err, tableNamePattern, tableNameLHSRemove))
	}
	return internaldto.NewDBTable(tableName, tableNameStump, tableHeirarchyIDs.GetTableName(), discoID, tableHeirarchyIDs), err
}
