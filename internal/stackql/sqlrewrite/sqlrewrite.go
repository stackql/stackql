package sqlrewrite

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type SQLRewriteInput interface { //nolint:revive //TODO: review
	GetNamespaceCollection() tablenamespace.Collection
	GetDRMConfig() drm.Config
	GetColumnDescriptors() []typing.RelationalColumn
	GetHoistedOnClauseTables() []sqlparser.SQLNode
	GetBaseControlCounters() internaldto.TxnControlCounters
	GetFromString() string
	GetIndirectContexts() []drm.PreparedStatementCtx
	GetPrepStmtOffset() int
	GetSelectQualifier() string
	GetSelectSuffix() string
	GetRewrittenWhere() string
	GetSecondaryCtrlCounters() []internaldto.TxnControlCounters
	GetTables() taxonomy.TblMap
	GetTableInsertionContainers() []tableinsertioncontainer.TableInsertionContainer
	WithIndirectContexts(indirectContexts []drm.PreparedStatementCtx) SQLRewriteInput
	WithPrepStmtOffset(offset int) SQLRewriteInput
	GetParameters() map[string]interface{}
	WithSelectQualifier(string) SQLRewriteInput
}

type standardSQLRewriteInput struct {
	dc                       drm.Config
	columnDescriptors        []typing.RelationalColumn
	baseControlCounters      internaldto.TxnControlCounters
	selectQualifier          string
	selectSuffix             string
	rewrittenWhere           string
	secondaryCtrlCounters    []internaldto.TxnControlCounters
	tables                   taxonomy.TblMap
	fromString               string
	tableInsertionContainers []tableinsertioncontainer.TableInsertionContainer
	namespaceCollection      tablenamespace.Collection
	indirectContexts         []drm.PreparedStatementCtx
	prepStmtOffset           int
	hoistedOnClauseTables    []sqlparser.SQLNode
	parameters               map[string]interface{}
}

func NewStandardSQLRewriteInput(
	dc drm.Config,
	columnDescriptors []typing.RelationalColumn,
	baseControlCounters internaldto.TxnControlCounters,
	selectSuffix string,
	rewrittenWhere string,
	secondaryCtrlCounters []internaldto.TxnControlCounters,
	tables taxonomy.TblMap,
	fromString string,
	tableInsertionContainers []tableinsertioncontainer.TableInsertionContainer,
	namespaceCollection tablenamespace.Collection,
	hoistedOnClauseTables []sqlparser.SQLNode,
	parameters map[string]interface{},
) SQLRewriteInput {
	return &standardSQLRewriteInput{
		dc:                       dc,
		columnDescriptors:        columnDescriptors,
		baseControlCounters:      baseControlCounters,
		selectSuffix:             selectSuffix,
		rewrittenWhere:           rewrittenWhere,
		secondaryCtrlCounters:    secondaryCtrlCounters,
		tables:                   tables,
		fromString:               fromString,
		tableInsertionContainers: tableInsertionContainers,
		namespaceCollection:      namespaceCollection,
		hoistedOnClauseTables:    hoistedOnClauseTables,
		parameters:               parameters,
	}
}

func (ri *standardSQLRewriteInput) GetHoistedOnClauseTables() []sqlparser.SQLNode {
	return ri.hoistedOnClauseTables
}

func (ri *standardSQLRewriteInput) WithSelectQualifier(selectQualifier string) SQLRewriteInput {
	ri.selectQualifier = selectQualifier
	return ri
}

func (ri *standardSQLRewriteInput) GetSelectQualifier() string {
	return ri.selectQualifier
}

func (ri *standardSQLRewriteInput) GetPrepStmtOffset() int {
	return ri.prepStmtOffset
}

func (ri *standardSQLRewriteInput) GetParameters() map[string]interface{} {
	return ri.parameters
}

func (ri *standardSQLRewriteInput) WithPrepStmtOffset(offset int) SQLRewriteInput {
	ri.prepStmtOffset = offset
	return ri
}

func (ri *standardSQLRewriteInput) GetDRMConfig() drm.Config {
	return ri.dc
}

func (ri *standardSQLRewriteInput) WithIndirectContexts(indirectContexts []drm.PreparedStatementCtx) SQLRewriteInput {
	ri.indirectContexts = indirectContexts
	return ri
}

func (ri *standardSQLRewriteInput) GetIndirectContexts() []drm.PreparedStatementCtx {
	return ri.indirectContexts
}

func (ri *standardSQLRewriteInput) GetNamespaceCollection() tablenamespace.Collection {
	return ri.namespaceCollection
}

func (ri *standardSQLRewriteInput) GetColumnDescriptors() []typing.RelationalColumn {
	return ri.columnDescriptors
}

func (ri *standardSQLRewriteInput) GetTableInsertionContainers() []tableinsertioncontainer.TableInsertionContainer {
	return ri.tableInsertionContainers
}

func (ri *standardSQLRewriteInput) GetBaseControlCounters() internaldto.TxnControlCounters {
	return ri.baseControlCounters
}

func (ri *standardSQLRewriteInput) GetSelectSuffix() string {
	return ri.selectSuffix
}

func (ri *standardSQLRewriteInput) GetFromString() string {
	return ri.fromString
}

func (ri *standardSQLRewriteInput) GetRewrittenWhere() string {
	return ri.rewrittenWhere
}

