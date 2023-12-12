package astindirect

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type View struct {
	viewDTO               internaldto.RelationDTO
	selectStmt            sqlparser.SelectStatement
	selCtx                drm.PreparedStatementCtx
	paramCollection       internaldto.TableParameterCollection
	underlyingSymbolTable symtab.SymTab
}

func (v *View) GetType() IndirectType {
	return ViewType
}

func (v *View) GetRelationalColumns() []typing.RelationalColumn {
	return nil
}

func (v *View) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *View) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *View) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *View) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *View) GetName() string {
	return v.viewDTO.GetName()
}

func (v *View) GetColumns() []typing.ColumnMetadata {
	return v.selCtx.GetNonControlColumns()
}

func (v *View) GetOptionalParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *View) GetRequiredParameters() map[string]anysdk.Addressable {
	return nil
}

func (v *View) GetColumnByName(name string) (typing.ColumnMetadata, bool) {
	nccs := v.selCtx.GetNonControlColumns()
	for _, col := range nccs {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *View) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	v.selCtx = selCtx
}

func (v *View) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (v *View) GetLoadDML() (string, bool) {
	return "", false
}

func (v *View) GetRelationalColumnByIdentifier(_ string) (typing.RelationalColumn, bool) {
	return nil, false
}

func (v *View) GetSelectContext() drm.PreparedStatementCtx {
	return v.selCtx
}

func (v *View) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *View) getAST() (sqlparser.Statement, error) {
	sqlParser, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	return sqlParser.ParseQuery(v.viewDTO.GetRawQuery())
}

func (v *View) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *View) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return v.selCtx, nil
}

func (v *View) Parse() error {
	parseResult, err := v.getAST()
	if err != nil {
		return err
	}
	switch pr := parseResult.(type) {
	case sqlparser.SelectStatement:
		v.selectStmt = pr
		return nil
	default:
		return fmt.Errorf("View of type '%T' not yet supported", pr)
	}
}
