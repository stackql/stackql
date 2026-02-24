//nolint:dupl,nolintlint,revive //TODO: fix this
package sql_system //nolint:revive,stylecheck // package name is meaningful and readable

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq/oid"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/db/sqlcontrol"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/relationaldto"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/pkg/serde"
)

func newPostgresSystem(
	sqlEngine sqlengine.SQLEngine,
	analyticsNamespaceLikeString string,
	controlAttributes sqlcontrol.ControlAttributes,
	formatter sqlparser.NodeFormatter,
	sqlCfg dto.SQLBackendCfg,
	authCfg map[string]*dto.AuthCtx,
	typCfg typing.Config,
	exportNamepsace string,
) (SQLSystem, error) {
	catalogName, err := sqlCfg.GetDatabaseName()
	if err != nil {
		return nil, err
	}
	tableSchemaName := sqlCfg.GetTableSchemaName()
	if tableSchemaName == "" {
		tableSchemaName = "public"
	}
	rv := &postgresSystem{
		defaultGolangKind:            reflect.String,
		defaultRelationalType:        "text",
		typeCfg:                      typCfg,
		controlAttributes:            controlAttributes,
		analyticsNamespaceLikeString: analyticsNamespaceLikeString,
		sqlEngine:                    sqlEngine,
		formatter:                    formatter,
		tableSchema:                  tableSchemaName,
		tableCatalog:                 catalogName,
		authCfg:                      authCfg,
		exportNamespace:              exportNamepsace,
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

//nolint:unparam // error return val is future proof
func (eng *postgresSystem) inferViewSchemataEnabled(schemataCfg dto.SQLBackendSchemata) (bool, error) {
	if schemataCfg.TableSchema == "" || schemataCfg.OpsViewSchema == "" || schemataCfg.IntelViewSchema == "" {
		return false, nil
	}
	return true, nil
}

type postgresSystem struct {
	controlAttributes            sqlcontrol.ControlAttributes
	analyticsNamespaceLikeString string
	sqlEngine                    sqlengine.SQLEngine
	formatter                    sqlparser.NodeFormatter
	typeCfg                      typing.Config
	defaultRelationalType        string
	defaultGolangKind            reflect.Kind
	tableSchema                  string
	viewSchemataEnabled          bool
	opsViewSchema                string
	intelViewSchema              string
	tableCatalog                 string
	authCfg                      map[string]*dto.AuthCtx
	exportNamespace              string
}

func (eng *postgresSystem) initPostgresEngine() error {
	_, err := eng.sqlEngine.Exec(postgresEngineSetupDDL)
	return err
}

func (eng *postgresSystem) generateDropTableStatement(relationalTable relationaldto.RelationalTable) (string, error) {
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`drop table if exists "%s"`, tableName), nil
}

func (eng *postgresSystem) GetFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return eng.getFullyQualifiedTableName(unqualifiedTableName)
}

func (eng *postgresSystem) IsRelationExported(relationName string) bool {
	if eng.exportNamespace == "" {
		return false
	}
	matches, _ := regexp.MatchString(fmt.Sprintf(`^%s.*$`, eng.exportNamespace), relationName)
	return matches
}

func (eng *postgresSystem) getFullyQualifiedTableName(unqualifiedTableName string) (string, error) {
	return fmt.Sprintf(`"%s"."%s"`, eng.tableSchema, unqualifiedTableName), nil
}

func (eng *postgresSystem) GetASTFormatter() sqlparser.NodeFormatter {
	return eng.formatter
}

func (eng *postgresSystem) GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter {
	return astfuncrewrite.GetPostgresASTFuncRewriter()
}

func (eng *postgresSystem) GenerateDDL(
	relationalTable relationaldto.RelationalTable,
	dropTable bool,
) ([]string, error) {
	return eng.generateDDL(relationalTable, dropTable)
}

func (eng *postgresSystem) RegisterExternalTable(
	connectionName string,
	tableDetails formulation.SQLExternalTable,
) error {
	return eng.registerExternalTable(connectionName, tableDetails)
}

