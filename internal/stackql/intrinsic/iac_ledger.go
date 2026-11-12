package intrinsic

import (
	"context"
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

// A capability check permits a clear upgrade error with an older published SDK.
type ledgerReader interface {
	Read(context.Context) ([]omnisdk.Row, error)
}

type ledgerReadPlan struct {
	reader   ledgerReader
	resource string
}

func (p ledgerReadPlan) Open(ctx context.Context) (omnisdk.Rows, error) {
	rows, err := p.reader.Read(ctx)
	if err != nil {
		return nil, err
	}
	if p.resource != "" {
		selected := make([]omnisdk.Row, 0, 1)
		for _, row := range rows {
			if row["resource_key"] == p.resource {
				selected = append(selected, row)
			}
		}
		rows = selected
	}
	return &materializedRelationRows{ctx: ctx, rows: rows}, nil
}

func ledgerSelectFunc(ctx queryContext, node *sqlparser.Select) func() internaldto.ExecutorOutput {
	params, config, refusal := ledgerPredicates(ctx, node, relationAddress(iacService, ledgerRelation))
	if refusal != nil {
		return refusal
	}
	resource := popPredicate(params, "resource_key")
	if len(params) != 0 {
		return refuse(fmt.Errorf("ledger accepts collection, state and optional resource_key predicates"))
	}
	reader, ok := config.(ledgerReader)
	if !ok {
		return refuse(fmt.Errorf("the Omni dependency needs the read-only ledger capability; upgrade Omni"))
	}
	return func() internaldto.ExecutorOutput {
		return streamPlan(ctx, ledgerReadPlan{reader: reader, resource: resource}, ledgerRelation, node.SelectExprs)
	}
}
