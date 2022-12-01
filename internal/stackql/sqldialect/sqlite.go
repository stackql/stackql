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

func newSQLiteDialect(sqlEngine sqlengine.SQLEngine, analyticsNamespaceLikeString string, controlAttributes sqlcontrol.ControlAttributes, formatter sqlparser.NodeFormatter, sqlCfg dto.SQLBackendCfg) (SQLDialect, error) {
	rv := &sqLiteDialect{
		defaultGolangKind:     reflect.String,
		defaultRelationalType: "text",
		typeMappings: map[string]internaldto.DRMCoupling{
			"array":   internaldto.NewDRMCoupling("text", reflect.Slice),
			"boolean": internaldto.NewDRMCoupling("boolean", reflect.Bool),
			"int":     internaldto.NewDRMCoupling("integer", reflect.Int),
			"integer": internaldto.NewDRMCoupling("integer", reflect.Int),
			"object":  internaldto.NewDRMCoupling("text", reflect.Map),
			"string":  internaldto.NewDRMCoupling("text", reflect.String),
		},
		controlAttributes:            controlAttributes,
		analyticsNamespaceLikeString: analyticsNamespaceLikeString,
		sqlEngine:                    sqlEngine,
		formatter:                    formatter,
	}
	err := rv.initSQLiteEngine()
	return rv, err
}

type sqLiteDialect struct {
	controlAttributes            sqlcontrol.ControlAttributes
	analyticsNamespaceLikeString string
	sqlEngine                    sqlengine.SQLEngine
	formatter                    sqlparser.NodeFormatter
	typeMappings                 map[string]internaldto.DRMCoupling
	defaultRelationalType        string
	defaultGolangKind            reflect.Kind
	defaultGolangValue           interface{}
}

func (eng *sqLiteDialect) initSQLiteEngine() error {
	_, err := eng.sqlEngine.Exec(sqLiteEngineSetupDDL)
	return err
}

func (se *sqLiteDialect) GetTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers, discoveryId int) (internaldto.DBTable, error) {
	return se.getTable(tableHeirarchyIDs, discoveryId)
}

func (se *sqLiteDialect) getTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers, discoveryId int) (internaldto.DBTable, error) {
	tableNameStump, err := se.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableName := fmt.Sprintf("%s.generation_%d", tableNameStump, discoveryId)
	return internaldto.NewDBTable(tableName, tableNameStump, tableHeirarchyIDs.GetTableName(), discoveryId, tableHeirarchyIDs), err
}

func (se *sqLiteDialect) GetCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

func (se *sqLiteDialect) getTableNameStump(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (string, error) {
	return tableHeirarchyIDs.GetTableName(), nil
}

func (se *sqLiteDialect) getCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	var tableName string
	var discoID int
	tableNameStump, err := se.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableNameStump)
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableNameStump)
	res := se.sqlEngine.QueryRow(`select name, CAST(REPLACE(name, ?, '') AS INTEGER) from sqlite_schema where type = 'table' and name like ? ORDER BY name DESC limit 1`, tableNameLHSRemove, tableNamePattern)
	err = res.Scan(&tableName, &discoID)
	if err != nil {
		logging.GetLogger().Errorln(fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'", err, tableNamePattern, tableNameLHSRemove))
	}
	return internaldto.NewDBTable(tableName, tableNameStump, tableHeirarchyIDs.GetTableName(), discoID, tableHeirarchyIDs), err
}

func (sl *sqLiteDialect) GetName() string {
	return constants.SQLDialectSQLite3
}

func (sl *sqLiteDialect) GetASTFormatter() sqlparser.NodeFormatter {
	return sl.formatter
}

func (sl *sqLiteDialect) GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter {
	return astfuncrewrite.GetNopFuncRewriter()
}

