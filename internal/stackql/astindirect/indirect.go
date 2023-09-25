package astindirect

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

var (
	_ Indirect = &view{}
	_ Indirect = &subquery{}
	_ Indirect = &materializedView{}
)

type IndirectType int

const (
	ViewType IndirectType = iota
	SubqueryType
	CTEType
	MaterializedViewType
	PhysicalTableType
	SelectType
	ExecType
	InsertRowsType
)

func NewViewIndirect(viewDTO internaldto.RelationDTO) (Indirect, error) {
	rv := &view{
		viewDTO:               viewDTO,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func NewMaterializedViewIndirect(viewDTO internaldto.RelationDTO, sqlSystem sql_system.SQLSystem) (Indirect, error) {
	rv := &materializedView{
		viewDTO:               viewDTO,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
		sqlSystem:             sqlSystem,
	}
	return rv, nil
}

func NewParserSelectIndirect(selectObj *sqlparser.Select, selCtx drm.PreparedStatementCtx) (Indirect, error) {
	rv := &parserSelectionIndirect{
		selCtx:                selCtx,
		selectObj:             selectObj,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func NewInsertRowsIndirect(insertObj *sqlparser.Insert, selCtx drm.PreparedStatementCtx) (Indirect, error) {
	rv := &parserSelectionIndirect{
		selCtx:                selCtx,
		insertObj:             insertObj,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func NewParserExecIndirect(execObj *sqlparser.Exec, selCtx drm.PreparedStatementCtx) (Indirect, error) {
	rv := &parserSelectionIndirect{
		selCtx:                selCtx,
		execObj:               execObj,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func NewPhysicalTableIndirect(tableDTO internaldto.RelationDTO, sqlSystem sql_system.SQLSystem) (Indirect, error) {
	rv := &physicalTable{
		tableDTO:              tableDTO,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
		sqlSystem:             sqlSystem,
	}
	return rv, nil
}

func NewSubqueryIndirect(subQueryDTO internaldto.SubqueryDTO) (Indirect, error) {
	if subQueryDTO == nil {
		return nil, fmt.Errorf("cannot accomodate nil subquery")
	}
	rv := &subquery{
		subQueryDTO:           subQueryDTO,
		subQuery:              subQueryDTO.GetSubquery(),
		selectStmt:            subQueryDTO.GetSubquery().Select,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

type Indirect interface {
	Parse() error
	GetAssignedParameters() (internaldto.TableParameterCollection, bool)
	GetColumnByName(name string) (typing.ColumnMetadata, bool)
	GetRelationalColumnByIdentifier(name string) (typing.RelationalColumn, bool)
	GetColumns() []typing.ColumnMetadata
	GetRelationalColumns() []typing.RelationalColumn
	GetName() string
	GetOptionalParameters() map[string]openapistackql.Addressable
	GetRequiredParameters() map[string]openapistackql.Addressable
	GetSelectAST() sqlparser.SelectStatement
	GetSelectContext() drm.PreparedStatementCtx
	GetType() IndirectType
	GetUnderlyingSymTab() symtab.SymTab
	SetAssignedParameters(internaldto.TableParameterCollection)
	SetSelectContext(drm.PreparedStatementCtx)
	SetUnderlyingSymTab(symtab.SymTab)
	GetTranslatedDDL() (string, bool)
	GetLoadDML() (string, bool)
}
