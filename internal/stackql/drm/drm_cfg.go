package drm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internal_relational_dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/relationaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/sqlmachinery"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_ DRMConfig = &staticDRMConfig{}
)

type DRMConfig interface {
	ColumnsToRelationalColumns(cols []internaldto.ColumnMetadata) []relationaldto.RelationalColumn
	ColumnToRelationalColumn(cols internaldto.ColumnMetadata) relationaldto.RelationalColumn
	ExtractFromGolangValue(interface{}) interface{}
	ExtractObjectFromSQLRows(r *sql.Rows, nonControlColumns []internaldto.ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{})
	GetCurrentTable(internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error)
	GetRelationalType(string) string
	GenerateDDL(util.AnnotatedTabulation, *openapistackql.OperationStore, int, bool) ([]string, error)
	GetControlAttributes() sqlcontrol.ControlAttributes
	GetGolangValue(string) interface{}
	GetGolangSlices([]internaldto.ColumnMetadata) ([]interface{}, []string)
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetParserTableName(internaldto.HeirarchyIdentifiers, int) sqlparser.TableName
	GetSQLSystem() sql_system.SQLSystem
	GetTable(internaldto.HeirarchyIdentifiers, int) (internaldto.DBTable, error)
	GenerateInsertDML(util.AnnotatedTabulation, *openapistackql.OperationStore, internaldto.TxnControlCounters) (PreparedStatementCtx, error)
	GenerateSelectDML(util.AnnotatedTabulation, internaldto.TxnControlCounters, string, string) (PreparedStatementCtx, error)
	ExecuteInsertDML(sqlengine.SQLEngine, PreparedStatementCtx, map[string]interface{}, string) (sql.Result, error)
	OpenapiColumnsToRelationalColumns(cols []openapistackql.ColumnDescriptor) []relationaldto.RelationalColumn
	OpenapiColumnsToRelationalColumn(col openapistackql.ColumnDescriptor) relationaldto.RelationalColumn
	QueryDML(sqlmachinery.Querier, PreparedStatementParameterized) (*sql.Rows, error)
}

type staticDRMConfig struct {
	namespaceCollection tablenamespace.TableNamespaceCollection
	controlAttributes   sqlcontrol.ControlAttributes
	sqlEngine           sqlengine.SQLEngine
	sqlSystem           sql_system.SQLSystem
}

func (dc *staticDRMConfig) GetSQLSystem() sql_system.SQLSystem {
	return dc.sqlSystem
}

func (dc *staticDRMConfig) GetTable(hids internaldto.HeirarchyIdentifiers, discoveryID int) (internaldto.DBTable, error) {
	return dc.sqlSystem.GetTable(hids, discoveryID)
}

func (dc *staticDRMConfig) OpenapiColumnsToRelationalColumns(cols []openapistackql.ColumnDescriptor) []relationaldto.RelationalColumn {
	var relationalColumns []relationaldto.RelationalColumn
	for _, col := range cols {
		var typeStr string
		schemaExists := false
		if col.Schema != nil {
			typeStr = dc.GetRelationalType(col.Schema.Type)
			schemaExists = true
		} else {
			if col.Val != nil {
				switch col.Val.Type {
				case sqlparser.BitVal:
				}
			}
		}
		relationalColumn := relationaldto.NewRelationalColumn(col.Name, typeStr).WithQualifier(col.Qualifier).WithAlias(col.Alias).WithDecorated(col.DecoratedCol).WithParserNode(col.Node)
		if schemaExists {
			inferredOID := internaldto.GetOidForSchema(col.Schema)
			relationalColumn = relationalColumn.WithOID(inferredOID)
		}
		// TODO: Need a way to handle postgres differences. This is a fragile point
		relationalColumns = append(relationalColumns, relationalColumn)
	}
	return relationalColumns
}

func (dc *staticDRMConfig) ToExternalSQLRelationalColumn(tabAnn util.AnnotatedTabulation, colName string) (relationaldto.RelationalColumn, error) {

	return nil, fmt.Errorf("cannot find column '%s' for external SQL table '%s'", colName, tabAnn.GetInputTableName())
}

