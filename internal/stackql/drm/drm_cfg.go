package drm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	sqlcontrol "github.com/stackql/stackql/internal/stackql/sqlengine/control"
	"github.com/stackql/stackql/internal/stackql/sqlmachinery"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/typing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const (
	textStr = "text"
)

var (
	_ Config = &staticDRMConfig{}
)

type Config interface {
	ColumnsToRelationalColumns(cols []typing.ColumnMetadata) []typing.RelationalColumn
	ColumnToRelationalColumn(cols typing.ColumnMetadata) typing.RelationalColumn
	ExtractFromGolangValue(interface{}) interface{}
	ExtractObjectFromSQLRows(
		r *sql.Rows,
		nonControlColumns []typing.ColumnMetadata,
		stream streaming.MapStream,
	) (map[string]map[string]interface{}, map[int]map[int]interface{})
	GetCurrentTable(internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error)
	GetRelationalType(string) string

	GetControlAttributes() sqlcontrol.ControlAttributes
	GetGolangValue(string) interface{}
	GetGolangSlices([]typing.ColumnMetadata) ([]interface{}, []string)
	GetNamespaceCollection() tablenamespace.Collection
	GetParserTableName(internaldto.HeirarchyIdentifiers, int) sqlparser.TableName
	GetSQLSystem() sql_system.SQLSystem
	GetTable(internaldto.HeirarchyIdentifiers, int) (internaldto.DBTable, error)

	ExecuteInsertDML(sqlengine.SQLEngine, PreparedStatementCtx, map[string]interface{}, string) (sql.Result, error)

	QueryDML(sqlmachinery.Querier, PreparedStatementParameterized) (*sql.Rows, error)
	ExecDDL(
		querier sqlmachinery.ExecQuerier,
		ctxParameterized PreparedStatementParameterized,
	) (sql.Result, error)
	CreateMaterializedView(
		relationName string,
		rawDDL string,
		ctxParameterized PreparedStatementParameterized,
		replaceAllowed bool,
	) error
	RefreshMaterializedView(
		relationName string,
		ctxParameterized PreparedStatementParameterized,
	) error
	// This one the DDL is ahead of time so table name already aware; it is the exception
	CreatePhysicalTable(
		fullyQualifiedRelationName string,
		rawDDL string,
		tableSpec *sqlparser.TableSpec,
		ifNotExists bool,
	) error
	InsertIntoPhysicalTable(
		relationName string,
		insertColumnsString string,
		ctxParameterized PreparedStatementParameterized,
	) error
	GetFullyQualifiedRelationName(string) string
	DelimitFullyQualifiedRelationName(string) string
}

type staticDRMConfig struct {
	namespaceCollection tablenamespace.Collection
	controlAttributes   sqlcontrol.ControlAttributes
	sqlEngine           sqlengine.SQLEngine
	sqlSystem           sql_system.SQLSystem
	typCfg              typing.Config
}

func (dc *staticDRMConfig) GetSQLSystem() sql_system.SQLSystem {
	return dc.sqlSystem
}

func (dc *staticDRMConfig) GetTable(
	hids internaldto.HeirarchyIdentifiers,
	discoveryID int,
) (internaldto.DBTable, error) {
	return dc.sqlSystem.GetTable(hids, discoveryID)
}

func (dc *staticDRMConfig) GetFullyQualifiedRelationName(relationName string) string {
	return dc.sqlSystem.GetFullyQualifiedRelationName(relationName)
}

func (dc *staticDRMConfig) DelimitFullyQualifiedRelationName(fqtn string) string {
	return dc.sqlSystem.DelimitFullyQualifiedRelationName(fqtn)
}

func (dc *staticDRMConfig) translateColDefTypeToRelationalType(
	col *sqlparser.ColumnDefinition) typing.RelationalColumn {
	relationalColumn := typing.NewRelationalColumn(
		col.Name.GetRawVal(),
		col.Type.Type,
	).WithOID(typing.GetOidForParserColType(col.Type))
	return relationalColumn
}

func (dc *staticDRMConfig) translateColumns(colz []*sqlparser.ColumnDefinition) []typing.RelationalColumn {
	var relationalColumns []typing.RelationalColumn
	for _, col := range colz {
		relationalColumn := dc.translateColDefTypeToRelationalType(col)
		relationalColumns = append(relationalColumns, relationalColumn)
	}
	return relationalColumns
}

