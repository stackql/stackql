package astindirect

import (
	"github.com/stackql/any-sdk/public/formulation"
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
	columns               []string // Column names extracted from SELECT
}

// NewCTEIndirect creates a new CTE indirect from a CommonTableExpr.
func NewCTEIndirect(cte *sqlparser.CommonTableExpr) (Indirect, error) {
	rv := &CTE{
		cte:                   cte,
		name:                  cte.Name.GetRawVal(),
		selectStmt:            cte.Select,
		underlyingSymbolTable: symtab.NewHashMapTreeSymTab(),
	}
	// Extract column names from the SELECT statement.
	rv.columns = rv.extractColumnsFromSelect()
	return rv, nil
}

// extractColumnsFromSelect extracts column names from the CTE's SELECT statement.
func (c *CTE) extractColumnsFromSelect() []string {
	var columns []string
	sel, ok := c.selectStmt.(*sqlparser.Select)
	if !ok {
		return columns
	}
	for _, expr := range sel.SelectExprs {
		if e, isAliased := expr.(*sqlparser.AliasedExpr); isAliased {
			// Use alias if present, otherwise try to get column name
			if e.As.GetRawVal() != "" {
				columns = append(columns, e.As.GetRawVal())
			} else if col, isCol := e.Expr.(*sqlparser.ColName); isCol {
				columns = append(columns, col.Name.GetRawVal())
			}
		}
	}
	return columns
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
	if c.selCtx != nil {
		return c.selCtx.GetNonControlColumns()
	}
	// If no selCtx, create simple column metadata from extracted column names.
	var cols []typing.ColumnMetadata
	for _, name := range c.columns {
		relCol := typing.NewRelationalColumn(name, "")
		cols = append(cols, typing.NewRelayedColDescriptor(relCol, ""))
	}
	return cols
}

func (c *CTE) GetOptionalParameters() map[string]formulation.Addressable {
	return nil
}

func (c *CTE) GetRequiredParameters() map[string]formulation.Addressable {
	return nil
}

func (c *CTE) GetColumnByName(name string) (typing.ColumnMetadata, bool) {
	if c.selCtx != nil {
		for _, col := range c.selCtx.GetNonControlColumns() {
			if col.GetIdentifier() == name {
				return col, true
			}
		}
		return nil, false
	}
	// If no selCtx, check extracted column names.
	for _, colName := range c.columns {
		if colName == name {
			relCol := typing.NewRelationalColumn(name, "")
			return typing.NewRelayedColDescriptor(relCol, ""), true
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