func (sl *sqLiteDialect) GCAdd(tableName string, parentTcc, lockableTcc internaldto.TxnControlCounters) error {
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
			"%s" = ? 
			AND 
			"%s" = ? 
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

func (sl *sqLiteDialect) GCCollectObsoleted(minTransactionID int) error {
	return sl.gCCollectObsoleted(minTransactionID)
}

func (sl *sqLiteDialect) gCCollectObsoleted(minTransactionID int) error {
	maxTxnColName := sl.controlAttributes.GetControlMaxTxnColumnName()
	obtainQuery := fmt.Sprintf(
		`
		SELECT
			'DELETE FROM "' || name || '" WHERE "%s" < %d ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
		  and
			name not like '__iql__%%' 
			and
			name NOT LIKE 'sqlite_%%' 
		`,
		maxTxnColName,
		minTransactionID,
	)
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *sqLiteDialect) GCCollectAll() error {
	return sl.gCCollectAll()
}

func (sl *sqLiteDialect) GetSQLEngine() sqlengine.SQLEngine {
	return sl.sqlEngine
}

func (sl *sqLiteDialect) gCCollectAll() error {
	obtainQuery := `
		SELECT
			'DELETE FROM "' || name || '"  ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
		  and
			name not like '__iql__%%' 
			and
			name NOT LIKE 'sqlite_%%' 
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *sqLiteDialect) generateDropTableStatement(relationalTable relationaldto.RelationalTable) (string, error) {
	s, err := relationalTable.GetName()
	return fmt.Sprintf(`drop table if exists "%s"`, s), err
}

func (sl *sqLiteDialect) GCControlTablesPurge() error {
	return sl.gcControlTablesPurge()
}

func (eng *sqLiteDialect) GenerateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
	return eng.generateDDL(relationalTable, dropTable)
}

func (eng *sqLiteDialect) generateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
	var colDefs, retVal []string
	var rv strings.Builder
	if dropTable {
		dt, err := eng.generateDropTableStatement(relationalTable)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, dt)
	}
	tableName, err := relationalTable.GetName()
	if err != nil {
		return nil, err
	}
	rv.WriteString(fmt.Sprintf(`create table if not exists "%s" ( `, tableName))
	colDefs = append(colDefs, fmt.Sprintf(`"iql_%s_id" INTEGER PRIMARY KEY AUTOINCREMENT`, tableName))
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
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIdColName, tableName, genIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIdColName, tableName, sessionIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIdColName, tableName, txnIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIdColName, tableName, insIdColName))
	return retVal, nil
}

func (eng *sqLiteDialect) IsTablePresent(tableName string, requestEncoding string, colName string) bool {
	rows, err := eng.sqlEngine.Query(fmt.Sprintf(`SELECT count(*) as ct FROM "%s" WHERE iql_insert_encoded=?;`, tableName), requestEncoding)
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

// In SQLite, `DateTime` objects are not properly aware; the zone is not recorded.
// That being said, those fields populated with `DateTime('now')` are UTC.
// As per https://www.sqlite.org/lang_datefunc.html:
//
//	The 'now' argument to date and time functions always returns exactly
//	the same value for multiple invocations within the same sqlite3_step()
//	call. Universal Coordinated Time (UTC) is used.
//
// Therefore, this method will behave correctly provided that the column `colName`
// is populated with `DateTime('now')`.
func (eng *sqLiteDialect) TableOldestUpdateUTC(tableName string, requestEncoding string, updateColName string, requestEncodingColName string) (time.Time, internaldto.TxnControlCounters) {
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	ssnIdColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	rows, err := eng.sqlEngine.Query(fmt.Sprintf("SELECT strftime('%%Y-%%m-%%dT%%H:%%M:%%S', min(%s)) as oldest_update, %s, %s, %s, %s FROM \"%s\" WHERE %s = '%s';", updateColName, genIdColName, ssnIdColName, txnIdColName, insIdColName, tableName, requestEncodingColName, requestEncoding))
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var oldest string
			var genID, sessionID, txnID, insertID int
			err = rows.Scan(&oldest, &genID, &sessionID, &txnID, &insertID)
			if err == nil {
				oldestTime, err := time.Parse("2006-01-02T15:04:05", oldest)
				if err == nil {
					tcc := internaldto.NewTxnControlCountersFromVals(genID, sessionID, txnID, insertID)
					tcc.SetTableName(tableName)
					return oldestTime, tcc
				}
			}
		}
	}
	return time.Time{}, nil
}

func (eng *sqLiteDialect) GetGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	return eng.getGCHousekeepingQuery(tableName, tcc)
}

func (eng *sqLiteDialect) getGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	templateQuery := `INSERT OR IGNORE INTO 
	  "__iql__.control.gc.txn_table_x_ref" (
			iql_generation_id, 
			iql_session_id, 
			iql_transaction_id, 
			table_name
		) values(%d, %d, %d, '%s')`
	return fmt.Sprintf(templateQuery, tcc.GetGenID(), tcc.GetSessionID(), tcc.GetTxnID(), tableName)
}

func (eng *sqLiteDialect) ComposeSelectQuery(columns []relationaldto.RelationalColumn, tableAliases []string, fromString string, rewrittenWhere string, selectSuffix string) (string, error) {
	return eng.composeSelectQuery(columns, tableAliases, fromString, rewrittenWhere, selectSuffix)
}

func (eng *sqLiteDialect) composeSelectQuery(columns []relationaldto.RelationalColumn, tableAliases []string, fromString string, rewrittenWhere string, selectSuffix string) (string, error) {
	var q strings.Builder
	var quotedColNames []string
	for _, col := range columns {
		quotedColNames = append(quotedColNames, col.CanonicalSelectionString())
	}
	genIdColName := eng.controlAttributes.GetControlGenIdColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := eng.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := eng.controlAttributes.GetControlInsIdColumnName()
	var wq strings.Builder
	var controlWhereComparisons []string
	i := 0
	for _, alias := range tableAliases {
		if alias != "" {
			gIDcn := fmt.Sprintf(`"%s"."%s"`, alias, genIdColName)
			sIDcn := fmt.Sprintf(`"%s"."%s"`, alias, sessionIDColName)
			tIDcn := fmt.Sprintf(`"%s"."%s"`, alias, txnIdColName)
			iIDcn := fmt.Sprintf(`"%s"."%s"`, alias, insIdColName)
			controlWhereComparisons = append(controlWhereComparisons, fmt.Sprintf(`%s = ? AND %s = ? AND %s = ? AND %s = ?`, gIDcn, sIDcn, tIDcn, iIDcn))
		} else {
			gIDcn := fmt.Sprintf(`"%s"`, genIdColName)
			sIDcn := fmt.Sprintf(`"%s"`, sessionIDColName)
			tIDcn := fmt.Sprintf(`"%s"`, txnIdColName)
			iIDcn := fmt.Sprintf(`"%s"`, insIdColName)
			controlWhereComparisons = append(controlWhereComparisons, fmt.Sprintf(`%s = ? AND %s = ? AND %s = ? AND %s = ?`, gIDcn, sIDcn, tIDcn, iIDcn))
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

func (eng *sqLiteDialect) GetFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return eng.getFullyQualifiedTableName(unqualifiedTableName)
}

func (eng *sqLiteDialect) getFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return fmt.Sprintf(`"%s"`, unqualifiedTableName), nil
}

func (eng *sqLiteDialect) SanitizeQueryString(queryString string) (string, error) {
	return eng.sanitizeQueryString(queryString)
}

func (eng *sqLiteDialect) sanitizeQueryString(queryString string) (string, error) {
	return queryString, nil
}

func (eng *sqLiteDialect) SanitizeWhereQueryString(queryString string) (string, error) {
	return eng.sanitizeWhereQueryString(queryString)
}

func (eng *sqLiteDialect) sanitizeWhereQueryString(queryString string) (string, error) {
	return queryString, nil
}

func (eng *sqLiteDialect) GenerateInsertDML(relationalTable relationaldto.RelationalTable, tcc internaldto.TxnControlCounters) (string, error) {
	return eng.generateInsertDML(relationalTable, tcc)
}

func (eng *sqLiteDialect) generateInsertDML(relationalTable relationaldto.RelationalTable, tcc internaldto.TxnControlCounters) (string, error) {
	var q strings.Builder
	var quotedColNames, vals []string
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s" `, tableName))
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
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	vals = append(vals, "?")
	for _, col := range relationalTable.GetColumns() {
		quotedColNames = append(quotedColNames, `"`+col.GetName()+`" `)
		vals = append(vals, "?")
	}
	q.WriteString(fmt.Sprintf(" (%s) ", strings.Join(quotedColNames, ", ")))
	q.WriteString(fmt.Sprintf(" VALUES (%s) ", strings.Join(vals, ", ")))
	return q.String(), nil
}

