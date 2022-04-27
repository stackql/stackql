package planbuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"vitess.io/vitess/go/vt/sqlparser"
)

func analyzeFrom(from sqlparser.TableExprs, router *parserutil.ParameterRouter) ([]*taxonomy.ExtendedTableMetadata, error) {
	if len(from) > 1 {
		return nil, fmt.Errorf("cannot accomodate cartesian joins")
	}

	return nil, nil
}

// *sqlparser.ColName

func analyzeAliasedTable(handlerCtx *handler.HandlerContext, tb *sqlparser.AliasedTableExpr, router *parserutil.ParameterRouter) (*taxonomy.ExtendedTableMetadata, error) {
	switch ex := tb.Expr.(type) {
	case sqlparser.TableName:
		err := router.Route(tb)
		tpc := router.GetAvailableParameters(tb)
		if err != nil {
			return nil, err
		}
		hr, remainingParams, err := taxonomy.GetHeirarchyFromStatement(handlerCtx, tb, tpc.GetStringified())
		if err != nil {
			return nil, err
		}
		reconstitutedConsumedParams, err := tpc.ReconstituteConsumedParams(remainingParams)
		if err != nil {
			return nil, err
		}
		err = router.InvalidateParams(reconstitutedConsumedParams)
		if err != nil {
			return nil, err
		}
		m := taxonomy.NewExtendedTableMetadata(hr, taxonomy.GetAliasFromStatement(tb))
		return m, nil
	default:
		return nil, fmt.Errorf("table of type '%T' not curently supported", ex)
	}
}