func (dc *staticDRMConfig) ColumnsToRelationalColumns(
	cols []typing.ColumnMetadata,
) []typing.RelationalColumn {
	var relationalColumns []typing.RelationalColumn
	for _, col := range cols {
		relationalColumn := typing.NewRelationalColumn(
			col.GetIdentifier(), col.GetRelationalType()).WithAlias(col.GetIdentifier()).WithDecorated(col.GetIdentifier())
		relationalColumns = append(relationalColumns, relationalColumn)
	}
	return relationalColumns
}

func (dc *staticDRMConfig) ColumnToRelationalColumn(
	col typing.ColumnMetadata,
) typing.RelationalColumn {
	relationalColumn := typing.NewRelationalColumn(
		col.GetName(),
		col.GetRelationalType(),
	).WithAlias(col.GetIdentifier())
	return relationalColumn
}

func (dc *staticDRMConfig) GetControlAttributes() sqlcontrol.ControlAttributes {
	return dc.getControlAttributes()
}

func (dc *staticDRMConfig) getControlAttributes() sqlcontrol.ControlAttributes {
	return dc.controlAttributes
}

func (dc *staticDRMConfig) GetGolangSlices(nonControlColumns []typing.ColumnMetadata) ([]interface{}, []string) {
	return dc.getGolangSlices(nonControlColumns)
}

func (dc *staticDRMConfig) ExtractObjectFromSQLRows(
	r *sql.Rows,
	nonControlColumns []typing.ColumnMetadata,
	stream streaming.MapStream,
) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
	return dc.extractObjectFromSQLRows(r, nonControlColumns, stream)
}

