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
	"github.com/stackql/stackql/internal/stackql/constants"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/typing"

	"github.com/stackql/stackql/pkg/serde"
)

func newSQLiteSystem(
	sqlEngine sqlengine.SQLEngine,
	analyticsNamespaceLikeString string,
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

func (eng *sqLiteSystem) RegisterExternalTable(
	connectionName string,
	tableDetails sqlengine.SQLExternalTable,
) error {
	return eng.registerExternalTable(connectionName, tableDetails)
}

func (eng *sqLiteSystem) registerExternalTable(
	connectionName string,
	tableDetails sqlengine.SQLExternalTable,
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

//nolint:gosec // who cares
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

//nolint:gosec // TODO: establish pattern
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

// Garbage collection and purge act on provider cache tables only. The physical
// tables behind user space tables and materialized views are user data and are
// excluded via the "__iql__.tables" and "__iql__.materialized_views" catalogues
// (issue #799).
func (eng *sqLiteSystem) GetSQLEngine() sqlengine.SQLEngine {
	return eng.sqlEngine
}

func (eng *sqLiteSystem) GetViewByName(viewName string) (internaldto.RelationDTO, bool) {
	rv, ok := eng.getViewByName(viewName)
	if !ok {
		return nil, false
	}
	candidates, err := eng.getAwareViewsByName(viewName)
	currentNode := rv
	if err == nil {
		for _, candidate := range candidates {
			if rv.GetName() != candidate.GetName() {
				currentNode = currentNode.WithNext(candidate)
			}
		}
	}
	return rv, ok
}

func (eng *sqLiteSystem) GetViewByNameAndParameters(
	viewName string, params map[string]any) (internaldto.RelationDTO, bool) {
	rv, err := eng.selectMatchingView(viewName, params)
	if err != nil {
		return nil, false
	}
	return rv, true
}

func (eng *sqLiteSystem) getViewByName(viewName string) (internaldto.RelationDTO, bool) {
	q := `SELECT view_ddl, required_params FROM "__iql__.views" WHERE view_name = ? and deleted_dttm IS NULL`
	row := eng.sqlEngine.QueryRow(q, viewName)
	if row == nil {
		return nil, false
	}
	var viewDDL, requiredParametersStr string
	err := row.Scan(&viewDDL, &requiredParametersStr)
	if err != nil {
		return nil, false
	}
	paramSerDe := serde.NewStringArrayMapSerDe()
	requiredParameters, serDeErr := paramSerDe.Deserialize(requiredParametersStr)
	if serDeErr != nil {
		return nil, false
	}
	rv := internaldto.NewViewDTO(viewName, viewDDL).WithRequiredParams(requiredParameters)
	return rv, true
}

func (eng *sqLiteSystem) selectMatchingView(viewName string, params map[string]any) (internaldto.RelationDTO, error) {
	candidates, err := eng.getAwareViewsByName(viewName)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if successfulCandidate, ok := candidate.MatchOnParams(params); ok {
			return successfulCandidate, nil
		}
	}
	return nil, fmt.Errorf("no matching view found for viewName = '%s'", viewName)
}

func (eng *sqLiteSystem) getAwareViewsByName(viewName string) ([]internaldto.RelationDTO, error) {
	q := `SELECT view_name, view_ddl, required_params 
	FROM "__iql__.views" WHERE view_name LIKE ? and deleted_dttm IS NULL`
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return nil, err
	}
	var rv []internaldto.RelationDTO
	defer txn.Commit() //nolint:errcheck // TODO: establish pattern
	rows, err := txn.Query(q, fmt.Sprintf(`%s%%`, viewName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	var hasRow bool
	for {
		if !rows.Next() {
			break
		}
		hasRow = true
		var viewNameAware, viewDDL, requiredParametersStr string
		err = rows.Scan(&viewNameAware, &viewDDL, &requiredParametersStr)
		if err != nil {
			return nil, err
		}
		paramSerDe := serde.NewStringArrayMapSerDe()
		requiredParameters, serDeErr := paramSerDe.Deserialize(requiredParametersStr)
		if serDeErr != nil {
			return nil, serDeErr
		}
		viewDTO := internaldto.NewViewDTO(viewNameAware, viewDDL).WithRequiredParams(requiredParameters)
		rv = append(rv, viewDTO)
	}
	if !hasRow {
		return nil, fmt.Errorf("no views found for viewName = '%s'", viewName)
	}
	return rv, nil
}

func (eng *sqLiteSystem) DropView(viewName string) error {
	_, err := eng.sqlEngine.Exec(`delete from "__iql__.views" where view_name = ?`, viewName)
	return err
}

func (eng *sqLiteSystem) CreateView(
	viewName string, rawDDL string, replaceAllowed bool, requiredParams []string) error {
	return eng.createView(viewName, rawDDL, replaceAllowed, requiredParams)
}

func (eng *sqLiteSystem) createView(
	viewName string, rawDDL string, replaceAllowed bool, requiredParams []string) error {
	paramSerDe := serde.NewStringArrayMapSerDe()
	requiredParamsString, serdeErr := paramSerDe.Serialize(requiredParams)
	if serdeErr != nil {
		return serdeErr
	}
	q := `
	INSERT INTO "__iql__.views" (
		view_name,
		view_ddl,
		required_params
	  ) 
	  VALUES (	
		?,
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
	_, err := eng.sqlEngine.Exec(q, viewName, rawDDL, requiredParamsString)
	return err
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
		replaceAllowed,
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
	insertQuery := fmt.Sprintf(`INSERT INTO "%s" %s %s`, fullyQualifiedRelationName, columnsString, selectQuery)
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

//nolint:errcheck,gosec // TODO: establish pattern
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
//nolint:errcheck,gosec // TODO: establish pattern
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
	return fmt.Sprintf(`%s.%s`, eng.exportNamespace, strippedTableName)
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
	tableName, requestEncoding, updateColName, requestEncodingColName string,
) (time.Time, bool) {
	rows, err := eng.sqlEngine.Query(
		fmt.Sprintf(`SELECT strftime('%%Y-%%m-%%dT%%H:%%M:%%S', min(%s)) FROM "%s" WHERE %s = ?`,
			updateColName, tableName, requestEncodingColName),
		requestEncoding,
	)
	if err != nil || rows == nil {
		return time.Time{}, false
	}
	defer rows.Close()
	if !rows.Next() {
		return time.Time{}, false
	}
	var oldest string
	if err = rows.Scan(&oldest); err != nil {
		return time.Time{}, false
	}
	stamp, err := time.Parse("2006-01-02T15:04:05", oldest)
	return stamp, err == nil && rows.Err() == nil
}

//nolint:revive // Liskov substitution principle
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

func (eng *sqLiteSystem) GCPurgeCache() error {
	return eng.gcPurgeCache()
}

func (eng *sqLiteSystem) gcPurgeCache() error {
	query := `
	select distinct 
		'DROP TABLE IF EXISTS "' || name || '" ; ' 
	from sqlite_schema 
	where type = 'table' and name like ?
    and name not like '__iql__%'
    and name not like 'sqlite_%'
    and name not in (select view_name from "__iql__.materialized_views")
    and name not in (select table_name from "__iql__.tables")
	`
	rows, err := eng.sqlEngine.Query(query, eng.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(rows)
}

func (eng *sqLiteSystem) GetOperatorOr() string {
	return "||"
}

func (eng *sqLiteSystem) GetOperatorStringConcat() string {
	return "|"
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
	sb.WriteString(fmt.Sprintf(`CREATE TABLE "%s" ( `, relationName))
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
	sb.WriteString(fmt.Sprintf(`INSERT INTO "%s" ( `, relationName))
	var colzString []string
	for _, col := range colz {
		colzString = append(colzString, fmt.Sprintf(`"%s"`, col.GetName()))
	}
	sb.WriteString(strings.Join(colzString, ", "))
	sb.WriteString(" ) ")
	sb.WriteString(selectQuery)
	return sb.String()
}

//nolint:errcheck,funlen // TODO: establish pattern
func (eng *sqLiteSystem) runMaterializedViewCreate(
	relationName string,
	colz []typing.RelationalColumn,
	rawDDL string,
	replaceAllowed bool,
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
	`
	if replaceAllowed {
		columnQuery += `
		  ON CONFLICT(view_name, column_name)
		  DO
		    UPDATE 
			  SET 
			    column_type = ?,
			    ordinal_position = ?,
			    "oid" = ?,
			    column_width = ?,
			    column_precision = ?
		`
	}
	for i, col := range colz {
		oid, oidExists := col.GetOID()
		if !oidExists {
			oid = 25
		}
		if !replaceAllowed {
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
		} else {
			_, err := txn.Exec(
				columnQuery,
				relationName,
				col.GetName(),
				col.GetType(),
				i+1,
				oid,
				col.GetWidth(),
				0, // TODO: implement precision record
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

func (eng *sqLiteSystem) DelimitFullyQualifiedRelationName(fqtn string) string {
	return fmt.Sprintf(`"%s"`, fqtn)
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
