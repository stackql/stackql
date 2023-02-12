package astindirect

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/symtab"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

var (
	_ Indirect = &view{}
)

type IndirectType int

const (
	ViewType IndirectType = iota
	SubqueryType
	CTEType
)

func NewViewIndirect(viewDTO internaldto.ViewDTO) (Indirect, error) {
	rv := &view{
		viewDTO:               viewDTO,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
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
	GetColumnByName(name string) (internaldto.ColumnMetadata, bool)
	GetColumns() []internaldto.ColumnMetadata
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
}