func (dc *staticDRMConfig) OpenapiColumnsToRelationalColumn(col openapistackql.ColumnDescriptor) relationaldto.RelationalColumn {
	var typeStr string
	schemaExists := false
	if col.Schema != nil {
		typeStr = dc.GetRelationalType(col.Schema.Type)
		schemaExists = true
	} else {
		if col.Val != nil {
			switch col.Val.Type {
			case sqlparser.BitVal:
			}
		}
	}
	decoratedCol := col.DecoratedCol
	// if col.Alias != "" {
	// 	decoratedCol = fmt.Sprintf(`%s AS "%s"`, decoratedCol, col.Alias)
	// }
	relationalColumn := relationaldto.NewRelationalColumn(col.Name, typeStr).WithQualifier(col.Qualifier).WithAlias(col.Alias).WithDecorated(decoratedCol).WithParserNode(col.Node)
	if schemaExists {
		inferredOID := internaldto.GetOidForSchema(col.Schema)
		relationalColumn = relationalColumn.WithOID(inferredOID)
	}
	// TODO: Need a way to handle postgres differences

	return relationalColumn
}

func (dc *staticDRMConfig) ColumnsToRelationalColumns(cols []internaldto.ColumnMetadata) []relationaldto.RelationalColumn {
	var relationalColumns []relationaldto.RelationalColumn
	for _, col := range cols {
		relationalColumn := relationaldto.NewRelationalColumn(col.GetIdentifier(), col.GetRelationalType()).WithAlias(col.GetIdentifier()).WithDecorated(col.GetIdentifier())
		relationalColumns = append(relationalColumns, relationalColumn)
	}
	return relationalColumns
}

func (dc *staticDRMConfig) ColumnToRelationalColumn(col internaldto.ColumnMetadata) relationaldto.RelationalColumn {
	relationalColumn := relationaldto.NewRelationalColumn(col.GetName(), col.GetRelationalType()).WithAlias(col.GetIdentifier())
	return relationalColumn
}

func (dc *staticDRMConfig) GetControlAttributes() sqlcontrol.ControlAttributes {
	return dc.getControlAttributes()
}

func (dc *staticDRMConfig) getControlAttributes() sqlcontrol.ControlAttributes {
	return dc.controlAttributes
}

func (dc *staticDRMConfig) GetGolangSlices(nonControlColumns []internaldto.ColumnMetadata) ([]interface{}, []string) {
	return dc.getGolangSlices(nonControlColumns)
}

func (dc *staticDRMConfig) ExtractObjectFromSQLRows(r *sql.Rows, nonControlColumns []internaldto.ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
	return dc.extractObjectFromSQLRows(r, nonControlColumns, stream)
}

func (dc *staticDRMConfig) extractObjectFromSQLRows(r *sql.Rows, nonControlColumns []internaldto.ColumnMetadata, stream streaming.MapStream) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
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

func (dc *staticDRMConfig) getGolangSlices(nonControlColumns []internaldto.ColumnMetadata) ([]interface{}, []string) {
	i := 0
	var keyArr []string
	var ifArr []interface{}
	for i < len(nonControlColumns) {
		x := nonControlColumns[i]
		y := dc.sqlSystem.GetGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.GetIdentifier())
		i++
	}
	return ifArr, keyArr
}

func (dc *staticDRMConfig) GetRelationalType(discoType string) string {
	return dc.sqlSystem.GetRelationalType(discoType)
}

func (dc *staticDRMConfig) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return dc.namespaceCollection
}

func (dc *staticDRMConfig) GetGolangValue(discoType string) interface{} {
	return dc.sqlSystem.GetGolangValue(discoType)
}

func (dc *staticDRMConfig) ExtractFromGolangValue(val interface{}) interface{} {
	return dc.extractFromGolangValue(val)
}

