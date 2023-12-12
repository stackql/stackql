package astindirect

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type parserSelectionIndirect struct {
	selectObj             *sqlparser.Select
	execObj               *sqlparser.Exec
	insertObj             *sqlparser.Insert
	selCtx                drm.PreparedStatementCtx
	paramCollection       internaldto.TableParameterCollection
	underlyingSymbolTable symtab.SymTab
}

func (v *parserSelectionIndirect) GetType() IndirectType {
	isExec := v.execObj != nil
	if isExec {
		return ExecType
	}
	return SelectType
}

func (v *parserSelectionIndirect) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *parserSelectionIndirect) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *parserSelectionIndirect) GetRelationalColumns() []typing.RelationalColumn {
	return nil
}

func (v *parserSelectionIndirect) GetRelationalColumnByIdentifier(_ string) (typing.RelationalColumn, bool) {
	return nil, false
}

func (v *parserSelectionIndirect) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *parserSelectionIndirect) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *parserSelectionIndirect) GetName() string {
	return ""
}

func (v *parserSelectionIndirect) GetColumns() []typing.ColumnMetadata {
	return v.selCtx.GetNonControlColumns()
}

func (v *parserSelectionIndirect) GetOptionalParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *parserSelectionIndirect) GetRequiredParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *parserSelectionIndirect) GetColumnByName(name string) (typing.ColumnMetadata, bool) {
	for _, col := range v.selCtx.GetNonControlColumns() {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *parserSelectionIndirect) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	v.selCtx = selCtx
}

func (v *parserSelectionIndirect) GetSelectContext() drm.PreparedStatementCtx {
	return v.selCtx
}

func (v *parserSelectionIndirect) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *parserSelectionIndirect) GetSelectAST() sqlparser.SelectStatement {
	return v.selectObj
}

func (v *parserSelectionIndirect) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return v.selCtx, nil
}

func (v *parserSelectionIndirect) Parse() error {
	return nil
}

func (v *parserSelectionIndirect) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (v *parserSelectionIndirect) GetLoadDML() (string, bool) {
	return "", false
}
