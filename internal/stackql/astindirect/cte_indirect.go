package astindirect

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/typing"
)

// CTE represents a Common Table Expression indirect reference.
type CTE struct {
	cte                   *sqlparser.CommonTableExpr
	name                  string
	selectStmt            sqlparser.SelectStatement
	selCtx                drm.PreparedStatementCtx
	paramCollection       internaldto.TableParameterCollection
	underlyingSymbolTable symtab.SymTab
	next                  Indirect
	isElide               bool
}

// NewCTEIndirect creates a new CTE indirect from a CommonTableExpr.
func NewCTEIndirect(cte *sqlparser.CommonTableExpr) (Indirect, error) {
	rv := &CTE{
		cte:                   cte,
		name:                  cte.Name.GetRawVal(),
		selectStmt:            cte.Select,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	return rv, nil
}

func (c *CTE) IsElide() bool {
	return c.isElide
}

func (c *CTE) SetElide(isElide bool) {
	c.isElide = isElide
}

func (c *CTE) GetType() IndirectType {
	return CTEType
}

func (c *CTE) MatchOnParams(_ map[string]any) (Indirect, bool) {
	return nil, false
}

func (c *CTE) WithNext(next Indirect) Indirect {
	c.next = next
	return c.next
}

func (c *CTE) Next() (Indirect, bool) {
	return c.next, c.next != nil
}

func (c *CTE) GetRelationalColumns() []typing.RelationalColumn {
	return nil
}

func (c *CTE) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return c.paramCollection, c.paramCollection != nil
}

func (c *CTE) SetAssignedParameters(paramCollection internaldto.TableParameterCollection) {
	c.paramCollection = paramCollection
}

func (c *CTE) GetUnderlyingSymTab() symtab.SymTab {
	return c.underlyingSymbolTable
}

func (c *CTE) SetUnderlyingSymTab(symbolTable symtab.SymTab) {
	c.underlyingSymbolTable = symbolTable
}

func (c *CTE) GetName() string {
	return c.name
}

func (c *CTE) GetColumns() []typing.ColumnMetadata {
	if c.selCtx == nil {
		return nil
	}
	return c.selCtx.GetNonControlColumns()
}

func (c *CTE) GetOptionalParameters() map[string]anysdk.Addressable {
	return nil
}

func (c *CTE) GetRequiredParameters() map[string]anysdk.Addressable {
	return nil
}

func (c *CTE) GetColumnByName(name string) (typing.ColumnMetadata, bool) {
	if c.selCtx == nil {
		return nil, false
	}
	for _, col := range c.selCtx.GetNonControlColumns() {
		if col.GetIdentifier() == name {
			return col, true
		}
	}
	return nil, false
}

func (c *CTE) SetSelectContext(selCtx drm.PreparedStatementCtx) {
	c.selCtx = selCtx
}

func (c *CTE) GetSelectContext() drm.PreparedStatementCtx {
	return c.selCtx
}

func (c *CTE) GetTables() sqlparser.TableExprs {
	return nil
}

func (c *CTE) GetSelectAST() sqlparser.SelectStatement {
	return c.selectStmt
}

func (c *CTE) GetSelectionCtx() (drm.PreparedStatementCtx, error) {
	return c.selCtx, nil
}

func (c *CTE) Parse() error {
	// CTE select statement is already parsed.
	return nil
}

func (c *CTE) GetTranslatedDDL() (string, bool) {
	return "", false
}

func (c *CTE) GetLoadDML() (string, bool) {
	return "", false
}

func (c *CTE) GetRelationalColumnByIdentifier(_ string) (typing.RelationalColumn, bool) {
	return nil, false
}