func (dc *staticDRMConfig) extractFromGolangValue(val interface{}) interface{} {
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

func (dc *staticDRMConfig) GetGolangKind(discoType string) reflect.Kind {
	return dc.sqlSystem.GetGolangKind(discoType)
}

func (dc *staticDRMConfig) GetCurrentTable(tableHeirarchyIDs internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error) {
	tn := tableHeirarchyIDs.GetTableName()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tn) {
		templatedName, err := dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(tn)
		if err != nil {
			return internaldto.NewDBTableAnalytics(templatedName, -1, tableHeirarchyIDs), err
		}
		return internaldto.NewDBTableAnalytics(templatedName, -1, tableHeirarchyIDs), nil
	}
	return dc.sqlSystem.GetCurrentTable(tableHeirarchyIDs)
}

func (dc *staticDRMConfig) GetTableName(hIds internaldto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	return dc.getTableName(hIds, discoveryGenerationID)
}

func (dc *staticDRMConfig) getTableName(hIds internaldto.HeirarchyIdentifiers, discoveryGenerationID int) (string, error) {
	tbl, err := dc.sqlSystem.GetTable(hIds, discoveryGenerationID)
	if err != nil {
		return "", err
	}
	unadornedTableName := tbl.GetNameStump()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(unadornedTableName) {
		return dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(unadornedTableName)
	}
	return tbl.GetName(), nil
}

func (dc *staticDRMConfig) GetParserTableName(hIds internaldto.HeirarchyIdentifiers, discoveryGenerationID int) sqlparser.TableName {
	return dc.getParserTableName(hIds, discoveryGenerationID)
}

func (dc *staticDRMConfig) getParserTableName(hIds internaldto.HeirarchyIdentifiers, discoveryGenerationID int) sqlparser.TableName {
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(hIds.GetTableName()) {
		return sqlparser.TableName{
			Name:            sqlparser.NewTableIdent(hIds.GetResourceStr()),
			Qualifier:       sqlparser.NewTableIdent(hIds.GetServiceStr()),
			QualifierSecond: sqlparser.NewTableIdent(hIds.GetProviderStr()),
		}
	}
	return sqlparser.TableName{
		Name:            sqlparser.NewTableIdent(fmt.Sprintf("generation_%d", discoveryGenerationID)),
		Qualifier:       sqlparser.NewTableIdent(hIds.GetResourceStr()),
		QualifierSecond: sqlparser.NewTableIdent(hIds.GetServiceStr()),
		QualifierThird:  sqlparser.NewTableIdent(hIds.GetProviderStr()),
	}
}

func (dc *staticDRMConfig) inferColType(col util.Column) string {
	relationalType := "text"
	schema := col.GetSchema()
	if schema != nil && schema.Type != "" {
		relationalType = dc.GetRelationalType(schema.Type)
	}
	return relationalType
}

