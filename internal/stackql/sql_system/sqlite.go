package sql_system

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/lib/pq/oid"
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/relationaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

func newSQLiteSystem(sqlEngine sqlengine.SQLEngine, analyticsNamespaceLikeString string, controlAttributes sqlcontrol.ControlAttributes, formatter sqlparser.NodeFormatter, sqlCfg dto.SQLBackendCfg, authCfg map[string]*dto.AuthCtx) (SQLSystem, error) {
	rv := &sqLiteSystem{
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
		authCfg:                      authCfg,
	}
	err := rv.initSQLiteEngine()
	return rv, err
}

type sqLiteSystem struct {
	controlAttributes            sqlcontrol.ControlAttributes
	analyticsNamespaceLikeString string
	sqlEngine                    sqlengine.SQLEngine
	formatter                    sqlparser.NodeFormatter
	typeMappings                 map[string]internaldto.DRMCoupling
	defaultRelationalType        string
	defaultGolangKind            reflect.Kind
	authCfg                      map[string]*dto.AuthCtx
}

func (eng *sqLiteSystem) initSQLiteEngine() error {
	_, err := eng.sqlEngine.Exec(sqLiteEngineSetupDDL)
	return err
}

func (se *sqLiteSystem) GetTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers, discoveryId int) (internaldto.DBTable, error) {
	return se.getTable(tableHeirarchyIDs, discoveryId)
}

func (se *sqLiteSystem) getTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers, discoveryId int) (internaldto.DBTable, error) {
	tableNameStump, err := se.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableName := fmt.Sprintf("%s.generation_%d", tableNameStump, discoveryId)
	return internaldto.NewDBTable(tableName, tableNameStump, tableHeirarchyIDs.GetTableName(), discoveryId, tableHeirarchyIDs), err
}

func (se *sqLiteSystem) GetCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	return se.getCurrentTable(tableHeirarchyIDs)
}

func (se *sqLiteSystem) getTableNameStump(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (string, error) {
	return tableHeirarchyIDs.GetTableName(), nil
}

func (se *sqLiteSystem) getCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	var tableName string
	var discoID int
	tableNameStump, err := se.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	if _, isView := tableHeirarchyIDs.GetView(); isView {
		return internaldto.NewDBTable(tableNameStump, tableNameStump, tableHeirarchyIDs.GetTableName(), discoID, tableHeirarchyIDs), nil
	}
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableNameStump)
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableNameStump)
	res := se.sqlEngine.QueryRow(`select name, CAST(REPLACE(name, ?, '') AS INTEGER) from sqlite_schema where type = 'table' and name like ? ORDER BY name DESC limit 1`, tableNameLHSRemove, tableNamePattern)
	err = res.Scan(&tableName, &discoID)
	if err != nil {
		logging.GetLogger().Errorln(fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'", err, tableNamePattern, tableNameLHSRemove))
	}
	return internaldto.NewDBTable(tableName, tableNameStump, tableHeirarchyIDs.GetTableName(), discoID, tableHeirarchyIDs), nil
}

func (sl *sqLiteSystem) GetName() string {
	return constants.SQLDialectSQLite3
}

func (sl *sqLiteSystem) GetASTFormatter() sqlparser.NodeFormatter {
	return sl.formatter
}

func (sl *sqLiteSystem) GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter {
	return astfuncrewrite.GetNopFuncRewriter()
}

