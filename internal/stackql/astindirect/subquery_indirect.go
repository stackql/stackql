package astindirect

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type Subquery struct {
	subQueryDTO           internaldto.SubqueryDTO
	subQuery              *sqlparser.Subquery
	selectStmt            sqlparser.SelectStatement
	selCtx                drm.PreparedStatementCtx
	paramCollection       internaldto.TableParameterCollection
	underlyingSymbolTable symtab.SymTab
}

func (v *Subquery) GetType() IndirectType {
	return SubqueryType
}

func (v *Subquery) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *Subquery) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *Subquery) GetRelationalColumns() []typing.RelationalColumn {
	return nil
}

func (v *Subquery) GetRelationalColumnByIdentifier(_ string) (typing.RelationalColumn, bool) {
	return nil, false
}

func (v *Subquery) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *Subquery) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *Subquery) GetName() string {
	return v.subQueryDTO.GetAlias().GetRawVal()
}

func (v *Subquery) GetCtrlColumnRepeats() int {
	return v.selCtx.GetCtrlColumnRepeats()
}

func (v *Subquery) GetColumns() []typing.ColumnMetadata {
	return v.selCtx.GetNonControlColumns()
}

func (v *Subquery) GetOptionalParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *Subquery) GetRequiredParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *Subquery) GetColumnByName(name string) (typing.ColumnMetadata, bool) {
	for _, col := range v.selCtx.GetNonControlColumns() {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *Subquery) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	v.selCtx = selCtx
}

func (v *Subquery) GetSelectContext() drm.PreparedStatementCtx {
	return v.selCtx
}

func (v *Subquery) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *Subquery) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *Subquery) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return v.selCtx, nil
}

func (v *Subquery) Parse() error {
	return nil
}

func (v *Subquery) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (v *Subquery) GetLoadDML() (string, bool) {
	return "", false
}
