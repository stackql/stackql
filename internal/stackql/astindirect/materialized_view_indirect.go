package astindirect

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parser"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type materializedView struct {
	viewDTO               internaldto.RelationDTO
	selectStmt            sqlparser.SelectStatement
	paramCollection       internaldto.TableParameterCollection
	sqlSystem             sql_system.SQLSystem
	underlyingSymbolTable symtab.SymTab
}

func (v *materializedView) GetType() IndirectType {
	return MaterializedViewType
}

func (v *materializedView) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *materializedView) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *materializedView) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *materializedView) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *materializedView) GetName() string {
	return v.viewDTO.GetName()
}

func (v *materializedView) GetColumns() []typing.ColumnMetadata {
	return nil
}

func (v *materializedView) GetRelationalColumns() []typing.RelationalColumn {
	return v.viewDTO.GetColumns()
}

func (v *materializedView) GetRelationalColumnByIdentifier(name string) (typing.RelationalColumn, bool) {
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

func (v *materializedView) GetOptionalParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *materializedView) GetRequiredParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *materializedView) GetColumnByName(_ string) (typing.ColumnMetadata, bool) {
	return nil, false
}

func (v *materializedView) SetSelectContext(_ drm.PreparedStatementCtx) {
	//
}

func (v *materializedView) GetSelectContext() drm.PreparedStatementCtx {
	return nil
}

func (v *materializedView) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *materializedView) getAST() (sqlparser.Statement, error) {
	sqlParser, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	return sqlParser.ParseQuery(v.viewDTO.GetRawQuery())
}

func (v *materializedView) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *materializedView) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return nil, fmt.Errorf("materialized view does not have select context")
}

func (v *materializedView) Parse() error {
	parseResult, err := v.getAST()
	if err != nil {
		return err
	}
	switch pr := parseResult.(type) {
	case *sqlparser.DDL:
		v.selectStmt = pr.SelectStatement
		return nil
	default:
		return fmt.Errorf("materializedView of type '%T' not yet supported", pr)
	}
}

func (v *materializedView) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (v *materializedView) GetLoadDML() (string, bool) {
	return "", false
}