func (sl *sqLiteSystem) GCAdd(tableName string, parentTcc, lockableTcc internaldto.TxnControlCounters) error {
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

func (sl *sqLiteSystem) GCCollectObsoleted(minTransactionID int) error {
	return sl.gCCollectObsoleted(minTransactionID)
}

func (sl *sqLiteSystem) RegisterExternalTable(connectionName string, tableDetails openapistackql.SQLExternalTable) error {
	return sl.registerExternalTable(connectionName, tableDetails)
}

func (sl *sqLiteSystem) registerExternalTable(connectionName string, tableDetails openapistackql.SQLExternalTable) error {
	q := `
	INSERT OR IGNORE INTO "__iql__.external.columns" (
		connection_name 
	   ,catalog_name 
	   ,schema_name 
	   ,table_name 
	   ,column_name 
	   ,column_type
	   ,ordinal_position 
	   ,"oid" 
	   ,column_width 
	   ,column_precision 
	 ) VALUES (
	    ? 
	   ,? 
	   ,? 
	   ,?
	   ,? 
	   ,? 
	   ,? 
	   ,? 
	   ,? 
	   ,?
	 )
	`
	tx, err := sl.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	for ord, col := range tableDetails.GetColumns() {
		_, err := tx.Exec(
			q,
			connectionName,
			tableDetails.GetCatalogName(),
			tableDetails.GetSchemaName(),
			tableDetails.GetName(),
			col.GetName(),
			col.GetType(),
			ord,
			col.GetOid(),
			col.GetWidth(),
			col.GetPrecision(),
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	return err
}

func (sl *sqLiteSystem) ObtainRelationalColumnsFromExternalSQLtable(hierarchyIDs internaldto.HeirarchyIdentifiers) ([]relationaldto.RelationalColumn, error) {
	return sl.obtainRelationalColumnsFromExternalSQLtable(hierarchyIDs)
}

func (sl *sqLiteSystem) ObtainRelationalColumnFromExternalSQLtable(hierarchyIDs internaldto.HeirarchyIdentifiers, colName string) (relationaldto.RelationalColumn, error) {
	return sl.obtainRelationalColumnFromExternalSQLtable(hierarchyIDs, colName)
}

func (sl *sqLiteSystem) getSQLExternalSchema(providerName string) string {
	rv := ""
	if sl.authCfg != nil {
		ac, ok := sl.authCfg[providerName]
		if ok && ac != nil {
			sqlCfg, ok := ac.GetSQLCfg()
			if ok {
				rv = sqlCfg.GetSchemaType()
			}
		}
	}
	if rv == "" {
		rv = constants.SQLDataSourceSchemaDefault
	}
	return rv
}

func (sl *sqLiteSystem) obtainRelationalColumnsFromExternalSQLtable(hierarchyIDs internaldto.HeirarchyIdentifiers) ([]relationaldto.RelationalColumn, error) {
	q := `
	SELECT
		column_name 
	   ,column_type
	   ,"oid" 
	   ,column_width 
	   ,column_precision 
	FROM
	  "__iql__.external.columns"
	WHERE
	  connection_name = ?
	  AND
	  catalog_name = ?
	  AND
	  schema_name = ?
	  AND 
	  table_name = ?
	ORDER BY ordinal_position ASC
	`
	providerName := hierarchyIDs.GetProviderStr()
	connectionName := sl.getSQLExternalSchema(providerName)
	catalogName := ""
	schemaName := hierarchyIDs.GetServiceStr()
	tableName := hierarchyIDs.GetResourceStr()
	rows, err := sl.sqlEngine.Query(
		q,
		connectionName,
		catalogName,
		schemaName,
		tableName,
	)
	if err != nil {
		return nil, err
	}
	hasRow := false
	var rv []relationaldto.RelationalColumn
	for {
		if !rows.Next() {
			break
		}
		hasRow = true
		var columnName, columnType string
		var oID, colWidth, colPrecision int
		err := rows.Scan(&columnName, &columnType, &oID, &colWidth, &colPrecision)
		if err != nil {
			return nil, err
		}
		relationalColumn := relationaldto.NewRelationalColumn(columnName, columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
		rv = append(rv, relationalColumn)

	}
	if !hasRow {
		return nil, fmt.Errorf("cannot generate relational table from external table = '%s': not present in external metadata", tableName)
	}
	return rv, nil
}

func (sl *sqLiteSystem) obtainRelationalColumnFromExternalSQLtable(hierarchyIDs internaldto.HeirarchyIdentifiers, colName string) (relationaldto.RelationalColumn, error) {
	q := `
	SELECT
		column_name 
	   ,column_type
	   ,"oid" 
	   ,column_width 
	   ,column_precision 
	FROM
	  "__iql__.external.columns"
	WHERE
	  connection_name = ?
	  AND
	  catalog_name = ?
	  AND
	  schema_name = ?
	  AND 
	  table_name = ?
	  AND
	  column_name = ?
	ORDER BY ordinal_position ASC
	`
	providerName := hierarchyIDs.GetProviderStr()
	connectionName := sl.getSQLExternalSchema(providerName)
	catalogName := ""
	schemaName := hierarchyIDs.GetServiceStr()
	tableName := hierarchyIDs.GetResourceStr()
	row := sl.sqlEngine.QueryRow(
		q,
		connectionName,
		catalogName,
		schemaName,
		tableName,
		colName,
	)
	var columnName, columnType string
	var oID, colWidth, colPrecision int
	err := row.Scan(&columnName, &columnType, &oID, &colWidth, &colPrecision)
	if err != nil {
		return nil, err
	}
	relationalColumn := relationaldto.NewRelationalColumn(columnName, columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
	return relationalColumn, nil
}

func (sl *sqLiteSystem) gCCollectObsoleted(minTransactionID int) error {
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

func (sl *sqLiteSystem) GCCollectAll() error {
	return sl.gCCollectAll()
}

func (sl *sqLiteSystem) GetSQLEngine() sqlengine.SQLEngine {
	return sl.sqlEngine
}

func (sl *sqLiteSystem) gCCollectAll() error {
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

func (eng *sqLiteSystem) generateDropTableStatement(relationalTable relationaldto.RelationalTable) (string, error) {
	s, err := relationalTable.GetName()
	return fmt.Sprintf(`drop table if exists "%s"`, s), err
}

func (sl *sqLiteSystem) GCControlTablesPurge() error {
	return sl.gcControlTablesPurge()
}

func (eng *sqLiteSystem) GenerateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
	return eng.generateDDL(relationalTable, dropTable)
}

func (eng *sqLiteSystem) generateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
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
	rawViewDDL, err := eng.generateViewDDL(relationalTable)
	if err != nil {
		return nil, err
	}
	retVal = append(retVal, rawViewDDL...)
	return retVal, nil
}

func (eng *sqLiteSystem) GetViewByName(viewName string) (internaldto.ViewDTO, bool) {
	return eng.getViewByName(viewName)
}

func (eng *sqLiteSystem) getViewByName(viewName string) (internaldto.ViewDTO, bool) {
	q := `SELECT view_ddl FROM "__iql__.views" WHERE view_name = ? and deleted_dttm IS NULL`
	row := eng.sqlEngine.QueryRow(q, viewName)
	if row != nil {
		var viewDDL string
		err := row.Scan(&viewDDL)
		if err != nil {
			return nil, false
		}
		return internaldto.NewViewDTO(viewName, viewDDL), true
	}
	return nil, false
}

func (eng *sqLiteSystem) DropView(viewName string) error {
	_, err := eng.sqlEngine.Exec(`delete from "__iql__.views" where view_name = ?`, viewName)
	return err
}

func (eng *sqLiteSystem) CreateView(viewName string, rawDDL string) error {
	return eng.createView(viewName, rawDDL)
}

func (eng *sqLiteSystem) createView(viewName string, rawDDL string) error {
	q := `
	INSERT INTO "__iql__.views" (
		view_name,
		view_ddl
	  ) 
	  VALUES (
		?,
		?
	  )
	`
	_, err := eng.sqlEngine.Exec(q, viewName, rawDDL)
	return err
}

func (eng *sqLiteSystem) generateViewDDL(relationalTable relationaldto.RelationalTable) ([]string, error) {
	var colNames, retVal []string
	var createViewBuilder strings.Builder
	retVal = append(retVal, fmt.Sprintf(`drop view if exists "%s" ; `, relationalTable.GetBaseName()))
	createViewBuilder.WriteString(fmt.Sprintf(`create view "%s" AS `, relationalTable.GetBaseName()))
	for _, col := range relationalTable.GetColumns() {
		var b strings.Builder
		colName := col.DelimitedSelectionString(`"`)
		b.WriteString(colName)
		colNames = append(colNames, b.String())
	}
	tableName, err := relationalTable.GetName()
	if err != nil {
		return nil, err
	}
	createViewBuilder.WriteString(fmt.Sprintf(`select %s from "%s" ;`, strings.Join(colNames, ", "), tableName))
	retVal = append(retVal, createViewBuilder.String())
	return retVal, nil
}

func (eng *sqLiteSystem) IsTablePresent(tableName string, requestEncoding string, colName string) bool {
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
func (eng *sqLiteSystem) TableOldestUpdateUTC(tableName string, requestEncoding string, updateColName string, requestEncodingColName string) (time.Time, internaldto.TxnControlCounters) {
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

func (eng *sqLiteSystem) GetGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	return eng.getGCHousekeepingQuery(tableName, tcc)
}

func (eng *sqLiteSystem) getGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	templateQuery := `INSERT OR IGNORE INTO 
	  "__iql__.control.gc.txn_table_x_ref" (
			iql_generation_id, 
			iql_session_id, 
			iql_transaction_id, 
			table_name
		) values(%d, %d, %d, '%s')`
	return fmt.Sprintf(templateQuery, tcc.GetGenID(), tcc.GetSessionID(), tcc.GetTxnID(), tableName)
}

func (eng *sqLiteSystem) ComposeSelectQuery(columns []relationaldto.RelationalColumn, tableAliases []string, fromString string, rewrittenWhere string, selectSuffix string) (string, error) {
	return eng.composeSelectQuery(columns, tableAliases, fromString, rewrittenWhere, selectSuffix)
}

func (eng *sqLiteSystem) composeSelectQuery(columns []relationaldto.RelationalColumn, tableAliases []string, fromString string, rewrittenWhere string, selectSuffix string) (string, error) {
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

func (eng *sqLiteSystem) GetFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return eng.getFullyQualifiedTableName(unqualifiedTableName)
}

func (eng *sqLiteSystem) getFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return fmt.Sprintf(`"%s"`, unqualifiedTableName), nil
}

func (eng *sqLiteSystem) SanitizeQueryString(queryString string) (string, error) {
	return eng.sanitizeQueryString(queryString)
}

func (eng *sqLiteSystem) sanitizeQueryString(queryString string) (string, error) {
	return queryString, nil
}

func (eng *sqLiteSystem) SanitizeWhereQueryString(queryString string) (string, error) {
	return eng.sanitizeWhereQueryString(queryString)
}

func (eng *sqLiteSystem) sanitizeWhereQueryString(queryString string) (string, error) {
	return queryString, nil
}

func (eng *sqLiteSystem) GenerateInsertDML(relationalTable relationaldto.RelationalTable, tcc internaldto.TxnControlCounters) (string, error) {
	return eng.generateInsertDML(relationalTable, tcc)
}

func (eng *sqLiteSystem) generateInsertDML(relationalTable relationaldto.RelationalTable, tcc internaldto.TxnControlCounters) (string, error) {
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

func (eng *sqLiteSystem) GenerateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
	return eng.generateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)
}

func (eng *sqLiteSystem) generateSelectDML(relationalTable relationaldto.RelationalTable, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (string, error) {
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

func (sl *sqLiteSystem) gcControlTablesPurge() error {
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

func (sl *sqLiteSystem) GCPurgeEphemeral() error {
	return sl.gcPurgeEphemeral()
}

func (sl *sqLiteSystem) GCPurgeCache() error {
	return sl.gcPurgeCache()
}

func (sl *sqLiteSystem) gcPurgeCache() error {
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

func (sl *sqLiteSystem) gcPurgeEphemeral() error {
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

func (sl *sqLiteSystem) PurgeAll() error {
	return sl.purgeAll()
}

func (sl *sqLiteSystem) GetOperatorOr() string {
	return "||"
}

func (sl *sqLiteSystem) GetOperatorStringConcat() string {
	return "|"
}

func (sl *sqLiteSystem) purgeAll() error {
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

func (eng *sqLiteSystem) DelimitGroupByColumn(term string) string {
	return term
}

func (eng *sqLiteSystem) DelimitOrderByColumn(term string) string {
	return term
}

func (sl *sqLiteSystem) readExecGeneratedQueries(queryResultSet *sql.Rows) error {
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

func (eng *sqLiteSystem) GetRelationalType(discoType string) string {
	return eng.getRelationalType(discoType)
}

func (eng *sqLiteSystem) getRelationalType(discoType string) string {
	rv, ok := eng.typeMappings[discoType]
	if ok {
		return rv.GetRelationalType()
	}
	return eng.defaultRelationalType
}

func (eng *sqLiteSystem) GetGolangValue(discoType string) interface{} {
	return eng.getGolangValue(discoType)
}

func (eng *sqLiteSystem) getGolangValue(discoType string) interface{} {
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
	case reflect.Float64:
		return &sql.NullFloat64{}
	}
	return eng.getDefaultGolangValue()
}

func (eng *sqLiteSystem) getDefaultGolangValue() interface{} {
	return &sql.NullString{}
}

func (eng *sqLiteSystem) GetGolangKind(discoType string) reflect.Kind {
	rv, ok := eng.typeMappings[discoType]
	if !ok {
		return eng.getDefaultGolangKind()
	}
	return rv.GetGolangKind()
}

func (eng *sqLiteSystem) getDefaultGolangKind() reflect.Kind {
	return eng.defaultGolangKind
}

func (eng *sqLiteSystem) QueryNamespaced(colzString, actualTableName, requestEncodingColName, requestEncoding string) (*sql.Rows, error) {
	return eng.sqlEngine.Query(fmt.Sprintf(`SELECT %s FROM "%s" WHERE "%s" = ?`, colzString, actualTableName, requestEncodingColName), requestEncoding)
}
