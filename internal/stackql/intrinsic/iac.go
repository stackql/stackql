package intrinsic

// stackql_preview.iac presents omnisdk's converge run as a relation, and the
// blueprints that render one as a catalogue alongside it. A run is issued as an imperative - these
// resources, under this collection name - and idempotence, ordering, locking and
// compensation are the SDK's problem, not stackql's.

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// stateDirName is where the ledger and run journals live when a query names no
// state directory. It sits under the application root, which is already local
// disk: O_EXCL and link are unreliable on a network share.
const stateDirName = "iac"

// resourceDTO is the wire shape of one entry in the 'resources' predicate. It
// is omnisdk's ManagedResource with the interface flattened into data.
type resourceDTO struct {
	Key              string            `json:"key"`
	Provider         string            `json:"provider"`
	Address          string            `json:"address"`
	Desired          json.RawMessage   `json:"desired"`
	Params           map[string]string `json:"params"`
	Inbound          []arrivalDTO      `json:"inbound"`
	ViaType          string            `json:"viaType"`
	ViaProgram       string            `json:"viaProgram"`
	Identity         string            `json:"identity"`
	AddressedBy      string            `json:"addressedBy"`
	CorrelationParam string            `json:"correlationParam"`
}

// arrivalDTO is one value reaching a resource from a sibling in the same
// collection.
type arrivalDTO struct {
	From string `json:"from"`
	As   string `json:"as"`
}

// iacSelectFunc routes a SELECT over stackql_preview.iac. Opening the plan
// performs the run, so a select over the converge relation is a mutation in
// every sense except its syntax.
func iacSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	resource string,
) (func() internaldto.ExecutorOutput, bool) {
	if !strings.EqualFold(resource, convergeRelation) {
		return blueprintSelectFunc(resource)
	}
	relation := relationAddress(iacService, convergeRelation)
	params, refusal := previewPredicates(node, relation)
	if refusal != nil {
		return refusal, true
	}
	collection := popPredicate(params, collectionPredicate)
	if strings.TrimSpace(collection) == "" {
		return refuse(fmt.Errorf(
			"relation '%s' needs a '%s' predicate; it is the ledger key prefix, the lease scope "+
				"and the correlation stamp, and it is how a later run addresses the same resources",
			relation, collectionPredicate)), true
	}
	handle := popPredicate(params, blueprintPredicate)
	spec := popPredicate(params, resourcesPredicate)
	state := popPredicate(params, statePredicate)
	runID := popPredicate(params, runIDPredicate)
	if (handle == "") == (spec == "") {
		return refuse(fmt.Errorf(
			"relation '%s' needs exactly one of '%s' or '%s'; "+
				"a blueprint renders the resources, a specification states them",
			relation, blueprintPredicate, resourcesPredicate)), true
	}
	if state == "" {
		state = filepath.Join(ctx.GetRuntimeContext().ApplicationFilesRootPath, stateDirName)
	}
	return func() internaldto.ExecutorOutput {
		resources, err := convergeResources(handle, spec, params)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		plan, planErr := omnisdk.Converge(
			registryRoot(ctx), collection, state, runID, resources,
			previewArgs(ctx, convergeCloud(resources), params))
		if planErr != nil {
			return internaldto.NewErroneousExecutorOutput(planErr)
		}
		return streamPlan(ctx, plan, convergeRelation, node.SelectExprs)
	}, true
}

// convergeResources renders what a run converges, from whichever of the two
// predicates was supplied.
func convergeResources(
	handle, spec string, params map[string]string) ([]omnisdk.ManagedResource, error) {
	if handle != "" {
		return blueprintResources(handle, params)
	}
	return specResources(spec)
}

// blueprintResources renders a named blueprint. Only the inputs it declares are
// passed: it rejects unknown ones rather than ignoring them, and the remaining
// predicates are the run's scope rather than the blueprint's.
func blueprintResources(handle string, params map[string]string) ([]omnisdk.ManagedResource, error) {
	blueprint, ok := blueprintFor(handle)
	if !ok {
		return nil, fmt.Errorf(
			"intrinsic: no blueprint '%s'; run SHOW RESOURCES IN %s.%s to list them",
			handle, ProviderName, iacService)
	}
	inputs := make(map[string]string, len(params))
	for _, param := range blueprint.Params() {
		if value, supplied := params[param.Name]; supplied {
			inputs[param.Name] = value
		}
	}
	return blueprint.Resources(inputs)
}

