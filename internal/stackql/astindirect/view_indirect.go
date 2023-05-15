package astindirect

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql/internal/stackql/symtab"
)

type view struct {
	viewDTO               internaldto.ViewDTO
	selectStmt            sqlparser.SelectStatement
	selCtx                drm.PreparedStatementCtx
	paramCollection       internaldto.TableParameterCollection
	underlyingSymbolTable symtab.SymTab
}

func (v *view) GetType() IndirectType {
	return ViewType
}

func (v *view) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *view) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *view) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *view) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *view) GetName() string {
	return v.viewDTO.GetName()
}

func (v *view) GetColumns() []internaldto.ColumnMetadata {
	return v.selCtx.GetNonControlColumns()
}

func (v *view) GetOptionalParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *view) GetRequiredParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *view) GetColumnByName(name string) (internaldto.ColumnMetadata, bool) {
	nccs := v.selCtx.GetNonControlColumns()
	for _, col := range nccs {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *view) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	v.selCtx = selCtx
}

func (v *view) GetSelectContext() drm.PreparedStatementCtx {
	return v.selCtx
}

func (v *view) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *view) getAST() (sqlparser.Statement, error) {
	sqlParser, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	return sqlParser.ParseQuery(v.viewDTO.GetRawQuery())
}

func (v *view) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *view) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return v.selCtx, nil
}

func (v *view) Parse() error {
	parseResult, err := v.getAST()
	if err != nil {
		return err
	}
	switch pr := parseResult.(type) {
	case sqlparser.SelectStatement:
		v.selectStmt = pr
		return nil
	default:
		return fmt.Errorf("view of type '%T' not yet supported", pr)
	}
}
