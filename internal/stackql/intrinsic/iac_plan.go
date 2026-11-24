package intrinsic

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

func iacPlanColumns() []column {
	return []column{
		{name: "resource_key", dataType: "string", description: "resource key within the collection"},
		{name: "action", dataType: "string", description: "create, update, replace, no-op or unknown"},
		{name: "reason", dataType: "string", description: "why the action was selected"},
		{name: "depends_on", dataType: "string", description: "JSON array of dependency keys"},
		{name: "change_set", dataType: "string", description: "versioned JSON accepted by INSERT INTO iac.plan"},
	}
}

type iacPreviewPlan func(context.Context) (omnisdk.ChangeSet, error)

func (preview iacPreviewPlan) Open(ctx context.Context) (omnisdk.Rows, error) {
	set, err := preview(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(set)
	if err != nil {
		return nil, err
	}
	serialized := string(raw)
	rows := make([]omnisdk.Row, 0, len(set.Changes))
	for _, change := range set.Changes {
		dependencies, depErr := json.Marshal(append([]string{}, change.DependsOn...))
		if depErr != nil {
			return nil, depErr
		}
		rows = append(rows, omnisdk.Row{
			"resource_key": change.Key, "action": change.Action, "reason": change.Reason,
			"depends_on": string(dependencies), "change_set": serialized,
		})
	}
	return &materializedRelationRows{ctx: ctx, rows: rows}, nil
}

func iacPlanSelect(ctx queryContext, node *sqlparser.Select) func() internaldto.ExecutorOutput {
	relation := relationAddress(iacService, planRelation)
	params, refusal := previewPredicates(node, relation)
	if refusal != nil {
		return refusal
	}
	if err := checkHost(ctx, "host/iac/plan", "execute"); err != nil {
		return refuse(err)
	}
	wal, ledgerErr := iacLedgerConfig(ctx, params, relation)
	if ledgerErr != nil {
		return refuse(ledgerErr)
	}
	spec := popPredicate(params, resourcesPredicate)
	if spec == "" {
		return refuse(errors.New("iac.plan SELECT requires the resources parameter"))
	}
	return func() internaldto.ExecutorOutput {
		resources, err := specResources(spec)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		args, err := iacPlanArgs(ctx, resources, params)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		preview := iacPreviewPlan(func(execution context.Context) (omnisdk.ChangeSet, error) {
			return omnisdk.Preview(execution, registryRoot(ctx), omnisdk.DesiredState(resources), wal, args)
		})
		return streamPlan(ctx, preview, planRelation, node.SelectExprs, iacPlanColumns()...)
	}
}

func iacPlanInsert(
	ctx queryContext, params map[string]string, returning sqlparser.SelectExprs,
) (func() internaldto.ExecutorOutput, error) {
	relation := relationAddress(iacService, planRelation)
	wal, ledgerErr := iacLedgerConfig(ctx, params, relation)
	if ledgerErr != nil {
		return nil, ledgerErr
	}
	raw := popPredicate(params, "change_set")
	if raw == "" {
		return nil, errors.New("iac.plan INSERT requires the change_set parameter")
	}
	var set omnisdk.ChangeSet
	if err := json.Unmarshal([]byte(raw), &set); err != nil {
		return nil, err
	}
	return func() internaldto.ExecutorOutput {
		resources, err := specResources(string(set.Desired))
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		args, err := iacPlanArgs(ctx, resources, params)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		plan, err := omnisdk.ApplyChangeSet(registryRoot(ctx), set, wal, args)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		return streamIaCMutation(ctx, plan, convergeRelation, returning)
	}, nil
}

func iacPlanArgs(
	ctx queryContext, resources []omnisdk.ManagedResource, params map[string]string,
) (omnisdk.Args, error) {
	inputs, environment, err := iacBindings(params)
	if err != nil {
		return omnisdk.Args{}, err
	}
	args, err := previewArgs(ctx, convergeCloud(resources), params)
	args.Inputs, args.Environment = inputs, environment
	return args, err
}
