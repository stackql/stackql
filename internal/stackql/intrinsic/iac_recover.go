package intrinsic

import (
	"context"
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type ledgerRecoverer interface {
	Recover(context.Context, string, string, string, string, omnisdk.Args) ([]omnisdk.Row, error)
}

type ledgerRecoveryPlan struct {
	recoverer                            ledgerRecoverer
	resource, outcome, identity, version string
	args                                 omnisdk.Args
}

func (p ledgerRecoveryPlan) Open(ctx context.Context) (omnisdk.Rows, error) {
	rows, err := p.recoverer.Recover(ctx, p.resource, p.outcome, p.identity, p.version, p.args)
	if err != nil {
		return nil, err
	}
	return &materializedRelationRows{ctx: ctx, rows: rows}, nil
}

func recoverSelectFunc(ctx queryContext, node *sqlparser.Select) func() internaldto.ExecutorOutput {
	params, config, refusal := ledgerPredicates(ctx, node, relationAddress(iacService, recoverRelation))
	if refusal != nil {
		return refusal
	}
	resource := popPredicate(params, "resource_key")
	outcome := popPredicate(params, "outcome")
	identity := popPredicate(params, "identity")
	version := popPredicate(params, "version")
	if len(params) != 0 || resource == "" || version == "" || (outcome != "absent" && outcome != "applied") {
		return refuse(fmt.Errorf("recovery requires resource_key, version and verified outcome absent or applied; " +
			"applied also requires identity"))
	}
	recoverer, ok := config.(ledgerRecoverer)
	if !ok {
		return refuse(fmt.Errorf("upgrade Omni to the release providing explicit ledger recovery"))
	}
	return func() internaldto.ExecutorOutput {
		args := omnisdk.Args{}
		err := configureCaller(ctx, &args)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		plan := ledgerRecoveryPlan{
			recoverer: recoverer, resource: resource, outcome: outcome,
			identity: identity, version: version, args: args,
		}
		output := streamPlan(ctx, plan, recoverRelation, node.SelectExprs)
		if outputErr := output.GetError(); outputErr != nil {
			return internaldto.NewErroneousExecutorOutput(iacOutcomeError{cause: outputErr})
		}
		return output
	}
}
