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

func recoverFunc(
	ctx queryContext, params map[string]string, returning sqlparser.SelectExprs,
) (func() internaldto.ExecutorOutput, error) {
	config, ledgerErr := iacLedgerConfig(ctx, params, relationAddress(iacService, recoverRelation))
	if ledgerErr != nil {
		return nil, ledgerErr
	}
	resource := popPredicate(params, "resource_key")
	outcome := popPredicate(params, "outcome")
	identity := popPredicate(params, "identity")
	version := popPredicate(params, "version")
	validOutcome := outcome == "absent" || (outcome == "applied" && identity != "")
	if len(params) != 0 || resource == "" || version == "" || !validOutcome {
		return nil, fmt.Errorf("recovery requires resource_key, version and verified outcome absent or applied; " +
			"applied also requires identity")
	}
	recoverer, ok := config.(ledgerRecoverer)
	if !ok {
		return nil, fmt.Errorf("upgrade Omni to the release providing explicit ledger recovery")
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
		return streamIaCMutation(ctx, plan, recoverRelation, returning)
	}, nil
}