// specResources reads the 'resources' predicate.
func specResources(spec string) ([]omnisdk.ManagedResource, error) {
	var specs []resourceDTO
	if err := json.Unmarshal([]byte(spec), &specs); err != nil {
		return nil, fmt.Errorf(
			"intrinsic: '%s' is not a valid resource specification: %w", resourcesPredicate, err)
	}
	out := make([]omnisdk.ManagedResource, 0, len(specs))
	for _, resource := range specs {
		arrivals := make([]omnisdk.Arrival, 0, len(resource.Inbound))
		for _, arrival := range resource.Inbound {
			arrivals = append(arrivals, omnisdk.Arrival{From: arrival.From, As: arrival.As})
		}
		out = append(out, omnisdk.NewResource(
			resource.Key, resource.Provider, resource.Address, resource.Desired, resource.Params,
			arrivals, resource.ViaType, resource.ViaProgram,
			resource.Identity, resource.AddressedBy, resource.CorrelationParam))
	}
	return out, nil
}

// convergeCloud is the cloud whose credential a run uses. omnisdk takes one
// credential per run, so a collection spanning two clouds carries the first
// resource's and leaves the rest to the canonical environment variables.
func convergeCloud(resources []omnisdk.ManagedResource) string {
	if len(resources) == 0 {
		return ""
	}
	return resources[0].Provider()
}

// blueprintHandle is the relation name a blueprint takes. A handle is written
// with hyphens, which no unquoted SQL identifier can carry.
func blueprintHandle(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// blueprintFor resolves a blueprint by handle, accepting either the handle as
// written or the relation name it takes.
func blueprintFor(name string) (omnisdk.Blueprint, bool) {
	if blueprint, ok := omnisdk.BlueprintFor(name); ok {
		return blueprint, true
	}
	for _, blueprint := range omnisdk.Blueprints() {
		if strings.EqualFold(blueprintHandle(blueprint.Handle()), blueprintHandle(name)) {
			return blueprint, true
		}
	}
	return nil, false
}

// blueprintTables presents the blueprints as relations, which is what makes
// them discoverable in the same shape as the query catalogue.
func blueprintTables() []table {
	blueprints := omnisdk.Blueprints()
	out := make([]table, 0, len(blueprints))
	for _, blueprint := range blueprints {
		out = append(out, table{
			service:     iacService,
			name:        blueprintHandle(blueprint.Handle()),
			description: blueprint.Summary(),
		})
	}
	return out
}

// blueprintColumns presents a blueprint's declared inputs as columns, so
// DESCRIBE answers "what does this deployment need".
func blueprintColumns(blueprint omnisdk.Blueprint) []column {
	params := blueprint.Params()
	out := make([]column, 0, len(params))
	for _, param := range params {
		description := param.Description
		if param.Required {
			description = "required; " + description
		}
		out = append(out, column{
			name:        param.Name,
			dataType:    param.Type.Name,
			description: description,
		})
	}
	return out
}

// blueprintSelectFunc refuses a select over a blueprint relation. A blueprint is
// a way of building a resource set, not a thing to read: naming it in a converge
// run is how it is applied.
func blueprintSelectFunc(resource string) (func() internaldto.ExecutorOutput, bool) {
	if _, ok := blueprintFor(resource); !ok {
		return refuse(fmt.Errorf(
			"'%s.%s' has no relation '%s'; run SHOW RESOURCES IN %s.%s to list them",
			ProviderName, iacService, resource, ProviderName, iacService)), true
	}
	return refuse(fmt.Errorf(
		"'%s' renders a deployment rather than rows; "+
			"apply it with SELECT ... FROM %s WHERE %s = '%s' AND %s = '<name>'",
		relationAddress(iacService, resource),
		relationAddress(iacService, convergeRelation),
		blueprintPredicate, resource, collectionPredicate)), true
}