func (eng *sqLiteDialect) GenerateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
	return eng.generateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)
}

func (eng *sqLiteDialect) generateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
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
	q.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" %s WHERE `, strings.Join(quotedColNames, ", "), tableName, aliasStr))
	q.WriteString(fmt.Sprintf(`( "%s" = ? AND "%s" = ? AND "%s" = ? AND "%s" = ? ) `, genIdColName, sessionIDColName, txnIdColName, insIdColName))
	if strings.TrimSpace(rewrittenWhere) != "" {
		q.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
	}
	q.WriteString(selectSuffix)

	return q.String(), nil
}

func (sl *sqLiteDialect) gcControlTablesPurge() error {
	obtainQuery := `
		SELECT
		  'DELETE FROM "' || name || '" ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
			and
			name like '__iql__%'
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *sqLiteDialect) GCPurgeEphemeral() error {
	return sl.gcPurgeEphemeral()
}

func (sl *sqLiteDialect) GCPurgeCache() error {
	return sl.gcPurgeCache()
}

func (sl *sqLiteDialect) gcPurgeCache() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || name || '" ; ' 
	from sqlite_schema 
	where type = 'table' and name like ?
	`
	rows, err := sl.sqlEngine.Query(query, sl.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(rows)
}

func (sl *sqLiteDialect) gcPurgeEphemeral() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || name || '" ; ' 
	from 
		sqlite_schema 
	where 
		type = 'table' 
		and 
		name NOT like ? 
		and 
		name not like '__iql__%' 
		and
		name NOT LIKE 'sqlite_%' 
	`
	rows, err := sl.sqlEngine.Query(query, sl.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(rows)
}

func (sl *sqLiteDialect) PurgeAll() error {
	return sl.purgeAll()
}

func (sl *sqLiteDialect) GetOperatorOr() string {
	return "||"
}

func (sl *sqLiteDialect) GetOperatorStringConcat() string {
	return "|"
}

func (sl *sqLiteDialect) purgeAll() error {
	obtainQuery := `
		SELECT
			'DROP TABLE IF EXISTS "' || name || '" ; '
		FROM
			sqlite_master 
		where 
			type = 'table'
		  AND
			name NOT LIKE '__iql__%'
			and
			name NOT LIKE 'sqlite_%'
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *sqLiteDialect) DelimitGroupByColumn(term string) string {
	return term
}

