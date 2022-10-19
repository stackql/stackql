package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"vitess.io/vitess/go/vt/sqlparser"
)

type TblMap map[sqlparser.SQLNode]*tablemetadata.ExtendedTableMetadata

func (tm TblMap) GetTable(node sqlparser.SQLNode) (*tablemetadata.ExtendedTableMetadata, error) {
	tbl, ok := tm[node]
	if !ok {
		return nil, fmt.Errorf("could not locate table for AST node: %v", node)
	}
	return tbl, nil
}

func (tm TblMap) getUniqueCount() int {
	found := make(map[*tablemetadata.ExtendedTableMetadata]struct{})
	for _, v := range tm {
		if _, ok := found[v]; !ok {
			found[v] = struct{}{}
		}
	}
	return len(found)
}

func (tm TblMap) getFirst() (*tablemetadata.ExtendedTableMetadata, bool) {
	for _, v := range tm {
		return v, true
	}
	return nil, false
}

func (tm TblMap) GetTableLoose(node sqlparser.SQLNode) (*tablemetadata.ExtendedTableMetadata, error) {
	tbl, ok := tm[node]
	if ok {
		return tbl, nil
	}
	searchAlias := ""
	switch node := node.(type) {
	case *sqlparser.AliasedExpr:
		switch expr := node.Expr.(type) {
		case *sqlparser.ColName:
			searchAlias = expr.Qualifier.GetRawVal()
		}
	}
	if searchAlias != "" {
		for k, v := range tm {
			switch k := k.(type) {
			case *sqlparser.AliasedTableExpr:
				alias := k.As.GetRawVal()
				if searchAlias == alias {
					return v, nil
				}
			}
		}
	}
	if searchAlias == "" && tm.getUniqueCount() == 1 {
		if first, ok := tm.getFirst(); ok {
			return first, nil
		}
	}
	return nil, fmt.Errorf("could not locate table for AST node: %v", node)
}

func (tm TblMap) SetTable(node sqlparser.SQLNode, table *tablemetadata.ExtendedTableMetadata) {
	tm[node] = table
}
