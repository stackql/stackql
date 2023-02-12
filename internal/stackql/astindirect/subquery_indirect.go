package astindirect

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/symtab"
)

type subquery struct {
	subQueryDTO           internaldto.SubqueryDTO
	subQuery              *sqlparser.Subquery
	selectStmt            sqlparser.SelectStatement
	selCtx                drm.PreparedStatementCtx
	paramCollection       internaldto.TableParameterCollection
	underlyingSymbolTable symtab.SymTab
}

func (v *subquery) GetType() IndirectType {
	return SubqueryType
}

func (v *subquery) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *subquery) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *subquery) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *subquery) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *subquery) GetName() string {
	return v.subQueryDTO.GetAlias().GetRawVal()
}

func (v *subquery) GetColumns() []internaldto.ColumnMetadata {
	return v.selCtx.GetNonControlColumns()
}

func (v *subquery) GetOptionalParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *subquery) GetRequiredParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *subquery) GetColumnByName(name string) (internaldto.ColumnMetadata, bool) {
	for _, col := range v.selCtx.GetNonControlColumns() {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *subquery) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	v.selCtx = selCtx
}

func (v *subquery) GetSelectContext() drm.PreparedStatementCtx {
	return v.selCtx
}

func (v *subquery) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *subquery) getAST() (sqlparser.Statement, error) {
	return v.subQuery.Select, nil
}

func (v *subquery) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *subquery) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return v.selCtx, nil
}

func (v *subquery) Parse() error {
	return nil
}