func (eng *postgresSystem) registerExternalTable(
	connectionName string,
	tableDetails formulation.SQLExternalTable,
) error {
	q := `
	INSERT INTO "__iql__.external.columns" (
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
	    $1 
	   ,$2 
	   ,$3 
	   ,$4
	   ,$5 
	   ,$6 
	   ,$7 
	   ,$8 
	   ,$9 
	   ,$10
	 )
	ON CONFLICT (connection_name, catalog_name, schema_name, table_name, column_name) DO NOTHING
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

func (eng *postgresSystem) ObtainRelationalColumnsFromExternalSQLtable(
	hierarchyIDs internaldto.HeirarchyIdentifiers,
) ([]typing.RelationalColumn, error) {
	return eng.obtainRelationalColumnsFromExternalSQLtable(hierarchyIDs)
}

func (eng *postgresSystem) ObtainRelationalColumnFromExternalSQLtable(
	hierarchyIDs internaldto.HeirarchyIdentifiers,
	colName string,
) (typing.RelationalColumn, error) {
	return eng.obtainRelationalColumnFromExternalSQLtable(hierarchyIDs, colName)
}

//nolint:gosec // who cares
func (eng *postgresSystem) obtainRelationalColumnsFromExternalSQLtable(
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
	  connection_name = $1
	  AND
	  catalog_name = $2
	  AND
	  schema_name = $3
	  AND 
	  table_name = $4
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
		//nolint:lll // chained method calls
		relationalColumn := typing.NewRelationalColumn(columnName, columnType).WithWidth(colWidth).WithOID(oid.Oid(oID))
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

func (eng *postgresSystem) getSQLExternalSchema(providerName string) string {
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
func (eng *postgresSystem) obtainRelationalColumnFromExternalSQLtable(
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
	  connection_name = $1
	  AND
	  catalog_name = $2
	  AND
	  schema_name = $3
	  AND 
	  table_name = $4
	  AND
	  column_name = $5
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

func (eng *postgresSystem) SanitizeQueryString(queryString string) (string, error) {
	return eng.sanitizeQueryString(queryString)
}

func (eng *postgresSystem) sanitizeQueryString(queryString string) (string, error) {
	return strings.ReplaceAll(queryString, "`", `"`), nil
}

func (eng *postgresSystem) SanitizeWhereQueryString(queryString string) (string, error) {
	return eng.sanitizeWhereQueryString(queryString)
}

func (eng *postgresSystem) sanitizeWhereQueryString(queryString string) (string, error) {
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(queryString, "`", `"`),
			"||", "OR",
		),
		"|", "||",
	), nil
}

func (eng *postgresSystem) generateViewDDL(
	srcSchemaName string,
	destSchemaName string,
	relationalTable relationaldto.RelationalTable,
) ([]string, error) {
	var colNames, retVal []string
	var createViewBuilder strings.Builder
	retVal = append(retVal, fmt.Sprintf(`drop view if exists "%s"."%s" ; `, destSchemaName, relationalTable.GetBaseName()))
	createViewBuilder.WriteString(
		fmt.Sprintf(
			`create or replace view "%s"."%s" AS `,
			destSchemaName,
			relationalTable.GetBaseName(),
		),
	)
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
	createViewBuilder.WriteString(
		fmt.Sprintf(
			`select %s from "%s"."%s" ;`,
			strings.Join(colNames, ", "),
			srcSchemaName,
			tableName,
		),
	)
	retVal = append(retVal, createViewBuilder.String())
	return retVal, nil
}

//nolint:funlen,lll // TODO: break this one up
func (eng *postgresSystem) generateDDL(relationalTable relationaldto.RelationalTable, dropTable bool) ([]string, error) {
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
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), genIDColName, eng.tableSchema, tableName, genIDColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), sessionIDColName, eng.tableSchema, tableName, sessionIDColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), txnIDColName, eng.tableSchema, tableName, txnIDColName))
	retVal = append(retVal, fmt.Sprintf(`create index if not exists "idx_%s_%s" on "%s"."%s" ( "%s" ) `, strings.ReplaceAll(tableName, ".", "_"), insIDColName, eng.tableSchema, tableName, insIDColName))
	rawViewDDL, err := eng.generateViewDDL(eng.tableSchema, eng.tableSchema, relationalTable)
	if err != nil {
		return nil, err
	}
	retVal = append(retVal, rawViewDDL...)
	if eng.viewSchemataEnabled {
		var intelViewDDL []string
		intelViewDDL, err = eng.generateViewDDL(eng.tableSchema, eng.intelViewSchema, relationalTable)
		if err != nil {
			return nil, err
		}
		retVal = append(retVal, intelViewDDL...)
	}
	return retVal, nil
}

func (eng *postgresSystem) DropView(viewName string) error {
	_, err := eng.sqlEngine.Exec(`delete from "__iql__.views" where view_name = $1`, viewName)
	return err
}

func (eng *postgresSystem) CreateView(
	viewName string, rawDDL string, replaceAllowed bool, requiredParams []string) error {
	return eng.createView(viewName, rawDDL, replaceAllowed, requiredParams)
}

func (eng *postgresSystem) createView(
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
		$1,
		$2,
		$3
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

func (eng *postgresSystem) GetViewByName(viewName string) (internaldto.RelationDTO, bool) {
	rv, ok := eng.getViewByName(viewName)
	if !ok {
		return nil, false
	}
	candidates, err := eng.getAwareViewsByName(fmt.Sprintf("%s%%", viewName))
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

func (eng *postgresSystem) GetViewByNameAndParameters(
	viewName string, params map[string]any) (internaldto.RelationDTO, bool) {
	rv, err := eng.selectMatchingView(viewName, params)
	if err != nil {
		return nil, false
	}
	return rv, true
}

func (eng *postgresSystem) getViewByName(viewName string) (internaldto.RelationDTO, bool) {
	q := `SELECT view_ddl, required_params FROM "__iql__.views" WHERE view_name = $1 and deleted_dttm IS NULL`
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

func (eng *postgresSystem) selectMatchingView(viewName string, params map[string]any) (internaldto.RelationDTO, error) {
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

func (eng *postgresSystem) getAwareViewsByName(viewName string) ([]internaldto.RelationDTO, error) {
	q := `SELECT view_name, view_ddl, required_params 
	FROM "__iql__.views" WHERE view_name LIKE $1 and deleted_dttm IS NULL`
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

func (eng *postgresSystem) CreateMaterializedView(
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
func (eng *postgresSystem) RefreshMaterializedView(naiveViewName string,
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
	// TODO: check colz against supplied columns
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

func (eng *postgresSystem) getExportSchemaCreateQuery() (string, bool) {
	if eng.exportNamespace == "" {
		return "", false
	}
	return fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", eng.exportNamespace), true
}

//nolint:errcheck,revive,staticcheck // TODO: establish pattern
func (eng *postgresSystem) InsertIntoPhysicalTable(naiveTableName string,
	columnsString string,
	selectQuery string,
	varargs ...any) error {
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveTableName)
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	// TODO: check colz against supplied columns
	relationDTO, relationDTOok := eng.getTableByName(naiveTableName, txn)
	if !relationDTOok {
		if len(relationDTO.GetColumns()) == 0 {
		}
		// no need to rollbak; assumed already done
		return fmt.Errorf("cannot refresh materialized view = '%s': not found", fullyQualifiedRelationName)
	}
	// guard against systemic weirdness and trust users not to be weird
	if naiveTableName == fullyQualifiedRelationName {
		fullyQualifiedRelationName = fmt.Sprintf(`"%s"`, naiveTableName)
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

func (eng *postgresSystem) DropMaterializedView(naiveViewName string) error {
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveViewName)
	dropRefQuery := `
	DELETE FROM "__iql__.materialized_views"
	WHERE view_name = $1
	`
	dropColsQuery := `
	DELETE
	FROM
	  "__iql__.materialized_views.columns"
	WHERE
	  view_name = $1
	`
	// TODO: clean up this legacy guard
	delimitedRelationName := fullyQualifiedRelationName
	if fullyQualifiedRelationName == naiveViewName {
		delimitedRelationName = fmt.Sprintf(`"%s"`, fullyQualifiedRelationName)
	}
	dropTableQuery := fmt.Sprintf(`
	DROP TABLE IF EXISTS %s
	`, delimitedRelationName)
	tx, err := eng.sqlEngine.GetTx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(dropRefQuery, fullyQualifiedRelationName)
	if err != nil {
		//nolint:errcheck // TODO: merge variadic error(s) into one
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(dropColsQuery, fullyQualifiedRelationName)
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
	commitErr := tx.Commit()
	return commitErr
}

//nolint:errcheck // TODO: establish pattern
func (eng *postgresSystem) GetMaterializedViewByName(viewName string) (internaldto.RelationDTO, bool) {
	txn, err := eng.sqlEngine.GetTx()
	if err != nil {
		return nil, false
	}
	rv, ok := eng.getMaterializedViewByName(viewName, txn)
	txn.Commit()
	return rv, ok
}

//nolint:errcheck,gosec // TODO: establish pattern
func (eng *postgresSystem) getMaterializedViewByName(
	naiveViewName string, txn *sql.Tx) (internaldto.RelationDTO, bool) {
	fullyQualifiedRelationName := eng.getFullyQualifiedRelationName(naiveViewName)
	q := `SELECT view_ddl FROM "__iql__.materialized_views" WHERE view_name = $1 and deleted_dttm IS NULL`
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
	  view_name = $1
	ORDER BY ordinal_position ASC
	`
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
func (eng *postgresSystem) GetPhysicalTableByName(
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
func (eng *postgresSystem) getTableByName(
	naiveTableName string,
	txn *sql.Tx) (internaldto.RelationDTO, bool) {
	fullyQualifiedTableName := eng.getFullyQualifiedRelationName(naiveTableName)
	q := `SELECT table_ddl FROM "__iql__.tables" WHERE table_name = $1 and deleted_dttm IS NULL`
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
	  table_name = $1
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
func (eng *postgresSystem) DropPhysicalTable(naiveTableName string,
	ifExists bool,
) error {
	fullyQualifiedTableName := eng.getFullyQualifiedRelationName(naiveTableName)
	dropRefQuery := `
	DELETE FROM "__iql__.tables"
	WHERE table_name = $1
	`
	// TODO: clean up this legacy guard
	delimitedTableName := fullyQualifiedTableName
	if fullyQualifiedTableName == naiveTableName {
		delimitedTableName = fmt.Sprintf(`"%s"`, fullyQualifiedTableName)
	}
	dropTableQuery := fmt.Sprintf(`
	DROP TABLE %s
	`, delimitedTableName)
	if ifExists {
		dropTableQuery = fmt.Sprintf(`
		DROP TABLE IF EXISTS %s
		`, delimitedTableName)
	}
	dropColsQuery := `
	DELETE
	FROM
	  "__iql__.tables.columns"
	WHERE
	  table_name = $1
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
	_, err = tx.Exec(dropColsQuery, fullyQualifiedTableName)
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
	commitErr := tx.Commit()
	return commitErr
}

// TODO: implement temp table creation
func (eng *postgresSystem) CreatePhysicalTable(
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

func (eng *postgresSystem) GetFullyQualifiedRelationName(tableName string) string {
	return eng.getFullyQualifiedRelationName(tableName)
}

func (eng *postgresSystem) DelimitFullyQualifiedRelationName(fqtn string) string {
	return fqtn
}

func (eng *postgresSystem) getFullyQualifiedRelationName(tableName string) string {
	if eng.exportNamespace == "" {
		return tableName
	}
	return fmt.Sprintf("%s.%s", eng.exportNamespace, tableName)
}

func (eng *postgresSystem) GetGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
	return eng.getGCHousekeepingQuery(tableName, tcc)
}

func (eng *postgresSystem) getGCHousekeepingQuery(tableName string, tcc internaldto.TxnControlCounters) string {
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

func (eng *postgresSystem) DelimitGroupByColumn(term string) string {
	return eng.quoteWrapTerm(term)
}

func (eng *postgresSystem) DelimitOrderByColumn(term string) string {
	return eng.quoteWrapTerm(term)
}

func (eng *postgresSystem) quoteWrapTerm(term string) string {
	return fmt.Sprintf(`"%s"`, term)
}

func (eng *postgresSystem) render(alias string,
	controlIterationCount int, aliasToCountersMap map[string][]internaldto.TxnControlCounters) string {
	j := controlIterationCount * constants.ControlColumnCount
	genIDColName := eng.controlAttributes.GetControlGenIDColumnName()
	sessionIDColName := eng.controlAttributes.GetControlSsnIDColumnName()
	txnIDColName := eng.controlAttributes.GetControlTxnIDColumnName()
	insIDColName := eng.controlAttributes.GetControlInsIDColumnName()
	var controls []string
	if alias != "" {
		for range aliasToCountersMap[alias] {
			gIDcn := fmt.Sprintf(`"%s"."%s"`, alias, genIDColName)
			sIDcn := fmt.Sprintf(`"%s"."%s"`, alias, sessionIDColName)
			tIDcn := fmt.Sprintf(`"%s"."%s"`, alias, txnIDColName)
			iIDcn := fmt.Sprintf(`"%s"."%s"`, alias, insIDColName)
			//nolint:lll // better expressed compactly
			controls = append(controls, fmt.Sprintf(`%s = $%d AND %s = $%d AND %s = $%d AND %s = $%d`, gIDcn, j+1, sIDcn, j+2, tIDcn, j+3, iIDcn, j+4)) //nolint:mnd // the magic numbers are offsets
			j += constants.ControlColumnCount
		}
		return fmt.Sprintf(`( %s )`, strings.Join(controls, " OR "))
	}
	for range aliasToCountersMap[alias] {
		gIDcn := fmt.Sprintf(`"%s"`, genIDColName)
		sIDcn := fmt.Sprintf(`"%s"`, sessionIDColName)
		tIDcn := fmt.Sprintf(`"%s"`, txnIDColName)
		iIDcn := fmt.Sprintf(`"%s"`, insIDColName)
		//nolint:lll // better expressed compactly
		controls = append(controls, fmt.Sprintf(`%s = $%d AND %s = $%d AND %s = $%d AND %s = $%d`, gIDcn, j+1, sIDcn, j+2, tIDcn, j+3, iIDcn, j+4)) //nolint:mnd // the magic numbers are offsets
		j += constants.ControlColumnCount
	}
	if len(controls) == 0 {
		return "1 = 1"
	}
	return fmt.Sprintf(`( %s )`, strings.Join(controls, " OR "))
}

func (eng *postgresSystem) ComposeSelectQuery(
	columns []typing.RelationalColumn,
	tableAliases []string,
	hoistedTableAliases []string,
	fromString string,
	rewrittenWhere string,
	selectQualifier string,
	selectSuffix string,
	parameterOffset int,
	aliasToCountersMap map[string][]internaldto.TxnControlCounters,
) (string, error) {
	return eng.composeSelectQuery(
		columns, tableAliases,
		hoistedTableAliases, fromString, rewrittenWhere, selectQualifier, selectSuffix, parameterOffset, aliasToCountersMap)
}

func (eng *postgresSystem) composeSelectQuery(
	columns []typing.RelationalColumn,
	tableAliases []string,
	hoistedTableAliases []string,
	fromString string,
	rewrittenWhere string,
	selectQualifier string,
	selectSuffix string,
	parameterOffset int,
	aliasToCountersMap map[string][]internaldto.TxnControlCounters,
) (string, error) {
	var q strings.Builder
	var quotedColNames []string
	for _, col := range columns {
		quotedColNames = append(quotedColNames, col.DelimitedSelectionString(`"`))
	}
	var wq strings.Builder
	var controlWhereComparisons []string
	i := parameterOffset
	var hoistedControlOnComparisons []any
	if len(hoistedTableAliases) > 0 {
		for _, alias := range hoistedTableAliases {
			hoistedControlOnComparisons = append(
				hoistedControlOnComparisons,
				eng.render(alias, i, aliasToCountersMap),
			)
			i += len(aliasToCountersMap[alias])
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
	for _, alias := range tableAliases {
		controlWhereComparisons = append(controlWhereComparisons, eng.render(alias, i, aliasToCountersMap))
		i += len(aliasToCountersMap[alias])
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

	q.WriteString(fmt.Sprintf(`SELECT %s %s `, selectQualifier, strings.Join(quotedColNames, ", ")))
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

func (eng *postgresSystem) GenerateInsertDML(
	relationalTable relationaldto.RelationalTable,
	tcc internaldto.TxnControlCounters,
) (string, error) {
	return eng.generateInsertDML(relationalTable, tcc)
}

//nolint:unparam // future proof
func (eng *postgresSystem) generateInsertDML(
	relationalTable relationaldto.RelationalTable,
	tcc internaldto.TxnControlCounters, //nolint:revive // future proof
) (string, error) {
	var q strings.Builder
	var quotedColNames, vals []string
	tableName, err := relationalTable.GetName()
	if err != nil {
		return "", err
	}
	q.WriteString(fmt.Sprintf(`INSERT INTO "%s"."%s" `, eng.tableSchema, tableName))
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
	vals = append(vals, "$1")
	vals = append(vals, "$2")
	vals = append(vals, "$3")
	vals = append(vals, "$4")
	vals = append(vals, "$5")
	i := 1
	for _, col := range relationalTable.GetColumns() {
		quotedColNames = append(quotedColNames, `"`+col.GetName()+`" `)
		if strings.ToLower(col.GetType()) != "text" {
			vals = append(vals, fmt.Sprintf("$%d", 5+i)) //nolint:mnd // the magic number is an offset
		} else {
			vals = append(vals, fmt.Sprintf("CAST($%d AS TEXT)", 5+i)) //nolint:mnd // the magic number is an offset
		}
		i++
	}
	q.WriteString(fmt.Sprintf(" (%s) ", strings.Join(quotedColNames, ", ")))
	q.WriteString(fmt.Sprintf(" VALUES (%s) ", strings.Join(vals, ", ")))
	return q.String(), nil
}

func (eng *postgresSystem) GenerateSelectDML(
	relationalTable relationaldto.RelationalTable,
	txnCtrlCtrs internaldto.TxnControlCounters,
	selectSuffix,
	rewrittenWhere string,
) (string, error) {
	return eng.generateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)
}

func (eng *postgresSystem) generateSelectDML(
	relationalTable relationaldto.RelationalTable,
	txnCtrlCtrs internaldto.TxnControlCounters, //nolint:unparam,revive // future proof
	selectSuffix,
	rewrittenWhere string,
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
	q.WriteString(
		fmt.Sprintf(
			`SELECT %s FROM "%s"."%s" %s WHERE `,
			strings.Join(
				quotedColNames,
				", ",
			),
			eng.tableSchema,
			tableName,
			aliasStr,
		),
	)
	q.WriteString(
		fmt.Sprintf(
			`( "%s" = $1 AND "%s" = $2 AND "%s" = $3 AND "%s" = $4 ) `,
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

func (eng *postgresSystem) GCAdd(
	tableName string,
	parentTcc, //nolint:revive // future proof
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

func (eng *postgresSystem) GCCollectObsoleted(minTransactionID int) error {
	return eng.gCCollectObsoleted(minTransactionID)
}

func (eng *postgresSystem) gCCollectObsoleted(minTransactionID int) error {
	maxTxnColName := eng.controlAttributes.GetControlMaxTxnColumnName()
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
		eng.tableSchema,
		maxTxnColName,
		minTransactionID,
	)
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery, eng.tableCatalog, eng.tableSchema)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *postgresSystem) GCCollectAll() error {
	return eng.gCCollectAll()
}

func (eng *postgresSystem) GetOperatorOr() string {
	return "OR"
}

func (eng *postgresSystem) GetOperatorStringConcat() string {
	return "||"
}

func (eng *postgresSystem) gCCollectAll() error {
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
		eng.tableSchema,
	)
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery, eng.tableCatalog, eng.tableSchema)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *postgresSystem) GCControlTablesPurge() error {
	return eng.gcControlTablesPurge()
}

func (eng *postgresSystem) IsTablePresent(
	tableName string,
	requestEncoding string,
	colName string, //nolint:revive // future proof
) bool {
	rows, err := eng.sqlEngine.Query( //nolint:rowserrcheck // TODO: fix this
		fmt.Sprintf(
			`SELECT count(*) as ct FROM "%s"."%s" WHERE iql_insert_encoded = $1 `,
			eng.tableSchema,
			tableName,
		),
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

// In Postgres, `Timestamp with time zone` objects are timezone-aware.
func (eng *postgresSystem) TableOldestUpdateUTC(
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
			"SELECT min(%s) as oldest_update, %s, %s, %s, %s FROM \"%s\".\"%s\" WHERE %s = '%s' GROUP BY %s, %s, %s, %s;",
			updateColName,
			genIDColName,
			ssnIDColName,
			txnIDColName,
			insIDColName,
			eng.tableSchema,
			tableName,
			requestEncodingColName,
			requestEncoding,
			genIDColName,
			ssnIDColName,
			txnIDColName,
			insIDColName,
		),
	)
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

func (eng *postgresSystem) gcControlTablesPurge() error {
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
		eng.tableSchema,
	)
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery, eng.tableCatalog, eng.tableSchema)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *postgresSystem) GCPurgeEphemeral() error {
	return eng.gcPurgeEphemeral()
}

func (eng *postgresSystem) GCPurgeCache() error {
	return eng.gcPurgeCache()
}

func (eng *postgresSystem) GetName() string {
	return constants.SQLDialectPostgres
}

func (eng *postgresSystem) gcPurgeCache() error {
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
	rows, err := eng.sqlEngine.Query(query, eng.tableCatalog, eng.tableSchema, eng.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(rows)
}

func (eng *postgresSystem) gcPurgeEphemeral() error {
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
	rows, err := eng.sqlEngine.Query(query, eng.tableCatalog, eng.tableSchema, eng.analyticsNamespaceLikeString)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(rows)
}

func (eng *postgresSystem) PurgeAll() error {
	return eng.purgeAll()
}

func (eng *postgresSystem) GetSQLEngine() sqlengine.SQLEngine {
	return eng.sqlEngine
}

func (eng *postgresSystem) purgeAll() error {
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
	deleteQueryResultSet, err := eng.sqlEngine.Query(obtainQuery, eng.tableCatalog, eng.tableSchema)
	if err != nil {
		return err
	}
	return eng.readExecGeneratedQueries(deleteQueryResultSet)
}

func (eng *postgresSystem) readExecGeneratedQueries(queryResultSet *sql.Rows) error {
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

func (eng *postgresSystem) GetRelationalType(discoType string) string {
	return eng.getRelationalType(discoType)
}

func (eng *postgresSystem) getRelationalType(discoType string) string {
	return eng.typeCfg.GetRelationalType(discoType)
}

func (eng *postgresSystem) GetGolangValue(discoType string) interface{} {
	return eng.getGolangValue(discoType)
}

func (eng *postgresSystem) getGolangValue(discoType string) interface{} {
	return eng.typeCfg.GetGolangValue(discoType)
}

func (eng *postgresSystem) GetGolangKind(discoType string) reflect.Kind {
	return eng.typeCfg.GetGolangKind(discoType)
}

func (eng *postgresSystem) QueryNamespaced(
	colzString string,
	actualTableName string,
	requestEncodingColName string,
	requestEncoding string,
) (*sql.Rows, error) {
	return eng.sqlEngine.Query(
		fmt.Sprintf(
			`SELECT %s FROM "%s"."%s" WHERE "%s" = $1`,
			colzString,
			eng.tableSchema,
			actualTableName,
			requestEncodingColName,
		),
		requestEncoding,
	)
}

func (eng *postgresSystem) GetTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
	discoveryID int,
) (internaldto.DBTable, error) {
	return eng.getTable(tableHeirarchyIDs, discoveryID)
}

func (eng *postgresSystem) getTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
	discoveryID int,
) (internaldto.DBTable, error) {
	tableNameStump, err := eng.getTableNameStump(tableHeirarchyIDs)
	if err != nil {
		return internaldto.NewDBTable("", "", "", 0, tableHeirarchyIDs), err
	}
	tableName := fmt.Sprintf("%s.generation_%d", tableNameStump, discoveryID)
	return internaldto.NewDBTable(
		tableName, tableNameStump,
		tableHeirarchyIDs.GetTableName(),
		discoveryID,
		tableHeirarchyIDs,
	).WithNameSpace(eng.tableSchema), err
}

func (eng *postgresSystem) GetCurrentTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
) (internaldto.DBTable, error) {
	return eng.getCurrentTable(tableHeirarchyIDs)
}

// In postgres, 63 chars is default length for IDs such as table names
// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS
//
//nolint:unparam // future proof
func (eng *postgresSystem) getTableNameStump(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (string, error) {
	rawTableName := tableHeirarchyIDs.GetTableName()
	maxRawTableNameWidth := constants.PostgresIDMaxWidth - (len(".generation_") + constants.MaxDigits32BitUnsigned)
	if len(rawTableName) > maxRawTableNameWidth {
		return rawTableName[:maxRawTableNameWidth], nil
	}
	return rawTableName, nil
}

func (eng *postgresSystem) getCurrentTable(
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
	res := eng.sqlEngine.QueryRow(`
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
		logging.GetLogger().Errorln(
			fmt.Sprintf(
				"err = %v for tableNamePattern = '%s' and tableNameLHSRemove = '%s'",
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
	), err
}

func (eng *postgresSystem) QueryMaterializedView(
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

func (eng *postgresSystem) generateTableDDL(
	relationName string,
	colz []typing.RelationalColumn,
) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(`CREATE TABLE %s ( `, relationName))
	var colzString []string
	for _, col := range colz {
		colType := col.GetType()
		if colType == "" {
			colType = "TEXT"
		}
		colzString = append(colzString, fmt.Sprintf(`"%s" %s`, col.GetName(), colType))
	}
	sb.WriteString(strings.Join(colzString, ", "))
	sb.WriteString(" ) ")
	return sb.String()
}

// func (eng *postgresSystem) generateMaterializedViewDDL(
// 	relationName string,
// 	selectQuery string,
// ) string {
// 	sb := strings.Builder{}
// 	sb.WriteString(fmt.Sprintf(`CREATE MATERIALIZED VIEW "%s" AS %s`, relationName, selectQuery))
// 	return sb.String()
// }

func (eng *postgresSystem) generateTableInsertDMLFromViewSelect(
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

//nolint:errcheck,funlen // TODO: establish pattern
func (eng *postgresSystem) runMaterializedViewCreate(
	naiveRelationName string,
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
	exportSchemaCreateQuery, isExportSchemaCreateQueryRequired := eng.getExportSchemaCreateQuery()
	if isExportSchemaCreateQueryRequired {
		_, txnErr = txn.Exec(exportSchemaCreateQuery)
		if txnErr != nil {
			txn.Rollback()
			return txnErr
		}
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
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7
	  )
	`
	if replaceAllowed {
		columnQuery += `
		  ON CONFLICT(view_name, column_name)
		  DO
		    UPDATE 
			  SET 
			    column_type = $3,
			    ordinal_position = $4,
			    "oid" = $5,
			    column_width = $6,
			    column_precision = $7
		`
	}
	for i, col := range colz {
		oid, oidExists := col.GetOID()
		if !oidExists {
			oid = 25
		}
		_, err := txn.Exec(
			columnQuery,
			naiveRelationName,
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
	tableDDL := eng.generateTableDDL(naiveRelationName, colz)
	_, err := txn.Exec(tableDDL)
	if err != nil {
		txn.Rollback()
		return err
	}
	insertQuery := eng.generateTableInsertDMLFromViewSelect(naiveRelationName, selectQuery, colz)
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
		$1,
		$2,
		$3,
		''
	  )
	  ;
	  `
	_, err = txn.Exec(
		relationCatalogueQuery,
		naiveRelationName,
		rawDDL,
		insertQuery,
	)
	if err != nil {
		txn.Rollback()
		return err
	}
	commitErr := txn.Commit()
	return commitErr
}

//nolint:errcheck // TODO: establish pattern
func (eng *postgresSystem) runPhysicalTableCreate(
	relationName string,
	colz []typing.RelationalColumn,
	rawDDL string,
	ifNotExists bool, //nolint:unparam,revive // future proof
) error {
	txn, txnErr := eng.sqlEngine.GetTx()
	if txnErr != nil {
		return txnErr
	}
	exportSchemaCreateQuery, isExportSchemaCreateQueryRequired := eng.getExportSchemaCreateQuery()
	if isExportSchemaCreateQueryRequired {
		_, txnErr = txn.Exec(exportSchemaCreateQuery)
		if txnErr != nil {
			txn.Rollback()
			return txnErr
		}
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
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7
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
		$1,
		$2
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
