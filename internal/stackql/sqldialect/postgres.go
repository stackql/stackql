package sqldialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/relationaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"vitess.io/vitess/go/vt/sqlparser"
)

func newPostgresDialect(sqlEngine sqlengine.SQLEngine, analyticsNamespaceLikeString string, controlAttributes sqlcontrol.ControlAttributes, formatter sqlparser.NodeFormatter) (SQLDialect, error) {
	rv := &postgresDialect{
		defaultGolangKind:     reflect.String,
		defaultRelationalType: "text",
		typeMappings: map[string]dto.DRMCoupling{
			"array":   {RelationalType: "text", GolangKind: reflect.Slice},
			"boolean": {RelationalType: "boolean", GolangKind: reflect.Bool},
			"int":     {RelationalType: "integer", GolangKind: reflect.Int},
			"integer": {RelationalType: "integer", GolangKind: reflect.Int},
			"object":  {RelationalType: "text", GolangKind: reflect.Map},
			"string":  {RelationalType: "text", GolangKind: reflect.String},
			"number":  {RelationalType: "numeric", GolangKind: reflect.Float64},
		},
		controlAttributes:            controlAttributes,
		analyticsNamespaceLikeString: analyticsNamespaceLikeString,
		sqlEngine:                    sqlEngine,
		formatter:                    formatter,
	}
	err := rv.initPostgresEngine()
	return rv, err
}

type postgresDialect struct {
	controlAttributes            sqlcontrol.ControlAttributes
	analyticsNamespaceLikeString string
	sqlEngine                    sqlengine.SQLEngine
	formatter                    sqlparser.NodeFormatter
	typeMappings                 map[string]dto.DRMCoupling
	defaultRelationalType        string
	defaultGolangKind            reflect.Kind
}

func (eng *postgresDialect) initPostgresEngine() error {
	_, err := eng.sqlEngine.Exec(postgresEngineSetupDDL)
	return err
}

func (eng *postgresDialect) generateDropTableStatement(relationalTable relationaldto.RelationalTable) string {
	return fmt.Sprintf(`drop table if exists "%s"`, relationalTable.GetName())
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

func (eng *postgresDialect) generateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
	var colDefs, retVal []string
	if dropTable {
		retVal = append(retVal, eng.generateDropTableStatement(relationalTable))
	}
	var rv strings.Builder
	tableName := relationalTable.GetName()
	rv.WriteString(fmt.Sprintf(`create table if not exists "%s" ( `, tableName))
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
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIdColName, tableName, genIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIdColName, tableName, sessionIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIdColName, tableName, txnIdColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIdColName, tableName, insIdColName))
	return retVal, nil
}

func (eng *postgresDialect) GetGCHousekeepingQuery(tableName string, tcc dto.TxnControlCounters) string {
	return eng.getGCHousekeepingQuery(tableName, tcc)
}

func (eng *postgresDialect) getGCHousekeepingQuery(tableName string, tcc dto.TxnControlCounters) string {
	templateQuery := `INSERT INTO 
	  "__iql__.control.gc.txn_table_x_ref" (
			iql_generation_id, 
			iql_session_id, 
			iql_transaction_id, 
			table_name
		) values(%d, %d, %d, '%s')
		ON CONFLICT (iql_generation_id, iql_session_id, iql_transaction_id, table_name) DO NOTHING
		`
	return fmt.Sprintf(templateQuery, tcc.GenId, tcc.SessionId, tcc.TxnId, tableName)
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

func (eng *postgresDialect) GenerateInsertDML(relationalTable relationaldto.RelationalTable, tcc *dto.TxnControlCounters) (string, error) {
	return eng.generateInsertDML(relationalTable, tcc)
}

func (eng *postgresDialect) generateInsertDML(relationalTable relationaldto.RelationalTable, tcc *dto.TxnControlCounters) (string, error) {
	var q strings.Builder
	var quotedColNames, vals []string
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s" `, relationalTable.GetName()))
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

func (eng *postgresDialect) GenerateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs *dto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
	return eng.generateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)
}

func (eng *postgresDialect) generateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs *dto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
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
	q.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" %s WHERE `, strings.Join(quotedColNames, ", "), relationalTable.GetName(), aliasStr))
	q.WriteString(fmt.Sprintf(`( "%s" = $1 AND "%s" = $2 AND "%s" = $3 AND "%s" = $4 ) `, genIdColName, sessionIDColName, txnIdColName, insIdColName))
	if strings.TrimSpace(rewrittenWhere) != "" {
		q.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
	}
	q.WriteString(selectSuffix)

	return q.String(), nil
}

func (sl *postgresDialect) GCAdd(tableName string, parentTcc, lockableTcc dto.TxnControlCounters) error {
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
	_, err := sl.sqlEngine.Exec(q, lockableTcc.TxnId, lockableTcc.InsertId)
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
			'DELETE FROM "' || table_name || '" WHERE "%s" < %d ; '
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
		maxTxnColName,
		minTransactionID,
	)
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.sqlEngine.GetTableCatalog(), sl.sqlEngine.GetTableSchema())
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
	obtainQuery := `
		SELECT
			'DELETE FROM "' || table_name || '"  ; '
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
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.sqlEngine.GetTableCatalog(), sl.sqlEngine.GetTableSchema())
	if err != nil {
		return err
	}
	return sl.readExecGeneratedQueries(deleteQueryResultSet)
}

func (sl *postgresDialect) GCControlTablesPurge() error {
	return sl.gcControlTablesPurge()
}

func (sl *postgresDialect) gcControlTablesPurge() error {
	obtainQuery := `
		SELECT
		  'DELETE FROM "' || table_name || '" ; '
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
		`
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.sqlEngine.GetTableCatalog(), sl.sqlEngine.GetTableSchema())
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
	rows, err := sl.sqlEngine.Query(query, sl.sqlEngine.GetTableCatalog(), sl.sqlEngine.GetTableSchema(), sl.analyticsNamespaceLikeString)
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
	rows, err := sl.sqlEngine.Query(query, sl.sqlEngine.GetTableCatalog(), sl.sqlEngine.GetTableSchema(), sl.analyticsNamespaceLikeString)
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
	deleteQueryResultSet, err := sl.sqlEngine.Query(obtainQuery, sl.sqlEngine.GetTableCatalog(), sl.sqlEngine.GetTableSchema())
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

func (eng *postgresDialect) zz(tableName string, tcc dto.TxnControlCounters) string {
	return eng.getGCHousekeepingQuery(tableName, tcc)
}

func (eng *postgresDialect) GetRelationalType(discoType string) string {
	return eng.getRelationalType(discoType)
}

func (eng *postgresDialect) getRelationalType(discoType string) string {
	rv, ok := eng.typeMappings[discoType]
	if ok {
		return rv.RelationalType
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
	return rv.GolangKind
}

func (eng *postgresDialect) getDefaultGolangKind() reflect.Kind {
	return eng.defaultGolangKind
}

func (eng *postgresDialect) QueryNamespaced(colzString string, actualTableName string, requestEncodingColName string, requestEncoding string) (*sql.Rows, error) {
	return eng.sqlEngine.Query(fmt.Sprintf(`SELECT %s FROM "%s" WHERE "%s" = $1`, colzString, actualTableName, requestEncodingColName), requestEncoding)
}
