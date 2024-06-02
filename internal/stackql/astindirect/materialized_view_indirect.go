package astindirect

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type MaterializedView struct {
	viewDTO               internaldto.RelationDTO
	selectStmt            sqlparser.SelectStatement
	paramCollection       internaldto.TableParameterCollection
	sqlSystem             sql_system.SQLSystem
	underlyingSymbolTable symtab.SymTab
}

func (v *MaterializedView) IsElide() bool {
	return false
}

func (v *MaterializedView) SetElide(bool) {
	//
}

func (v *MaterializedView) MatchOnParams(_ map[string]any) (Indirect, bool) {
	return nil, false
}

func (v *MaterializedView) WithNext(Indirect) Indirect {
	return v
}

func (v *MaterializedView) Next() (Indirect, bool) {
	return nil, false
}

func (v *MaterializedView) GetType() IndirectType {
	return MaterializedViewType
}

func (v *MaterializedView) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *MaterializedView) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *MaterializedView) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *MaterializedView) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *MaterializedView) GetName() string {
	return v.viewDTO.GetName()
}

func (v *MaterializedView) GetColumns() []typing.ColumnMetadata {
	return nil
}

func (v *MaterializedView) GetRelationalColumns() []typing.RelationalColumn {
	return v.viewDTO.GetColumns()
}

func (v *MaterializedView) GetRelationalColumnByIdentifier(name string) (typing.RelationalColumn, bool) {
	for _, col := range v.viewDTO.GetColumns() {
		if col.GetName() == name {
			return col, true
		}
		if col.GetAlias() != "" && col.GetAlias() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *MaterializedView) GetOptionalParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *MaterializedView) GetRequiredParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *MaterializedView) GetColumnByName(_ string) (typing.ColumnMetadata, bool) {
	return nil, false
}

func (v *MaterializedView) SetSelectContext(_ drm.PreparedStatementCtx) {
	//
}

func (v *MaterializedView) GetSelectContext() drm.PreparedStatementCtx {
	return nil
}

func (v *MaterializedView) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *MaterializedView) getAST() (sqlparser.Statement, error) {
	sqlParser, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	return sqlParser.ParseQuery(v.viewDTO.GetRawQuery())
}

func (v *MaterializedView) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *MaterializedView) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return nil, fmt.Errorf("materialized view does not have select context")
}

func (v *MaterializedView) Parse() error {
	parseResult, err := v.getAST()
	if err != nil {
		return err
	}
	switch pr := parseResult.(type) {
	case *sqlparser.DDL:
		v.selectStmt = pr.SelectStatement
	default:
		return fmt.Errorf("MaterializedView of type '%T' not yet supported", pr)
	}
	for _, col := range v.viewDTO.GetColumns() {
		colID := col.GetIdentifier()
		colType := col.GetType()
		colEntry := symtab.NewSymTabEntry(
			colType,
			"",
			"",
		)
		err = v.underlyingSymbolTable.SetSymbol(colID, colEntry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *MaterializedView) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (v *MaterializedView) GetLoadDML() (string, bool) {
	return "", false
}
