package internaldto

import "vitess.io/vitess/go/vt/sqlparser"

var (
	_ SubqueryDTO = &standardSubqueryDTO{}
)

func NewSubqueryDTO(aliasedTableExpr *sqlparser.AliasedTableExpr, subQuery *sqlparser.Subquery) SubqueryDTO {
	return &standardSubqueryDTO{
		aliasedTableExpr: aliasedTableExpr,
		subQuery:         subQuery,
	}
}

type SubqueryDTO interface {
	GetSubquery() *sqlparser.Subquery
	GetAlias() sqlparser.TableIdent
	GetAliasedTableExpr() *sqlparser.AliasedTableExpr
}

type standardSubqueryDTO struct {
	subQuery         *sqlparser.Subquery
	aliasedTableExpr *sqlparser.AliasedTableExpr
}

func (v *standardSubqueryDTO) GetSubquery() *sqlparser.Subquery {
	return v.subQuery
}

func (v *standardSubqueryDTO) GetAlias() sqlparser.TableIdent {
	return v.aliasedTableExpr.As
}

func (v *standardSubqueryDTO) GetAliasedTableExpr() *sqlparser.AliasedTableExpr {
	return v.aliasedTableExpr
}
