//nolint:dupl,nolintlint //TODO: fix this
package sql_system //nolint:revive,stylecheck // package name is meaningful and readable

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq/oid"
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/relationaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/typing"
)

func newSQLiteSystem(
	sqlEngine sqlengine.SQLEngine,
	analyticsNamespaceLikeString string,
	controlAttributes sqlcontrol.ControlAttributes,
	formatter sqlparser.NodeFormatter,
	sqlCfg dto.SQLBackendCfg, //nolint:unparam,revive // future proof
	authCfg map[string]*dto.AuthCtx,
	typCfg typing.Config,
	exportNamepsace string,
) (SQLSystem, error) {
	rv := &sqLiteSystem{
		defaultGolangKind:            reflect.String,
		defaultRelationalType:        "text",
		typeCfg:                      typCfg,
		controlAttributes:            controlAttributes,
		analyticsNamespaceLikeString: analyticsNamespaceLikeString,
		sqlEngine:                    sqlEngine,
		formatter:                    formatter,
		authCfg:                      authCfg,
		exportNamespace:              exportNamepsace,
	}
	err := rv.initSQLiteEngine()
	return rv, err
}

type sqLiteSystem struct {
	controlAttributes            sqlcontrol.ControlAttributes
	analyticsNamespaceLikeString string
	sqlEngine                    sqlengine.SQLEngine
	formatter                    sqlparser.NodeFormatter
	typeCfg                      typing.Config
	defaultRelationalType        string
	defaultGolangKind            reflect.Kind
	authCfg                      map[string]*dto.AuthCtx
	exportNamespace              string
}

func (eng *sqLiteSystem) initSQLiteEngine() error {
	_, err := eng.sqlEngine.Exec(sqLiteEngineSetupDDL)
	return err
}

func (eng *sqLiteSystem) GetTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
	discoveryID int,
) (internaldto.DBTable, error) {
	return eng.getTable(tableHeirarchyIDs, discoveryID)
}

func (eng *sqLiteSystem) getTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
	discoveryID int,
) (internaldto.DBTable, error) {
	tableNameStump, err := eng.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableName := fmt.Sprintf("%s.generation_%d", tableNameStump, discoveryID)
	return internaldto.NewDBTable(
		tableName,
		tableNameStump,
		tableHeirarchyIDs.GetTableName(),
		discoveryID,
		tableHeirarchyIDs), err
}

func (eng *sqLiteSystem) GetCurrentTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
) (internaldto.DBTable, error) {
	return eng.getCurrentTable(tableHeirarchyIDs)
}

//nolint:unparam // future proof
func (eng *sqLiteSystem) getTableNameStump(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (string, error) {
	return tableHeirarchyIDs.GetTableName(), nil
}

func (eng *sqLiteSystem) getCurrentTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
) (internaldto.DBTable, error) {
	var tableName string
	var discoID int
	tableNameStump, err := eng.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	if _, isView := tableHeirarchyIDs.GetView(); isView {
		return internaldto.NewDBTable(
			tableNameStump,
			tableNameStump,
			tableHeirarchyIDs.GetTableName(),
			discoID,
			tableHeirarchyIDs,
		), nil
	}
	tableNamePattern := fmt.Sprintf("%s.generation_%%", tableNameStump)
	tableNameLHSRemove := fmt.Sprintf("%s.generation_", tableNameStump)
	res := eng.sqlEngine.QueryRow(
		`select name, CAST(REPLACE(name, ?, '') AS INTEGER) from sqlite_schema where type = 'table' and name like ? ORDER BY name DESC limit 1`, // nolint:lll // this is a long query, but it's a single line
		tableNameLHSRemove,
		tableNamePattern,
	)
	err = res.Scan(&tableName, &discoID)
	if err != nil {
		logging.GetLogger().Errorln(
			fmt.Sprintf("err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'",
				err,
				tableNamePattern,
				tableNameLHSRemove,
			),
		)
	}
	return internaldto.NewDBTable(
		tableName,
		tableNameStump,
		tableHeirarchyIDs.GetTableName(),
		discoID,
		tableHeirarchyIDs,
	), nil
}

func (eng *sqLiteSystem) GetName() string {
	return constants.SQLDialectSQLite3
}

func (eng *sqLiteSystem) GetASTFormatter() sqlparser.NodeFormatter {
	return eng.formatter
}

func (eng *sqLiteSystem) GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter {
	return astfuncrewrite.GetNopFuncRewriter()
}

//nolint:revive // future proof
func (eng *sqLiteSystem) GCAdd(
	tableName string, parentTcc,
	lockableTcc internaldto.TxnControlCounters,
) error {
	maxTxnColName := eng.controlAttributes.GetControlMaxTxnColumnName()
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
		eng.controlAttributes.GetControlTxnIDColumnName(),
		eng.controlAttributes.GetControlInsIDColumnName(),
		maxTxnColName,
		maxTxnColName,
		maxTxnColName,
		maxTxnColName,
	)
	_, err := eng.sqlEngine.Exec(q, lockableTcc.GetTxnID(), lockableTcc.GetInsertID())
	return err
}