func (eng *sqLiteDialect) DelimitOrderByColumn(term string) string {
	return term
}

func (sl *sqLiteDialect) readExecGeneratedQueries(queryResultSet *sql.Rows) error {
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

func (eng *sqLiteDialect) GetRelationalType(discoType string) string {
	return eng.getRelationalType(discoType)
}

func (eng *sqLiteDialect) getRelationalType(discoType string) string {
	rv, ok := eng.typeMappings[discoType]
	if ok {
		return rv.GetRelationalType()
	}
	return eng.defaultRelationalType
}

func (eng *sqLiteDialect) GetGolangValue(discoType string) interface{} {
	return eng.getGolangValue(discoType)
}

func (eng *sqLiteDialect) getGolangValue(discoType string) interface{} {
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
	case reflect.Int:
		return &sql.NullInt64{}
	}
	return eng.getDefaultGolangValue()
}

func (eng *sqLiteDialect) getDefaultGolangValue() interface{} {
	return &sql.NullString{}
}

func (eng *sqLiteDialect) GetGolangKind(discoType string) reflect.Kind {
	rv, ok := eng.typeMappings[discoType]
	if !ok {
		return eng.getDefaultGolangKind()
	}
	return rv.GetGolangKind()
}

func (eng *sqLiteDialect) getDefaultGolangKind() reflect.Kind {
	return eng.defaultGolangKind
}

func (eng *sqLiteDialect) QueryNamespaced(colzString, actualTableName, requestEncodingColName, requestEncoding string) (*sql.Rows, error) {
	return eng.sqlEngine.Query(fmt.Sprintf(`SELECT %s FROM "%s" WHERE "%s" = ?`, colzString, actualTableName, requestEncodingColName), requestEncoding)
}
