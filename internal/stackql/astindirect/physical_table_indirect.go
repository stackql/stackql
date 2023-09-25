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

type physicalTable struct {
	tableDTO              internaldto.RelationDTO
	selectStmt            sqlparser.SelectStatement
	paramCollection       internaldto.TableParameterCollection
	sqlSystem             sql_system.SQLSystem
	underlyingSymbolTable symtab.SymTab
}

func (v *physicalTable) GetType() IndirectType {
	return PhysicalTableType
}

func (v *physicalTable) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return v.paramCollection, v.paramCollection != nil
}

func (v *physicalTable) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	v.paramCollection = paramCollection
}

func (v *physicalTable) GetUnderlyingSymTab() symtab.SymTab {
	return v.underlyingSymbolTable
}

func (v *physicalTable) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	v.underlyingSymbolTable = symbolTable
}

func (v *physicalTable) GetName() string {
	return v.tableDTO.GetName()
}

func (v *physicalTable) GetColumns() []typing.ColumnMetadata {
	return nil
}

func (v *physicalTable) GetRelationalColumns() []typing.RelationalColumn {
	return v.tableDTO.GetColumns()
}

func (v *physicalTable) GetRelationalColumnByIdentifier(name string) (typing.RelationalColumn, bool) {
	for _, col := range v.tableDTO.GetColumns() {
		if col.GetName() == name {
			return col, true
		}
		if col.GetAlias() == name {
			return col, true
		}
	}
	return nil, false
}

func (v *physicalTable) GetOptionalParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *physicalTable) GetRequiredParameters() map[string]openapistackql.Addressable {
	return nil
}

func (v *physicalTable) GetColumnByName(_ string) (typing.ColumnMetadata, bool) {
	return nil, false
}

func (v *physicalTable) SetSelectContext(_ drm.PreparedStatementCtx) {
	//
}

func (v *physicalTable) GetSelectContext() drm.PreparedStatementCtx {
	return nil
}

func (v *physicalTable) GetTables() sqlparser.TableExprs {
	return nil
}

func (v *physicalTable) getAST() (sqlparser.Statement, error) {
	sqlParser, err := parser.NewParser()
	if err != nil {
		return nil, err
	}
	return sqlParser.ParseQuery(v.tableDTO.GetRawQuery())
}

func (v *physicalTable) GetSelectAST() sqlparser.SelectStatement {
	return v.selectStmt
}

func (v *physicalTable) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return nil, fmt.Errorf("physical table does not have select context")
}

func (v *physicalTable) Parse() error {
	parseResult, err := v.getAST()
	if err != nil {
		return err
	}
	switch pr := parseResult.(type) {
	case *sqlparser.DDL:
		v.selectStmt = pr.SelectStatement
		return nil
	default:
		return fmt.Errorf("physical table of type '%T' not yet supported", pr)
	}
}

func (v *physicalTable) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (v *physicalTable) GetLoadDML() (string, bool) {
	return "", false
}
