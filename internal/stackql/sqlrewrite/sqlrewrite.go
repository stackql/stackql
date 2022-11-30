package sqlrewrite

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/relationaldto"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"vitess.io/vitess/go/vt/sqlparser"
)

type SQLRewriteInput interface {
	GetNamespaceCollection() tablenamespace.TableNamespaceCollection
	GetDRMConfig() drm.DRMConfig
	GetColumnDescriptors() []openapistackql.ColumnDescriptor
	GetBaseControlCounters() dto.TxnControlCounters
	GetFromString() string
	GetSelectSuffix() string
	GetRewrittenWhere() string
	GetSecondaryCtrlCounters() []dto.TxnControlCounters
	GetTables() taxonomy.TblMap
	GetTableInsertionContainers() []tableinsertioncontainer.TableInsertionContainer
}

type StandardSQLRewriteInput struct {
	dc                       drm.DRMConfig
	columnDescriptors        []openapistackql.ColumnDescriptor
	baseControlCounters      dto.TxnControlCounters
	selectSuffix             string
	rewrittenWhere           string
	secondaryCtrlCounters    []dto.TxnControlCounters
	tables                   taxonomy.TblMap
	fromString               string
	tableInsertionContainers []tableinsertioncontainer.TableInsertionContainer
	namespaceCollection      tablenamespace.TableNamespaceCollection
}

func NewStandardSQLRewriteInput(
	dc drm.DRMConfig,
	columnDescriptors []openapistackql.ColumnDescriptor,
	baseControlCounters dto.TxnControlCounters,
	selectSuffix string,
	rewrittenWhere string,
	secondaryCtrlCounters []dto.TxnControlCounters,
	tables taxonomy.TblMap,
	fromString string,
	tableInsertionContainers []tableinsertioncontainer.TableInsertionContainer,
	namespaceCollection tablenamespace.TableNamespaceCollection,
) SQLRewriteInput {
	return &StandardSQLRewriteInput{
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
	}
}

func (ri *StandardSQLRewriteInput) GetDRMConfig() drm.DRMConfig {
	return ri.dc
}

func (ri *StandardSQLRewriteInput) GetNamespaceCollection() tablenamespace.TableNamespaceCollection {
	return ri.namespaceCollection
}

func (ri *StandardSQLRewriteInput) GetColumnDescriptors() []openapistackql.ColumnDescriptor {
	return ri.columnDescriptors
}

func (ri *StandardSQLRewriteInput) GetTableInsertionContainers() []tableinsertioncontainer.TableInsertionContainer {
	return ri.tableInsertionContainers
}

func (ri *StandardSQLRewriteInput) GetBaseControlCounters() dto.TxnControlCounters {
	return ri.baseControlCounters
}

func (ri *StandardSQLRewriteInput) GetSelectSuffix() string {
	return ri.selectSuffix
}

func (ri *StandardSQLRewriteInput) GetFromString() string {
	return ri.fromString
}

func (ri *StandardSQLRewriteInput) GetRewrittenWhere() string {
	return ri.rewrittenWhere
}

func (ri *StandardSQLRewriteInput) GetSecondaryCtrlCounters() []dto.TxnControlCounters {
	return ri.secondaryCtrlCounters
}

func (ri *StandardSQLRewriteInput) GetTables() taxonomy.TblMap {
	return ri.tables
}

func GenerateSelectDML(input SQLRewriteInput) (drm.PreparedStatementCtx, error) {
	dc := input.GetDRMConfig()
	cols := input.GetColumnDescriptors()
	var txnCtrlCtrs dto.TxnControlCounters
	var secondaryCtrlCounters []dto.TxnControlCounters
	selectSuffix := input.GetSelectSuffix()
	rewrittenWhere := input.GetRewrittenWhere()
	var columns []drm.ColumnMetadata
	var relationalColumns []relationaldto.RelationalColumn
	var tableAliases []string
	for _, col := range cols {
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
		relationalColumn := relationaldto.NewRelationalColumn(col.Name, typeStr).WithQualifier(col.Qualifier).WithAlias(col.Alias).WithDecorated(col.DecoratedCol).WithParserNode(col.Node)
		columns = append(columns, drm.NewColDescriptor(col, typeStr))
		// TODO: Need a way to handle postgres differences. This is a fragile point
		relationalColumns = append(relationalColumns, relationalColumn)
	}
	genIdColName := dc.GetControlAttributes().GetControlGenIdColumnName()
	sessionIDColName := dc.GetControlAttributes().GetControlSsnIdColumnName()
	txnIdColName := dc.GetControlAttributes().GetControlTxnIdColumnName()
	insIdColName := dc.GetControlAttributes().GetControlInsIdColumnName()
	insEncodedColName := dc.GetControlAttributes().GetControlInsertEncodedIdColumnName()
	inputContainers := input.GetTableInsertionContainers()
	if len(inputContainers) > 0 {
		_, txnCtrlCtrs = inputContainers[0].GetTableTxnCounters()
	} else {
		txnCtrlCtrs = input.GetBaseControlCounters()
		secondaryCtrlCounters = input.GetSecondaryCtrlCounters()
	}
	i := 0
	for _, tb := range inputContainers {
		if i > 0 {
			_, secondaryCtr := tb.GetTableTxnCounters()
			secondaryCtrlCounters = append(secondaryCtrlCounters, secondaryCtr)
		}
		v := tb.GetTableMetadata()
		alias := v.GetAlias()
		tableAliases = append(tableAliases, alias)
		i++
	}

	query, err := dc.GetSQLDialect().ComposeSelectQuery(relationalColumns, tableAliases, input.GetFromString(), rewrittenWhere, selectSuffix)
	if err != nil {
		return nil, err
	}
	return drm.NewPreparedStatementCtx(
		query,
		"",
		genIdColName,
		sessionIDColName,
		nil,
		txnIdColName,
		insIdColName,
		insEncodedColName,
		columns,
		len(input.GetTables()),
		txnCtrlCtrs,
		secondaryCtrlCounters,
		input.GetDRMConfig().GetNamespaceCollection(),
		dc.GetSQLDialect(),
	), nil
}
