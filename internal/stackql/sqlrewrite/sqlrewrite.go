package sqlrewrite

import (
	"fmt"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"vitess.io/vitess/go/vt/sqlparser"
)

type SQLRewriteInput interface {
	GetDRMConfig() drm.DRMConfig
	GetColumnDescriptors() []openapistackql.ColumnDescriptor
	GetBaseControlCounters() *dto.TxnControlCounters
	GetFromString() string
	GetSelectSuffix() string
	GetRewrittenWhere() string
	GetSecondaryCtrlCounters() []*dto.TxnControlCounters
	GetTables() taxonomy.TblMap
	GetTableSlice() []*taxonomy.ExtendedTableMetadata
}

type StandardSQLRewriteInput struct {
	dc                    drm.DRMConfig
	columnDescriptors     []openapistackql.ColumnDescriptor
	baseControlCounters   *dto.TxnControlCounters
	selectSuffix          string
	rewrittenWhere        string
	secondaryCtrlCounters []*dto.TxnControlCounters
	tables                taxonomy.TblMap
	fromString            string
	tableSlice            []*taxonomy.ExtendedTableMetadata
}

func NewStandardSQLRewriteInput(
	dc drm.DRMConfig,
	columnDescriptors []openapistackql.ColumnDescriptor,
	baseControlCounters *dto.TxnControlCounters,
	selectSuffix string,
	rewrittenWhere string,
	secondaryCtrlCounters []*dto.TxnControlCounters,
	tables taxonomy.TblMap,
	fromString string,
	tableSlice []*taxonomy.ExtendedTableMetadata,
) SQLRewriteInput {
	return &StandardSQLRewriteInput{
		dc:                    dc,
		columnDescriptors:     columnDescriptors,
		baseControlCounters:   baseControlCounters,
		selectSuffix:          selectSuffix,
		rewrittenWhere:        rewrittenWhere,
		secondaryCtrlCounters: secondaryCtrlCounters,
		tables:                tables,
		fromString:            fromString,
		tableSlice:            tableSlice,
	}
}

func (ri *StandardSQLRewriteInput) GetDRMConfig() drm.DRMConfig {
	return ri.dc
}

func (ri *StandardSQLRewriteInput) GetColumnDescriptors() []openapistackql.ColumnDescriptor {
	return ri.columnDescriptors
}

func (ri *StandardSQLRewriteInput) GetTableSlice() []*taxonomy.ExtendedTableMetadata {
	return ri.tableSlice
}

func (ri *StandardSQLRewriteInput) GetBaseControlCounters() *dto.TxnControlCounters {
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

func (ri *StandardSQLRewriteInput) GetSecondaryCtrlCounters() []*dto.TxnControlCounters {
	return ri.secondaryCtrlCounters
}

func (ri *StandardSQLRewriteInput) GetTables() taxonomy.TblMap {
	return ri.tables
}

func GenerateSelectDML(input SQLRewriteInput) (*drm.PreparedStatementCtx, error) {
	dc := input.GetDRMConfig()
	cols := input.GetColumnDescriptors()
	txnCtrlCtrs := input.GetBaseControlCounters()
	selectSuffix := input.GetSelectSuffix()
	rewrittenWhere := input.GetRewrittenWhere()
	var q strings.Builder
	var quotedColNames []string
	var columns []drm.ColumnMetadata
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
		columns = append(columns, drm.NewColDescriptor(col, typeStr))
		var colEntry strings.Builder
		if col.DecoratedCol == "" {
			colEntry.WriteString(fmt.Sprintf(`"%s" `, col.Name))
			if col.Alias != "" {
				colEntry.WriteString(fmt.Sprintf(` AS "%s"`, col.Alias))
			}
		} else {
			colEntry.WriteString(fmt.Sprintf("%s ", col.DecoratedCol))
		}
		quotedColNames = append(quotedColNames, fmt.Sprintf("%s ", colEntry.String()))
	}
	genIdColName := dc.GetGenerationControlColumn()
	sessionIDColName := dc.GetSessionControlColumn()
	txnIdColName := dc.GetTxnControlColumn()
	insIdColName := dc.GetInsControlColumn()
	var wq strings.Builder
	var controlWhereComparisons []string
	for _, v := range input.GetTableSlice() {
		alias := v.Alias
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
	}
	controlWhereSubClause := fmt.Sprintf("( %s )", strings.Join(controlWhereComparisons, " AND "))
	wq.WriteString(controlWhereSubClause)
	if strings.TrimSpace(rewrittenWhere) != "" {
		wq.WriteString(fmt.Sprintf(" AND ( %s ) ", rewrittenWhere))
	}
	whereExprsStr := wq.String()

	q.WriteString(fmt.Sprintf(`SELECT %s FROM `, strings.Join(quotedColNames, ", ")))
	q.WriteString(input.GetFromString())
	if whereExprsStr != "" {
		q.WriteString(" WHERE ")
		q.WriteString(whereExprsStr)
	}
	q.WriteString(selectSuffix)

	query := q.String()
	return drm.NewPreparedStatementCtx(
		query,
		"",
		genIdColName,
		sessionIDColName,
		nil,
		txnIdColName,
		insIdColName,
		columns,
		len(input.GetTables()),
		txnCtrlCtrs,
		input.GetSecondaryCtrlCounters(),
	), nil
}