func (eng *sqLiteSystem) GCCollectObsoleted(minTransactionID int) error {
	return eng.gCCollectObsoleted(minTransactionID)
}

func (eng *sqLiteSystem) RegisterExternalTable(
	connectionName string,
	tableDetails anysdk.SQLExternalTable,
) error {
	return eng.registerExternalTable(connectionName, tableDetails)
}

func (eng *sqLiteSystem) registerExternalTable(
	connectionName string,
	tableDetails anysdk.SQLExternalTable,
) error {
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
	tx, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	for ord, col := range tableDetails.GetColumns() {
		_, err = tx.Exec(
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
			//nolint:errcheck // TODO: merge variadic error(s) into one
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	return err
}

func (eng *sqLiteSystem) ObtainRelationalColumnsFromExternalSQLtable(
	hierarchyIDs internaldto.HeirarchyIdentifiers,
) ([]typing.RelationalColumn, error) {
	return eng.obtainRelationalColumnsFromExternalSQLtable(hierarchyIDs)
}

func (eng *sqLiteSystem) ObtainRelationalColumnFromExternalSQLtable(
	hierarchyIDs internaldto.HeirarchyIdentifiers,
	colName string,
) (typing.RelationalColumn, error) {
	return eng.obtainRelationalColumnFromExternalSQLtable(hierarchyIDs, colName)
}

func (eng *sqLiteSystem) getSQLExternalSchema(providerName string) string {
	rv := ""
	if eng.authCfg != nil {
		ac, ok := eng.authCfg[providerName]
		if ok && ac != nil {
			sqlCfg, sqlOk := ac.GetSQLCfg()
			if sqlOk {
				rv = sqlCfg.GetSchemaType()
			}
		}
	}
	if rv == "" {
		rv = constants.SQLDataSourceSchemaDefault
	}
	return rv
}

func (eng *sqLiteSystem) obtainRelationalColumnsFromExternalSQLtable(
	hierarchyIDs internaldto.HeirarchyIdentifiers,
) ([]typing.RelationalColumn, error) {
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
	connectionName := eng.getSQLExternalSchema(providerName)
	catalogName := ""
	schemaName := hierarchyIDs.GetServiceStr()
	tableName := hierarchyIDs.GetResourceStr()
	rows, err := eng.sqlEngine.Query( //nolint:rowserrcheck // TODO: fix this
		q,
		connectionName,
		catalogName,
		schemaName,
		tableName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hasRow := false
	var rv []typing.RelationalColumn
	for {
		if !rows.Next() {
			break
		}
		hasRow = true
		var columnName, columnType string
		var oID, colWidth, colPrecision int
		err = rows.Scan(&columnName, &columnType, &oID, &colWidth, &colPrecision)
		if err != nil {
			return nil, err
		}
		relationalColumn := typing.NewRelationalColumn(
			columnName,
			columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
		rv = append(rv, relationalColumn)
	}
	if !hasRow {
		return nil, fmt.Errorf(
			"cannot generate relational table from external table = '%s': not present in external metadata",
			tableName,
		)
	}
	return rv, nil
}

func (eng *sqLiteSystem) obtainRelationalColumnFromExternalSQLtable(
	hierarchyIDs internaldto.HeirarchyIdentifiers,
	colName string,
) (typing.RelationalColumn, error) {
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
	connectionName := eng.getSQLExternalSchema(providerName)
	catalogName := ""
	schemaName := hierarchyIDs.GetServiceStr()
	tableName := hierarchyIDs.GetResourceStr()
	row := eng.sqlEngine.QueryRow(
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
	relationalColumn := typing.NewRelationalColumn(columnName, columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
	return relationalColumn, nil
}

func (eng *sqLiteSystem) gCCollectObsoleted(minTransactionID int) error {
	maxTxnColName := eng.controlAttributes.GetControlMaxTxnColumnName()
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
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *sqLiteSystem) GCCollectAll() error {
	return eng.gCCollectAll()
}

func (eng *sqLiteSystem) GetSQLEngine() sqlengine.SQLEngine {
	return eng.sqlEngine
}

func (eng *sqLiteSystem) gCCollectAll() error {
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
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *sqLiteSystem) generateDropTableStatement(relationalTable relationaldto.RelationalTable) (string, error) {
	s, err := relationalTable.GetName()
	return fmt.Sprintf(`drop table if exists "%s"`, s), err
}

func (eng *sqLiteSystem) GCControlTablesPurge() error {
	return eng.gcControlTablesPurge()
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
	genIDColName := eng.controlAttributes.GetControlGenIDColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIDColumnName()
	txnIDColName := eng.controlAttributes.GetControlTxnIDColumnName()
	maxTxnIDColName := eng.controlAttributes.GetControlMaxTxnColumnName()
	insIDColName := eng.controlAttributes.GetControlInsIDColumnName()
	lastUpdateColName := eng.controlAttributes.GetControlLatestUpdateColumnName()
	insertEncodedColName := eng.controlAttributes.GetControlInsertEncodedIDColumnName()
	gcStatusColName := eng.controlAttributes.GetControlGCStatusColumnName()
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, genIDColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, sessionIDColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, txnIDColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, maxTxnIDColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" INTEGER `, insIDColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" TEXT `, insertEncodedColName))
	colDefs = append(colDefs, fmt.Sprintf(`"%s" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP `, lastUpdateColName)) //nolint:lll // this is a long line but it is a string
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
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIDColName, tableName, genIDColName))         //nolint:lll // this is a long line but it is more readable this way
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIDColName, tableName, sessionIDColName)) //nolint:lll // this is a long line but it is more readable this way
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIDColName, tableName, txnIDColName))         //nolint:lll // this is a long line but it is more readable this way
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIDColName, tableName, insIDColName))         //nolint:lll // this is a long line but it is more readable this way
	rawViewDDL, err := eng.generateViewDDL(relationalTable)
	if err != nil {
		return nil, err
	}
	retVal = append(retVal, rawViewDDL...)
	return retVal, nil
}

func (eng *sqLiteSystem) GetViewByName(viewName string) (internaldto.RelationDTO, bool) {
	return eng.getViewByName(viewName)
}

func (eng *sqLiteSystem) getViewByName(viewName string) (internaldto.RelationDTO, bool) {
	q := `SELECT view_ddl FROM "__iql__.views" WHERE view_name = ? and deleted_dttm IS NULL`
	row := eng.sqlEngine.QueryRow(q, viewName)
	if row == nil {
		return nil, false
	}
	var viewDDL string
	err := row.Scan(&viewDDL)
	if err != nil {
		return nil, false
	}
	rv := internaldto.NewViewDTO(viewName, viewDDL)
	return rv, true
}

func (eng *sqLiteSystem) DropView(viewName string) error {
	_, err := eng.sqlEngine.Exec(`delete from "__iql__.views" where view_name = ?`, viewName)
	return err
}

func (eng *sqLiteSystem) CreateView(viewName string, rawDDL string, replaceAllowed bool) error {
	return eng.createView(viewName, rawDDL, replaceAllowed)
}

func (eng *sqLiteSystem) createView(viewName string, rawDDL string, replaceAllowed bool) error {
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
	if replaceAllowed {
		q += `
		  ON CONFLICT(view_name)
		  DO
		    UPDATE SET view_ddl = EXCLUDED.view_ddl
		`
	}
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

//nolint:unparam,revive // future proof
func (eng *sqLiteSystem) CreateMaterializedView(
	relationName string,
	colz []typing.RelationalColumn,
	rawDDL string,
	replaceAllowed bool,
	selectQuery string,
	varargs ...any,
) error {
	return eng.runMaterializedViewCreate(
		relationName,
		colz,
		rawDDL,
		selectQuery,
		varargs...,
	)
}

//nolint:errcheck,revive,staticcheck // TODO: establish pattern
func (eng *sqLiteSystem) RefreshMaterializedView(naiveViewName string,
	colz []typing.RelationalColumn,
	selectQuery string,
	varargs ...any) error {
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveViewName)
	//nolint:gosec // no viable alternative
	deleteQuery := fmt.Sprintf(`
		DELETE FROM "%s"`,
		fullyQualifiedRelationName,
	)
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	_, err = txn.Exec(deleteQuery)
	if err != nil {
		txn.Rollback()
		return err
	}
	// TODO: check colz against DTO
	relationDTO, relationDTOok := eng.getMaterializedViewByName(naiveViewName, txn)
	if !relationDTOok {
		if len(relationDTO.GetColumns()) == 0 {
		}
		// no need to rollbak; assumed already done
		return fmt.Errorf("cannot refresh materialized view = '%s': not found", naiveViewName)
	}
	insertQuery := eng.generateTableInsertDMLFromViewSelect(fullyQualifiedRelationName, selectQuery, colz)
	_, err = txn.Exec(insertQuery, varargs...)
	if err != nil {
		txn.Rollback()
		return err
	}
	commitErr := txn.Commit()
	return commitErr
}

//nolint:errcheck,revive,staticcheck // TODO: establish pattern
func (eng *sqLiteSystem) InsertIntoPhysicalTable(naiveTableName string,
	columnsString string,
	selectQuery string,
	varargs ...any) error {
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveTableName)
	// TODO: check colz against supplied columns
	relationDTO, relationDTOok := eng.getTableByName(naiveTableName, txn)
	if !relationDTOok {
		if len(relationDTO.GetColumns()) == 0 {
		}
		// no need to rollbak; assumed already done
		return fmt.Errorf("cannot refresh materialized view = '%s': not found", fullyQualifiedRelationName)
	}
	//nolint:gosec // no viable alternative
	insertQuery := fmt.Sprintf("INSERT INTO %s %s %s", fullyQualifiedRelationName, columnsString, selectQuery)
	_, err = txn.Exec(insertQuery, varargs...)
	if err != nil {
		txn.Rollback()
		return err
	}
	commitErr := txn.Commit()
	return commitErr
}

//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) DropMaterializedView(naiveViewName string) error {
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveViewName)
	dropRefQuery := `
	DELETE FROM "__iql__.materialized_views"
	WHERE view_name = ?
	`
	dropColsQuery := `
	DELETE
	FROM
	  "__iql__.materialized_views.columns"
	WHERE
	  view_name = ?
	`
	dropTableQuery := fmt.Sprintf(`
	DROP TABLE IF EXISTS "%s"
	`, fullyQualifiedRelationName)
	tx, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(dropRefQuery, fullyQualifiedRelationName)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(dropColsQuery, fullyQualifiedRelationName)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(dropTableQuery)
	if err != nil {
		tx.Rollback()
		return err
	}
	commitErr := tx.Commit()
	return commitErr
}

//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) GetMaterializedViewByName(viewName string) (internaldto.RelationDTO, bool) {
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return nil, false
	}
	rv, ok := eng.getMaterializedViewByName(viewName, txn)
	txn.Commit()
	return rv, ok
}

func (eng *sqLiteSystem) IsRelationExported(relationName string) bool {
	if eng.exportNamespace == "" {
		return false
	}
	matches, _ := regexp.MatchString(fmt.Sprintf(`^%s.*$`, eng.exportNamespace), relationName)
	return matches
}

//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) getMaterializedViewByName(naiveViewName string, txn *sql.Tx) (internaldto.RelationDTO, bool) {
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveViewName)
	q := `SELECT view_ddl FROM "__iql__.materialized_views" WHERE view_name = ? and deleted_dttm IS NULL`
	colQuery := `
	SELECT
		column_name 
	   ,column_type
	   ,"oid" 
	   ,column_width 
	   ,column_precision 
	FROM
	  "__iql__.materialized_views.columns"
	WHERE
	  view_name = ?
	ORDER BY ordinal_position ASC
	`
	// txn, txnErr := eng.sqlEngine.GetTx()
	// if txnErr != nil {
	// 	return nil, false
	// }
	row := txn.QueryRow(q, fullyQualifiedRelationName)
	if row == nil {
		txn.Rollback()
		return nil, false
	}
	var viewDDL string
	err := row.Scan(&viewDDL)
	if err != nil {
		txn.Rollback()
		return nil, false
	}
	rv := internaldto.NewMaterializedViewDTO(fullyQualifiedRelationName, viewDDL, eng.exportNamespace)
	rows, err := txn.Query(colQuery, fullyQualifiedRelationName)
	if err != nil || rows == nil || rows.Err() != nil {
		txn.Rollback()
		return nil, false
	}
	defer rows.Close()
	hasRow := false
	var columns []typing.RelationalColumn
	for {
		if !rows.Next() {
			break
		}
		hasRow = true
		var columnName, columnType string
		var oID, colWidth, colPrecision int
		err = rows.Scan(&columnName, &columnType, &oID, &colWidth, &colPrecision)
		if err != nil {
			txn.Rollback()
			return nil, false
		}
		relationalColumn := typing.NewRelationalColumn(
			columnName,
			columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
		columns = append(columns, relationalColumn)
	}
	rv.SetColumns(columns)
	if !hasRow {
		txn.Rollback()
		return nil, false
	}
	return rv, true
}

//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) GetPhysicalTableByName(
	tableName string) (internaldto.RelationDTO, bool) {
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return nil, false
	}
	rv, ok := eng.getTableByName(tableName, txn)
	txn.Commit()
	return rv, ok
}

// TODO: implement temp tables
//
//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) getTableByName(
	naiveTableName string,
	txn *sql.Tx,
) (internaldto.RelationDTO, bool) {
	fullyQualifiedTableName := eng.getFullyQualifiedRelationName(naiveTableName)
	q := `SELECT table_ddl FROM "__iql__.tables" WHERE table_name = ? and deleted_dttm IS NULL`
	colQuery := `
	SELECT
		column_name 
	   ,column_type
	   ,"oid" 
	   ,column_width 
	   ,column_precision 
	FROM
	  "__iql__.tables.columns"
	WHERE
	  table_name = ?
	ORDER BY ordinal_position ASC
	`
	row := txn.QueryRow(q, fullyQualifiedTableName)
	if row == nil {
		txn.Rollback()
		return nil, false
	}
	var viewDDL string
	err := row.Scan(&viewDDL)
	if err != nil {
		txn.Rollback()
		return nil, false
	}
	rv := internaldto.NewPhysicalTableDTO(fullyQualifiedTableName, viewDDL, eng.exportNamespace)
	rows, err := txn.Query(colQuery, fullyQualifiedTableName)
	if err != nil || rows == nil || rows.Err() != nil {
		txn.Rollback()
		return nil, false
	}
	defer rows.Close()
	hasRow := false
	var columns []typing.RelationalColumn
	for {
		if !rows.Next() {
			break
		}
		hasRow = true
		var columnName, columnType string
		var oID, colWidth, colPrecision int
		err = rows.Scan(&columnName, &columnType, &oID, &colWidth, &colPrecision)
		if err != nil {
			txn.Rollback()
			return nil, false
		}
		relationalColumn := typing.NewRelationalColumn(
			columnName,
			columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
		columns = append(columns, relationalColumn)
	}
	rv.SetColumns(columns)
	if !hasRow {
		txn.Rollback()
		return nil, false
	}
	return rv, true
}

// TODO: implement temp table drop
func (eng *sqLiteSystem) DropPhysicalTable(naiveTableName string,
	ifExists bool,
) error {
	fullyQualifiedTableName := eng.getFullyQualifiedRelationName(naiveTableName)
	dropRefQuery := `
	DELETE FROM "__iql__.tables"
	WHERE table_name = ?
	`
	dropTableQuery := fmt.Sprintf(`
	DROP TABLE "%s"
	`, fullyQualifiedTableName)
	if ifExists {
		dropTableQuery = fmt.Sprintf(`
		DROP TABLE IF EXISTS "%s"
		`, fullyQualifiedTableName)
	}
	dropColsQuery := `
	DELETE
	FROM
	  "__iql__.tables.columns"
	WHERE
	  table_name = ?
	`
	tx, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(dropRefQuery, fullyQualifiedTableName)
	if err != nil {
		//nolint:errcheck // TODO: merge variadic error(s) into one
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(dropTableQuery)
	if err != nil {
		//nolint:errcheck // TODO: merge variadic error(s) into one
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(dropColsQuery, fullyQualifiedTableName)
	if err != nil {
		//nolint:errcheck // TODO: merge variadic error(s) into one
		tx.Rollback()
		return err
	}
	commitErr := tx.Commit()
	return commitErr
}

func (eng *sqLiteSystem) GetFullyQualifiedRelationName(tableName string) string {
	return eng.getFullyQualifiedRelationName(tableName)
}

func (eng *sqLiteSystem) getFullyQualifiedRelationName(tableName string) string {
	if eng.exportNamespace == "" {
		return tableName
	}
	strippedTableName := strings.ReplaceAll(tableName, `"`, "")
	return fmt.Sprintf(`"%s.%s"`, eng.exportNamespace, strippedTableName)
}

// TODO: implement temp table creation
func (eng *sqLiteSystem) CreatePhysicalTable(
	relationName string,
	colz []typing.RelationalColumn,
	rawDDL string,
	ifNotExists bool,
) error {
	return eng.runPhysicalTableCreate(
		relationName,
		colz,
		rawDDL,
		ifNotExists,
	)
}

func (eng *sqLiteSystem) IsTablePresent(
	tableName string,
	requestEncoding string,
	colName string, //nolint:revive // future proof
) bool {
	rows, err := eng.sqlEngine.Query( //nolint:rowserrcheck // TODO: fix this
		fmt.Sprintf(`SELECT count(*) as ct FROM "%s" WHERE iql_insert_encoded=?;`, tableName),
		requestEncoding,
	)
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var ct int
			//nolint:errcheck // TODO: merge variadic error(s) into one
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
func (eng *sqLiteSystem) TableOldestUpdateUTC(
	tableName string,
	requestEncoding string,
	updateColName string,
	requestEncodingColName string,
) (time.Time, internaldto.TxnControlCounters) {
	genIDColName := eng.controlAttributes.GetControlGenIDColumnName()
	ssnIDColName := eng.controlAttributes.GetControlSsnIDColumnName()
	txnIDColName := eng.controlAttributes.GetControlTxnIDColumnName()
	insIDColName := eng.controlAttributes.GetControlInsIDColumnName()
	rows, err := eng.sqlEngine.Query( //nolint:rowserrcheck // TODO: fix this
		fmt.Sprintf(
			"SELECT strftime('%%Y-%%m-%%dT%%H:%%M:%%S', min(%s)) as oldest_update, %s, %s, %s, %s FROM \"%s\" WHERE %s = '%s';",
			updateColName,
			genIDColName,
			ssnIDColName,
			txnIDColName,
			insIDColName,
			tableName,
			requestEncodingColName,
			requestEncoding,
		),
	)
	//nolint:nestif // TODO: simplify nested if statements
	if err == nil && rows != nil {
		defer rows.Close()
		rowExists := rows.Next()
		if rowExists {
			var oldest string
			var genID, sessionID, txnID, insertID int
			err = rows.Scan(&oldest, &genID, &sessionID, &txnID, &insertID)
			if err == nil {
				var oldestTime time.Time
				oldestTime, err = time.Parse("2006-01-02T15:04:05", oldest)
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

func (eng *sqLiteSystem) render(alias string) string {
	genIDColName := eng.controlAttributes.GetControlGenIDColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIDColumnName()
	txnIDColName := eng.controlAttributes.GetControlTxnIDColumnName()
	insIDColName := eng.controlAttributes.GetControlInsIDColumnName()
	if alias != "" {
		gIDcn := fmt.Sprintf(`"%s"."%s"`, alias, genIDColName)
		sIDcn := fmt.Sprintf(`"%s"."%s"`, alias, sessionIDColName)
		tIDcn := fmt.Sprintf(`"%s"."%s"`, alias, txnIDColName)
		iIDcn := fmt.Sprintf(`"%s"."%s"`, alias, insIDColName)
		return fmt.Sprintf(`%s = ? AND %s = ? AND %s = ? AND %s = ?`, gIDcn, sIDcn, tIDcn, iIDcn)
	}
	gIDcn := fmt.Sprintf(`"%s"`, genIDColName)
	sIDcn := fmt.Sprintf(`"%s"`, sessionIDColName)
	tIDcn := fmt.Sprintf(`"%s"`, txnIDColName)
	iIDcn := fmt.Sprintf(`"%s"`, insIDColName)
	return fmt.Sprintf(`%s = ? AND %s = ? AND %s = ? AND %s = ?`, gIDcn, sIDcn, tIDcn, iIDcn)
}

//nolint:revive // Liskov substitution principle
func (eng *sqLiteSystem) ComposeSelectQuery(
	columns []typing.RelationalColumn,
	tableAliases []string,
	hoistedTableAliases []string,
	fromString string,
	rewrittenWhere string,
	selectSuffix string,
	parameterOffset int,
) (string, error) {
	return eng.composeSelectQuery(columns, tableAliases, hoistedTableAliases, fromString, rewrittenWhere, selectSuffix)
}

func (eng *sqLiteSystem) composeSelectQuery(
	columns []typing.RelationalColumn,
	tableAliases []string,
	hoistedTableAliases []string,
	fromString string,
	rewrittenWhere string,
	selectSuffix string,
) (string, error) {
	var q strings.Builder
	var quotedColNames []string
	for _, col := range columns {
		quotedColNames = append(quotedColNames, col.CanonicalSelectionString())
	}
	var wq strings.Builder
	var hoistedControlOnComparisons []any
	i := 0
	if len(hoistedTableAliases) > 0 {
		for _, alias := range hoistedTableAliases {
			hoistedControlOnComparisons = append(
				hoistedControlOnComparisons,
				eng.render(alias),
			)
			i++
		}
		// BLOCK protect LHS string formats for indirect replacement
		remainingStringFormats := strings.Count(fromString, `%s`)
		diffCount := remainingStringFormats - len(hoistedControlOnComparisons)
		if diffCount > 0 {
			fromString = fmt.Sprintf(strings.Replace(fromString, `%s`, `%%s`, diffCount), hoistedControlOnComparisons...)
		} else {
			fromString = fmt.Sprintf(fromString, hoistedControlOnComparisons...)
		}
		// END BLOCK
	}
	var controlWhereComparisons []string
	for _, alias := range tableAliases {
		controlWhereComparisons = append(
			controlWhereComparisons,
			eng.render(alias),
		)
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

	q.WriteString(fmt.Sprintf(`SELECT %s `, strings.Join(quotedColNames, ", ")))
	if fromString != "" {
		q.WriteString(fmt.Sprintf(`FROM %s `, fromString))
	}
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

func (eng *sqLiteSystem) GenerateInsertDML(
	relationalTable relationaldto.RelationalTable,
	tcc internaldto.TxnControlCounters,
) (string, error) {
	return eng.generateInsertDML(relationalTable, tcc)
}

func (eng *sqLiteSystem) generateInsertDML(
	relationalTable relationaldto.RelationalTable,
	tcc internaldto.TxnControlCounters, //nolint:unparam,revive // future proof
) (string, error) {
	var q strings.Builder
	var quotedColNames, vals []string
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s" `, tableName))
	genIDColName := eng.controlAttributes.GetControlGenIDColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIDColumnName()
	txnIDColName := eng.controlAttributes.GetControlTxnIDColumnName()
	insIDColName := eng.controlAttributes.GetControlInsIDColumnName()
	insEncodedColName := eng.controlAttributes.GetControlInsertEncodedIDColumnName()
	quotedColNames = append(quotedColNames, `"`+genIDColName+`" `)
	quotedColNames = append(quotedColNames, `"`+sessionIDColName+`" `)
	quotedColNames = append(quotedColNames, `"`+txnIDColName+`" `)
	quotedColNames = append(quotedColNames, `"`+insIDColName+`" `)
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

func (eng *sqLiteSystem) GenerateSelectDML(
	relationalTable relationaldto.RelationalTable,
	txnCtrlCtrs internaldto.TxnControlCounters,
	selectSuffix,
	rewrittenWhere string,
) (string, error) {
	return eng.generateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)
}

func (eng *sqLiteSystem) generateSelectDML(
	relationalTable relationaldto.RelationalTable,
	txnCtrlCtrs internaldto.TxnControlCounters, //nolint:unparam,revive // future proof
	selectSuffix, rewrittenWhere string,
) (string, error) {
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
	genIDColName := eng.controlAttributes.GetControlGenIDColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIDColumnName()
	txnIDColName := eng.controlAttributes.GetControlTxnIDColumnName()
	insIDColName := eng.controlAttributes.GetControlInsIDColumnName()
	aliasStr := ""
	if relationalTable.GetAlias() != "" {
		aliasStr = fmt.Sprintf(` AS "%s" `, relationalTable.GetAlias())
	}
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	q.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" %s WHERE `, strings.Join(quotedColNames, ", "), tableName, aliasStr))
	q.WriteString(
		fmt.Sprintf(
			`( "%s" = ? AND "%s" = ? AND "%s" = ? AND "%s" = ? ) `,
			genIDColName,
			sessionIDColName,
			txnIDColName,
			insIDColName,
		),
	)
	if strings.TrimSpace(rewrittenWhere) != "" {
		q.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
	}
	q.WriteString(selectSuffix)

	return q.String(), nil
}

func (eng *sqLiteSystem) gcControlTablesPurge() error {
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
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *sqLiteSystem) GCPurgeEphemeral() error {
	return eng.gcPurgeEphemeral()
}

func (eng *sqLiteSystem) GCPurgeCache() error {
	return eng.gcPurgeCache()
}

func (eng *sqLiteSystem) gcPurgeCache() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || name || '" ; ' 
	from sqlite_schema 
	where type = 'table' and name like ?
	`
	rows, err := eng.sqlEngine.Query(query, eng.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(rows)
}

func (eng *sqLiteSystem) gcPurgeEphemeral() error {
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
	rows, err := eng.sqlEngine.Query(query, eng.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(rows)
}

func (eng *sqLiteSystem) PurgeAll() error {
	return eng.purgeAll()
}

func (eng *sqLiteSystem) GetOperatorOr() string {
	return "||"
}

func (eng *sqLiteSystem) GetOperatorStringConcat() string {
	return "|"
}

func (eng *sqLiteSystem) purgeAll() error {
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
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *sqLiteSystem) DelimitGroupByColumn(term string) string {
	return term
}

func (eng *sqLiteSystem) DelimitOrderByColumn(term string) string {
	return term
}

func (eng *sqLiteSystem) readExecGeneratedQueries(queryResultSet *sql.Rows) error {
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
	err := eng.sqlEngine.ExecInTxn(queries)
	return err
}

func (eng *sqLiteSystem) GetRelationalType(discoType string) string {
	return eng.getRelationalType(discoType)
}

func (eng *sqLiteSystem) getRelationalType(discoType string) string {
	return eng.typeCfg.GetRelationalType(discoType)
}

func (eng *sqLiteSystem) GetGolangValue(discoType string) interface{} {
	return eng.getGolangValue(discoType)
}

func (eng *sqLiteSystem) getGolangValue(discoType string) interface{} {
	return eng.typeCfg.GetGolangValue(discoType)
}

func (eng *sqLiteSystem) GetGolangKind(discoType string) reflect.Kind {
	return eng.typeCfg.GetGolangKind(discoType)
}

func (eng *sqLiteSystem) QueryNamespaced(
	colzString,
	actualTableName,
	requestEncodingColName,
	requestEncoding string,
) (*sql.Rows, error) {
	return eng.sqlEngine.Query(
		fmt.Sprintf(
			`SELECT %s FROM "%s" WHERE "%s" = ?`,
			colzString,
			actualTableName,
			requestEncodingColName,
		),
		requestEncoding,
	)
}

func (eng *sqLiteSystem) QueryMaterializedView(
	colzString,
	actualRelationName,
	whereClause string,
) (*sql.Rows, error) {
	return eng.sqlEngine.Query(
		fmt.Sprintf(
			`SELECT %s FROM "%s" WHERE %s`,
			colzString,
			actualRelationName,
			whereClause,
		),
	)
}

func (eng *sqLiteSystem) generateTableDDL(
	relationName string,
	colz []typing.RelationalColumn,
) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(`CREATE TABLE %s ( `, relationName))
	var colzString []string
	for _, col := range colz {
		colzString = append(colzString, fmt.Sprintf(`"%s" %s`, col.GetName(), col.GetType()))
	}
	sb.WriteString(strings.Join(colzString, ", "))
	sb.WriteString(" ) ")
	return sb.String()
}

func (eng *sqLiteSystem) generateTableInsertDMLFromViewSelect(
	relationName string,
	selectQuery string,
	colz []typing.RelationalColumn,
) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(`INSERT INTO %s ( `, relationName))
	var colzString []string
	for _, col := range colz {
		colzString = append(colzString, fmt.Sprintf(`"%s"`, col.GetName()))
	}
	sb.WriteString(strings.Join(colzString, ", "))
	sb.WriteString(" ) ")
	sb.WriteString(selectQuery)
	return sb.String()
}

//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) runMaterializedViewCreate(
	relationName string,
	colz []typing.RelationalColumn,
	rawDDL string,
	selectQuery string,
	varargs ...any,
) error {
	txn, txnErr := eng.sqlEngine.GetTx()
	if txnErr != nil {
		return txnErr
	}
	columnQuery := `
	INSERT INTO "__iql__.materialized_views.columns" (
		view_name,
		column_name,
		column_type,
		ordinal_position,
		"oid",
		column_width,
		column_precision
	  ) 
	  VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?
	  )
	  ;
	`
	for i, col := range colz {
		oid, oidExists := col.GetOID()
		if !oidExists {
			oid = 25
		}
		_, err := txn.Exec(
			columnQuery,
			relationName,
			col.GetName(),
			col.GetType(),
			i+1,
			oid,
			col.GetWidth(),
			0, // TODO: implement precision record
		)
		if err != nil {
			txn.Rollback()
			return err
		}
	}
	tableDDL := eng.generateTableDDL(relationName, colz)
	_, err := txn.Exec(tableDDL)
	if err != nil {
		txn.Rollback()
		return err
	}
	insertQuery := eng.generateTableInsertDMLFromViewSelect(relationName, selectQuery, colz)
	_, err = txn.Exec(insertQuery, varargs...)
	if err != nil {
		txn.Rollback()
		return err
	}
	relationCatalogueQuery := `
	INSERT INTO "__iql__.materialized_views" (
		view_name,
		view_ddl,
		translated_ddl,
		translated_inline_dml
	  ) 
	  VALUES (
		?,
		?,
		?,
		''
	  )
	  ;
	  `
	_, err = txn.Exec(
		relationCatalogueQuery,
		relationName,
		rawDDL,
		tableDDL,
	)
	if err != nil {
		txn.Rollback()
		return err
	}
	commitErr := txn.Commit()
	return commitErr
}

//nolint:errcheck // TODO: establish pattern
func (eng *sqLiteSystem) runPhysicalTableCreate(
	relationName string,
	colz []typing.RelationalColumn,
	rawDDL string,
	ifNotExists bool, //nolint:unparam,revive // future proof
) error {
	txn, txnErr := eng.sqlEngine.GetTx()
	if txnErr != nil {
		return txnErr
	}
	columnQuery := `
	INSERT INTO "__iql__.tables.columns" (
		table_name,
		column_name,
		column_type,
		ordinal_position,
		"oid",
		column_width,
		column_precision
	  ) 
	  VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?
	  )
	  ;
	`
	for i, col := range colz {
		oid, oidExists := col.GetOID()
		if !oidExists {
			oid = 25
		}
		_, err := txn.Exec(
			columnQuery,
			relationName,
			col.GetName(),
			col.GetType(),
			i+1,
			oid,
			col.GetWidth(),
			0, // TODO: implement precision record
		)
		if err != nil {
			txn.Rollback()
			return err
		}
	}
	_, err := txn.Exec(rawDDL)
	if err != nil {
		txn.Rollback()
		return err
	}
	relationCatalogueQuery := `
	INSERT INTO "__iql__.tables" (
		table_name,
		table_ddl
	  ) 
	  VALUES (
		?,
		?
	  )
	  ;
	  `
	_, err = txn.Exec(
		relationCatalogueQuery,
		relationName,
		rawDDL,
	)
	if err != nil {
		txn.Rollback()
		return err
	}
	commitErr := txn.Commit()
	return commitErr
}