func (dc *staticDRMConfig) genRelationalTableFromExternalSQLTable(tabAnn util.AnnotatedTabulation, discoveryGenerationID int) (relationaldto.RelationalTable, error) {
	tableName, err := dc.getTableName(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	relationalTable := relationaldto.NewRelationalTable(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID, tableName, tabAnn.GetInputTableName())
	tableColumns, err := dc.sqlSystem.ObtainRelationalColumnsFromExternalSQLtable(tabAnn.GetHeirarchyIdentifiers())
	if err != nil {
		return nil, err
	}
	for _, col := range tableColumns {
		relationalTable.PushBackColumn(col)
	}
	return relationalTable, nil
}

func (dc *staticDRMConfig) genRelationalColumnsFromExternalSQLTable(tabAnn util.AnnotatedTabulation, discoveryGenerationID int) (relationaldto.RelationalTable, error) {
	tableName, err := dc.getTableName(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	relationalTable := relationaldto.NewRelationalTable(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID, tableName, tabAnn.GetInputTableName())
	tableColumns, err := dc.sqlSystem.ObtainRelationalColumnsFromExternalSQLtable(tabAnn.GetHeirarchyIdentifiers())
	if err != nil {
		return nil, err
	}
	for _, col := range tableColumns {
		relationalTable.PushBackColumn(col)
	}
	return relationalTable, nil
}

func (dc *staticDRMConfig) genRelationalTable(tabAnn util.AnnotatedTabulation, m *openapistackql.OperationStore, discoveryGenerationID int) (relationaldto.RelationalTable, error) {
	tableName, err := dc.getTableName(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	_, isSQLDataSource := tabAnn.GetSQLDataSource()
	if isSQLDataSource {
		return dc.genRelationalTableFromExternalSQLTable(tabAnn, discoveryGenerationID)
	}
	relationalTable := relationaldto.NewRelationalTable(tabAnn.GetHeirarchyIdentifiers(), discoveryGenerationID, tableName, tabAnn.GetInputTableName())
	schemaAnalyzer := util.NewTableSchemaAnalyzer(tabAnn.GetTabulation().GetSchema(), m)
	tableColumns, err := schemaAnalyzer.GetColumns()
	if err != nil {
		return nil, err
	}
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

func (dc *staticDRMConfig) GenerateDDL(tabAnn util.AnnotatedTabulation, m *openapistackql.OperationStore, discoveryGenerationID int, dropTable bool) ([]string, error) {
	relationalTable, err := dc.genRelationalTable(tabAnn, m, discoveryGenerationID)
	if err != nil {
		return nil, err
	}
	return dc.sqlSystem.GenerateDDL(relationalTable, dropTable)
}

func (dc *staticDRMConfig) GenerateInsertDML(tabAnnotated util.AnnotatedTabulation, method *openapistackql.OperationStore, tcc internaldto.TxnControlCounters) (PreparedStatementCtx, error) {
	var columns []internaldto.ColumnMetadata
	_, isSQLDataSource := tabAnnotated.GetSQLDataSource()
	var tableName string
	var discoverID int
	var err error
	if isSQLDataSource {
		tableObj, err := dc.GetCurrentTable(tabAnnotated.GetHeirarchyIdentifiers())
		tableName = tableObj.GetName()
		discoverID = tableObj.GetDiscoveryID()
		if err != nil {
			return nil, err
		}
	} else {
		tableObj, err := dc.GetCurrentTable(tabAnnotated.GetHeirarchyIdentifiers())
		tableName = tableObj.GetName()
		discoverID = tableObj.GetDiscoveryID()
		if err != nil {
			return nil, err
		}
	}
	genIdColName := dc.controlAttributes.GetControlGenIdColumnName()
	sessionIdColName := dc.controlAttributes.GetControlSsnIdColumnName()
	txnIdColName := dc.controlAttributes.GetControlTxnIdColumnName()
	insIdColName := dc.controlAttributes.GetControlInsIdColumnName()
	insEncodedColName := dc.controlAttributes.GetControlInsertEncodedIdColumnName()

	relationalTable := relationaldto.NewRelationalTable(tabAnnotated.GetHeirarchyIdentifiers(), discoverID, tableName, tabAnnotated.GetInputTableName())
	if isSQLDataSource {
		tableColumns, err := dc.sqlSystem.ObtainRelationalColumnsFromExternalSQLtable(tabAnnotated.GetHeirarchyIdentifiers())
		if err != nil {
			return nil, err
		}
		for _, col := range tableColumns {
			columns = append(columns, internal_relational_dto.NewRelayedColDescriptor(col, col.GetType()))
			relationalTable.PushBackColumn(col)
		}
	} else {
		schemaAnalyzer := util.NewTableSchemaAnalyzer(tabAnnotated.GetTabulation().GetSchema(), method)
		tableColumns, err := schemaAnalyzer.GetColumnDescriptors(tabAnnotated)
		if err != nil {
			return nil, err
		}
		for _, col := range tableColumns {
			relationalType := "text"
			schema := col.Schema
			if schema != nil && schema.Type != "" {
				relationalType = dc.GetRelationalType(schema.Type)
			}
			columns = append(columns, internaldto.NewColDescriptor(col, relationalType))
			relationalColumn := relationaldto.NewRelationalColumn(col.Name, relationalType).WithParserNode(col.Node)
			relationalTable.PushBackColumn(relationalColumn)
		}
	}
	queryString, err := dc.sqlSystem.GenerateInsertDML(relationalTable, tcc)
	if err != nil {
		return nil, err
	}
	return NewPreparedStatementCtx(
			queryString,
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
			dc.sqlSystem,
		),
		nil
}

func (dc *staticDRMConfig) GenerateSelectDML(tabAnnotated util.AnnotatedTabulation, txnCtrlCtrs internaldto.TxnControlCounters, selectSuffix, rewrittenWhere string) (PreparedStatementCtx, error) {
	var quotedColNames []string
	var columns []internaldto.ColumnMetadata

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
		columns = append(columns, internaldto.NewColDescriptor(col, typeStr))
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
	queryString, err := dc.sqlSystem.GenerateSelectDML(relationalTable, txnCtrlCtrs, selectSuffix, rewrittenWhere)

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
		dc.sqlSystem,
	), nil
}

func (dc *staticDRMConfig) generateControlVarArgs(cp PreparedStatementParameterized, isInsert bool) ([]interface{}, error) {
	var varArgs []interface{}
	if cp.IsControlArgsRequired() {
		ctrSlice := cp.GetCtx().GetAllCtrlCtrs()
		for _, ctrs := range ctrSlice {
			if ctrs == nil {
				continue
			}
			varArgs = append(varArgs, ctrs.GetGenID())
			varArgs = append(varArgs, ctrs.GetSessionID())
			varArgs = append(varArgs, ctrs.GetTxnID())
			varArgs = append(varArgs, ctrs.GetInsertID())
			if isInsert {
				varArgs = append(varArgs, cp.GetRequestEncoding())
			}
		}
	}
	return varArgs, nil
}

func (dc *staticDRMConfig) generateVarArgs(cp PreparedStatementParameterized, isInsert bool) (PreparedStatementArgs, error) {
	retVal := NewPreparedStatementArgs(cp.GetCtx().GetQuery())
	for i, child := range cp.GetChildren() {
		chidRv, err := dc.generateVarArgs(child, isInsert)
		if err != nil {
			return retVal, err
		}
		retVal.SetChild(i, chidRv)
	}
	varArgs, _ := dc.generateControlVarArgs(cp, isInsert)
	psArgs := cp.GetArgs()
	if psArgs != nil && len(psArgs) > 0 {
		for _, col := range cp.GetCtx().GetNonControlColumns() {
			va, ok := psArgs[col.GetName()]
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
				if strings.ToLower(col.GetRelationalType()) == "text" && strings.ToLower(dc.sqlSystem.GetName()) == constants.SQLDialectPostgres {
					varArgs = append(varArgs, fmt.Sprintf("%v", va))
					continue
				}
				varArgs = append(varArgs, va)
			}
		}
	}
	retVal.SetArgs(varArgs)
	return retVal, nil
}

func (dc *staticDRMConfig) ExecuteInsertDML(dbEngine sqlengine.SQLEngine, ctx PreparedStatementCtx, payload map[string]interface{}, requestEncoding string) (sql.Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cannot execute on nil PreparedStatementContext")
	}
	stmtArgs, err := dc.generateVarArgs(NewPreparedStatementParameterized(ctx, payload, true).WithRequestEncoding(requestEncoding), true)
	if err != nil {
		return nil, err
	}
	return dbEngine.Exec(stmtArgs.GetQuery(), stmtArgs.GetArgs()...)
}

func (dc *staticDRMConfig) QueryDML(querier sqlmachinery.Querier, ctxParameterized PreparedStatementParameterized) (*sql.Rows, error) {
	if ctxParameterized.GetCtx() == nil {
		return nil, fmt.Errorf("cannot execute based upon nil PreparedStatementContext")
	}
	rootArgs, err := dc.generateVarArgs(ctxParameterized, false)
	if err != nil {
		return nil, err
	}
	err = rootArgs.Analyze()
	if err != nil {
		return nil, err
	}
	query := rootArgs.GetExpandedQuery()
	varArgs := rootArgs.GetExpandedArgs()
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	return querier.Query(query, varArgs...)
}

func GetDRMConfig(sqlSystem sql_system.SQLSystem, namespaceCollection tablenamespace.TableNamespaceCollection, controlAttributes sqlcontrol.ControlAttributes) (DRMConfig, error) {
	rv := &staticDRMConfig{
		namespaceCollection: namespaceCollection,
		controlAttributes:   controlAttributes,
		sqlEngine:           sqlSystem.GetSQLEngine(),
		sqlSystem:           sqlSystem,
	}
	return rv, nil
}