func (ri *standardSQLRewriteInput) GetSecondaryCtrlCounters() []internaldto.TxnControlCounters {
	return ri.secondaryCtrlCounters
}

func (ri *standardSQLRewriteInput) GetTables() taxonomy.TblMap {
	return ri.tables
}

//nolint:funlen,gocognit //TODO: review
func GenerateRewrittenSelectDML(input SQLRewriteInput) (drm.PreparedStatementCtx, error) {
	dc := input.GetDRMConfig()
	cols := input.GetColumnDescriptors()
	var txnCtrlCtrs internaldto.TxnControlCounters
	var secondaryCtrlCounters []internaldto.TxnControlCounters
	selectQualifier := input.GetSelectQualifier()
	selectSuffix := input.GetSelectSuffix()
	rewrittenWhere := input.GetRewrittenWhere()
	var columns []typing.ColumnMetadata
	var relationalColumns []typing.RelationalColumn
	var tableAliases []string
	for _, col := range cols {
		relationalColumn := col
		columns = append(
			columns,
			typing.NewRelayedColDescriptor(
				relationalColumn, relationalColumn.GetType()))
		// TODO: Need a way to handle postgres differences. This is a fragile point
		relationalColumns = append(relationalColumns, relationalColumn)
	}
	genIDColName := dc.GetControlAttributes().GetControlGenIDColumnName()
	sessionIDColName := dc.GetControlAttributes().GetControlSsnIDColumnName()
	txnIDColName := dc.GetControlAttributes().GetControlTxnIDColumnName()
	insIDColName := dc.GetControlAttributes().GetControlInsIDColumnName()
	insEncodedColName := dc.GetControlAttributes().GetControlInsertEncodedIDColumnName()
	inputContainers := input.GetTableInsertionContainers()
	hoistedOnClauseTables := input.GetHoistedOnClauseTables()
	hoistedTableAliases := make([]string, len(input.GetHoistedOnClauseTables()))
	tblMap := input.GetTables()
	if len(inputContainers) == 0 {
		txnCtrlCtrs = input.GetBaseControlCounters()
		secondaryCtrlCounters = input.GetSecondaryCtrlCounters()
	}
	i := 0
	// TODO: Only add control stuff to where clause if not
	//       already in
	//       an ON clause
	// First pass; deal with ON clause hoisted tables
	for _, tb := range inputContainers {
		v := tb.GetTableMetadata()
		isOnClauseHoistable := v.IsOnClauseHoistable()
		if !isOnClauseHoistable {
			continue
		}
		var aliasFound bool
		var foundIdx int
		for idx, node := range hoistedOnClauseTables {
			t := tblMap[node]
			if v.GetAlias() == t.GetAlias() {
				hoistedTableAliases[idx] = t.GetAlias()
				aliasFound = true
				foundIdx = idx
				break
			}
		}
		if !aliasFound {
			return nil, fmt.Errorf("could not find alias for hoisted table")
		}
		// This is required because of TOPO SORT
		if foundIdx == 0 && txnCtrlCtrs == nil {
			_, txnCtrlCtrs = tb.GetTableTxnCounters()
		}
		if foundIdx > 0 {
			_, secondaryCtr := tb.GetTableTxnCounters()
			secondaryCtrlCounters = append(secondaryCtrlCounters, secondaryCtr)
		}
		i++
	}
	aliasCache := make(map[string]struct{})
	// Second pass; deal with non-hoisted tables
	for _, tb := range inputContainers {
		v := tb.GetTableMetadata()
		isOnClauseHoistable := v.IsOnClauseHoistable()
		if isOnClauseHoistable {
			continue
		}
		// This is required because of TOPO SORT
		if txnCtrlCtrs == nil {
			_, txnCtrlCtrs = tb.GetTableTxnCounters()
		}
		// TODO: fix this hack
		//       Alias is a marker for "inside insertion group"
		alias := v.GetAlias()
		_, aliasPresent := aliasCache[alias]
		if i > 0 && !aliasPresent {
			_, secondaryCtr := tb.GetTableTxnCounters()
			secondaryCtrlCounters = append(secondaryCtrlCounters, secondaryCtr)
		}
		i++
		if aliasPresent {
			continue
		}
		aliasCache[alias] = struct{}{}
		tableAliases = append(tableAliases, alias)
	}

	// TODO add in some handle for ON clause predicates
	query, err := dc.GetSQLSystem().ComposeSelectQuery(
		relationalColumns, tableAliases, hoistedTableAliases, input.GetFromString(),
		rewrittenWhere, selectQualifier, selectSuffix, input.GetPrepStmtOffset())
	if err != nil {
		return nil, err
	}
	countedTables := 0
	for _, tbl := range input.GetTables() {
		isMaterializedView := tbl.IsMaterializedView()
		isPhysicalTable := tbl.IsPhysicalTable()
		if tbl != nil && !isMaterializedView && !isPhysicalTable {
			countedTables++
		}
	}
	rv := drm.NewPreparedStatementCtx(
		query,
		"",
		genIDColName,
		sessionIDColName,
		nil,
		txnIDColName,
		insIDColName,
		insEncodedColName,
		columns,
		countedTables,
		txnCtrlCtrs,
		secondaryCtrlCounters,
		input.GetDRMConfig().GetNamespaceCollection(),
		dc.GetSQLSystem(),
		input.GetParameters(),
	)
	rv.SetIndirectContexts(input.GetIndirectContexts())
	return rv, nil
}