func (dc *staticDRMConfig) extractObjectFromSQLRows(
	r *sql.Rows,
	nonControlColumns []typing.ColumnMetadata,
	stream streaming.MapStream,
) (map[string]map[string]interface{}, map[int]map[int]interface{}) {
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
				logging.GetLogger().Infoln(
					fmt.Sprintf(
						"col #%d '%s':  %v  type: %T",
						ord,
						nonControlColumns[ord].GetName(),
						val,
						val,
					),
				)
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
			stream.Write([]map[string]interface{}{im}) //nolint:errcheck // TODO: Refactor this function to return an error
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

func (dc *staticDRMConfig) getGolangSlices(nonControlColumns []typing.ColumnMetadata) ([]interface{}, []string) {
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

func (dc *staticDRMConfig) GetNamespaceCollection() tablenamespace.Collection {
	return dc.namespaceCollection
}

func (dc *staticDRMConfig) GetGolangValue(discoType string) interface{} {
	return dc.sqlSystem.GetGolangValue(discoType)
}

func (dc *staticDRMConfig) ExtractFromGolangValue(val interface{}) interface{} {
	return dc.typCfg.ExtractFromGolangValue(val)
}

func (dc *staticDRMConfig) GetGolangKind(discoType string) reflect.Kind {
	return dc.sqlSystem.GetGolangKind(discoType)
}

func (dc *staticDRMConfig) GetCurrentTable(
	tableHeirarchyIDs internaldto.HeirarchyIdentifiers,
) (internaldto.DBTable, error) {
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

func (dc *staticDRMConfig) GetTableName(
	hIDs internaldto.HeirarchyIdentifiers,
	discoveryGenerationID int,
) (string, error) {
	return dc.getTableName(hIDs, discoveryGenerationID)
}

func (dc *staticDRMConfig) getTableName(
	hIDs internaldto.HeirarchyIdentifiers,
	discoveryGenerationID int,
) (string, error) {
	tbl, err := dc.sqlSystem.GetTable(hIDs, discoveryGenerationID)
	if err != nil {
		return "", err
	}
	unadornedTableName := tbl.GetNameStump()
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(unadornedTableName) {
		return dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().RenderTemplate(unadornedTableName)
	}
	return tbl.GetName(), nil
}

func (dc *staticDRMConfig) GetParserTableName(
	hIDs internaldto.HeirarchyIdentifiers,
	discoveryGenerationID int,
) sqlparser.TableName {
	return dc.getParserTableName(hIDs, discoveryGenerationID)
}

func (dc *staticDRMConfig) getParserTableName(
	hIDs internaldto.HeirarchyIdentifiers,
	discoveryGenerationID int,
) sqlparser.TableName {
	if dc.namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(hIDs.GetTableName()) {
		return sqlparser.TableName{
			Name:            sqlparser.NewTableIdent(hIDs.GetResourceStr()),
			Qualifier:       sqlparser.NewTableIdent(hIDs.GetServiceStr()),
			QualifierSecond: sqlparser.NewTableIdent(hIDs.GetProviderStr()),
		}
	}
	return sqlparser.TableName{
		Name:            sqlparser.NewTableIdent(fmt.Sprintf("generation_%d", discoveryGenerationID)),
		Qualifier:       sqlparser.NewTableIdent(hIDs.GetResourceStr()),
		QualifierSecond: sqlparser.NewTableIdent(hIDs.GetServiceStr()),
		QualifierThird:  sqlparser.NewTableIdent(hIDs.GetProviderStr()),
	}
}

//nolint:unparam // defer fixing this
func (dc *staticDRMConfig) generateControlVarArgs(
	cp PreparedStatementParameterized,
	isInsert bool,
) ([]interface{}, error) {
	var varArgs []interface{}
	if cp.IsControlArgsRequired() {
		ctrSlice := cp.GetCtx().GetOrderedTccs()
		legacyCtrSlice := cp.GetCtx().GetAllCtrlCtrs()
		if len(ctrSlice) < len(legacyCtrSlice) {
			ctrSlice = legacyCtrSlice
		}
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

//nolint:gocognit // defer fixing this
func (dc *staticDRMConfig) generateVarArgs(
	cp PreparedStatementParameterized,
	isInsert bool,
	controlCount int,
) (PreparedStatementArgs, error) {
	retVal := NewPreparedStatementArgs(cp.GetCtx().GetQuery())
	for i, child := range cp.GetChildren() {
		chidRv, err := dc.generateVarArgs(child, isInsert, child.GetCtx().GetCtrlColumnRepeats())
		if err != nil {
			return retVal, err
		}
		retVal.SetChild(i, chidRv)
	}
	varArgs, _ := dc.generateControlVarArgs(cp, isInsert)
	if controlCount == 0 {
		varArgs = []interface{}{}
	}
	psArgs := cp.GetArgs()
	if len(psArgs) > 0 && cp.GetCtx().GetCtrlColumnRepeats() > 0 {
		for _, col := range cp.GetCtx().GetNonControlColumns() {
			// Extract response values by the key any-sdk used for the row payload, which is the
			// schema property key. That equals GetWireName when the wire name carries the case
			// difference (e.g. snake_case_aliases: display cidr_block over wire cidrBlock) and
			// GetName when the display name does (e.g. an xml:name override: display
			// AvailabilityZone over wire availabilityZone). Try the wire name first, then fall
			// back to the display name, so both cases resolve their value instead of NULL. For an
			// undecorated column GetWireName == GetName, so this is unchanged.
			va, ok := psArgs[col.GetWireName()]
			if !ok {
				va, ok = psArgs[col.GetName()]
			}
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
				if strings.ToLower(
					col.GetRelationalType(),
				) == textStr && strings.ToLower(
					dc.sqlSystem.GetName(),
				) == constants.SQLDialectPostgres {
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

func (dc *staticDRMConfig) ExecuteInsertDML(
	dbEngine sqlengine.SQLEngine,
	ctx PreparedStatementCtx,
	payload map[string]interface{},
	requestEncoding string,
) (sql.Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cannot execute on nil PreparedStatementContext")
	}
	stmtArgs, err := dc.generateVarArgs(NewPreparedStatementParameterized(
		ctx,
		payload,
		true,
	).WithRequestEncoding(requestEncoding),
		true,
		ctx.GetCtrlColumnRepeats(),
	)
	if err != nil {
		return nil, err
	}
	query := stmtArgs.GetQuery()
	return dbEngine.Exec(query, stmtArgs.GetArgs()...)
}

func (dc *staticDRMConfig) QueryDML(
	querier sqlmachinery.Querier,
	ctxParameterized PreparedStatementParameterized,
) (*sql.Rows, error) {
	prepStmt, err := dc.prepareCtx(ctxParameterized)
	if err != nil {
		return nil, err
	}
	query := prepStmt.GetRawQuery()
	varArgs := prepStmt.GetArgs()
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s, varArgs = %v", query, varArgs))
	return querier.Query(query, varArgs...)
}

func (dc *staticDRMConfig) prepareCtx(ctxParameterized PreparedStatementParameterized) (internaldto.PrepStmt, error) {
	if ctxParameterized.GetCtx() == nil {
		return nil, fmt.Errorf("cannot execute based upon nil PreparedStatementContext")
	}
	rootArgs, err := dc.generateVarArgs(ctxParameterized, false, ctxParameterized.GetCtx().GetCtrlColumnRepeats())
	if err != nil {
		return nil, err
	}
	err = rootArgs.Analyze()
	if err != nil {
		return nil, err
	}
	query := rootArgs.GetExpandedQuery()
	varArgs := rootArgs.GetExpandedArgs()
	return internaldto.NewPrepStmt(query, varArgs), nil
}

func (dc *staticDRMConfig) ExecDDL(
	querier sqlmachinery.ExecQuerier,
	ctxParameterized PreparedStatementParameterized,
) (sql.Result, error) {
	prepStmt, err := dc.prepareCtx(ctxParameterized)
	if err != nil {
		return nil, err
	}
	query := prepStmt.GetRawQuery()
	varArgs := prepStmt.GetArgs()
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	return querier.Exec(query, varArgs...)
}

func (dc *staticDRMConfig) CreateMaterializedView(
	relationName string,
	rawDDL string,
	ctxParameterized PreparedStatementParameterized,
	replaceAllowed bool,
) error {
	relationalColumns := dc.ColumnsToRelationalColumns(ctxParameterized.GetNonControlColumns())
	prepStmt, err := dc.prepareCtx(ctxParameterized)
	if err != nil {
		return err
	}
	query := prepStmt.GetRawQuery()
	varArgs := prepStmt.GetArgs()
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	return dc.sqlSystem.CreateMaterializedView(
		relationName,
		relationalColumns,
		rawDDL,
		replaceAllowed,
		query,
		varArgs...,
	)
}

func (dc *staticDRMConfig) CreatePhysicalTable(
	relationName string,
	rawDDL string,
	tableSpec *sqlparser.TableSpec,
	ifNotExists bool,
) error {
	relationalColumns := dc.translateColumns(tableSpec.Columns)
	return dc.sqlSystem.CreatePhysicalTable(
		relationName,
		relationalColumns,
		rawDDL,
		ifNotExists,
	)
}

func (dc *staticDRMConfig) RefreshMaterializedView(
	relationName string,
	ctxParameterized PreparedStatementParameterized,
) error {
	relationalColumns := dc.ColumnsToRelationalColumns(ctxParameterized.GetNonControlColumns())
	prepStmt, err := dc.prepareCtx(ctxParameterized)
	if err != nil {
		return err
	}
	query := prepStmt.GetRawQuery()
	varArgs := prepStmt.GetArgs()
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	return dc.sqlSystem.RefreshMaterializedView(
		relationName,
		relationalColumns,
		query,
		varArgs...,
	)
}

func (dc *staticDRMConfig) InsertIntoPhysicalTable(
	relationName string,
	insertColumnsString string,
	ctxParameterized PreparedStatementParameterized,
) error {
	prepStmt, err := dc.prepareCtx(ctxParameterized)
	if err != nil {
		return err
	}
	query := prepStmt.GetRawQuery()
	varArgs := prepStmt.GetArgs()
	logging.GetLogger().Infoln(fmt.Sprintf("query = %s", query))
	return dc.sqlSystem.InsertIntoPhysicalTable(
		relationName,
		insertColumnsString,
		query,
		varArgs...,
	)
}

func GenerateDRMConfig(
	sqlSystem sql_system.SQLSystem,
	typCfg typing.Config,
	namespaceCollection tablenamespace.Collection,
	controlAttributes sqlcontrol.ControlAttributes,
) (Config, error) {
	rv := &staticDRMConfig{
		namespaceCollection: namespaceCollection,
		controlAttributes:   controlAttributes,
		sqlEngine:           sqlSystem.GetSQLEngine(),
		sqlSystem:           sqlSystem,
		typCfg:              typCfg,
	}
	return rv, nil
}
