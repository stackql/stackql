package intrinsic

// stackql_preview.iac presents omnisdk's converge run and destroy as relations, and the
// blueprints that render a run as a catalogue alongside them. A run is issued as an imperative -
// these resources, under this collection name - and idempotence, ordering, locking and
// compensation are the SDK's problem, not stackql's.

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/public/authorization"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// stateDirName is where the ledger and run journals live when the operator sets
// no iacStateRoot. It sits under the application root, which is already local
// disk: O_EXCL and link are unreliable on a network share.
const stateDirName = "iac"

// errStateEscapesRoot refuses a 'state' predicate that would put the ledger
// outside the state root.
var errStateEscapesRoot = errors.New("must name a directory inside the iac state root")

// runIDLayout stamps a run that names no run id, as omnisdk's Converge always has.
const runIDLayout = "20060102T150405Z"

// iacSelectFunc routes a SELECT over stackql_preview.iac. Opening the plan
// performs the run, so a select over the converge relation is a mutation in
// every sense except its syntax.
func iacSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	resource string,
) (func() internaldto.ExecutorOutput, bool) {
	var action func() internaldto.ExecutorOutput
	switch {
	case strings.EqualFold(resource, convergeRelation):
		action = convergeSelectFunc(ctx, node)
	case strings.EqualFold(resource, destroyRelation):
		action = destroySelectFunc(ctx, node)
	default:
		action = blueprintSelectFunc(resource)
	}
	return func() internaldto.ExecutorOutput {
		output := action()
		if err := output.GetError(); err != nil {
			return internaldto.NewErroneousExecutorOutput(iacOutcomeError{cause: err})
		}
		return output
	}, true
}

// convergeSelectFunc routes a SELECT over the converge relation.
func convergeSelectFunc(ctx queryContext, node *sqlparser.Select) func() internaldto.ExecutorOutput {
	relation := relationAddress(iacService, convergeRelation)
	params, ledger, refusal := ledgerPredicates(ctx, node, relation)
	if refusal != nil {
		return refusal
	}
	handle := popPredicate(params, blueprintPredicate)
	spec := popPredicate(params, resourcesPredicate)
	file := popPredicate(params, resourcesFilePredicate)
	if countSupplied(handle, spec, file) != 1 {
		return refuse(fmt.Errorf(
			"relation '%s' needs exactly one of '%s', '%s' or '%s'; "+
				"a blueprint renders the resources, a specification or a file states them",
			relation, blueprintPredicate, resourcesPredicate, resourcesFilePredicate))
	}
	return func() internaldto.ExecutorOutput {
		inputs, environment, err := iacBindings(params)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		resources, err := convergeResources(handle, spec, file, params)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		args, argsErr := previewArgs(ctx, convergeCloud(resources), params)
		if argsErr != nil {
			return internaldto.NewErroneousExecutorOutput(argsErr)
		}
		args.Inputs, args.Environment = inputs, environment
		plan, planErr := omnisdk.Prepare(registryRoot(ctx), omnisdk.DesiredState(resources), ledger, args)
		if planErr != nil {
			return internaldto.NewErroneousExecutorOutput(planErr)
		}
		return streamPlan(ctx, plan, convergeRelation, node.SelectExprs)
	}
}

// destroySelectFunc routes a SELECT over the destroy relation. What the collection holds comes
// from its ledger alone, so the remaining predicates are only the scope the deletes run in. With
// no resources to name a provider, credentials come from each cloud's canonical environment
// variables.
func destroySelectFunc(ctx queryContext, node *sqlparser.Select) func() internaldto.ExecutorOutput {
	relation := relationAddress(iacService, destroyRelation)
	params, ledger, refusal := ledgerPredicates(ctx, node, relation)
	if refusal != nil {
		return refusal
	}
	selection, selected := params["keys"]
	delete(params, "keys")
	var keys []string
	if selected {
		if err := json.Unmarshal([]byte(selection), &keys); err != nil || len(keys) == 0 {
			return refuse(errors.New("partial destroy requires a nonempty JSON array of keys"))
		}
		for _, key := range keys {
			if strings.TrimSpace(key) == "" {
				return refuse(errors.New("partial destroy requires nonempty keys"))
			}
		}
	}
	return func() internaldto.ExecutorOutput {
		args, argsErr := previewArgs(ctx, "", params)
		if argsErr != nil {
			return internaldto.NewErroneousExecutorOutput(argsErr)
		}
		plan, planErr := omnisdk.Prepare(registryRoot(ctx), omnisdk.Destroy(keys...), ledger, args)
		if planErr != nil {
			return internaldto.NewErroneousExecutorOutput(planErr)
		}
		return streamPlan(ctx, plan, destroyRelation, node.SelectExprs)
	}
}

// ledgerPredicates reads the predicates every ledger backed relation takes: the collection it
// addresses, and the state directory and run id, each defaulted when absent. The state directory
// is a name under the operator's state root, never a path of the caller's choosing, and it is
// checked before anything touches disk. The predicates left over are returned for the caller.
func ledgerPredicates(
	ctx queryContext,
	node *sqlparser.Select,
	relation string,
) (map[string]string, omnisdk.WALConfig, func() internaldto.ExecutorOutput) {
	params, refusal := previewPredicates(node, relation)
	if refusal != nil {
		return nil, nil, refusal
	}
	collection := popPredicate(params, collectionPredicate)
	if strings.TrimSpace(collection) == "" {
		return nil, nil, refuse(fmt.Errorf(
			"relation '%s' needs a '%s' predicate; it is the ledger key prefix, the lease scope "+
				"and the correlation stamp, and it is how a later run addresses the same resources",
			relation, collectionPredicate))
	}
	root := previewCfg.getIaCStateRoot()
	if root == "" {
		root = filepath.Join(ctx.GetRuntimeContext().ApplicationFilesRootPath, stateDirName)
	}
	if ctx.GetRuntimeContext().AccessConfigFile != "" {
		host, ok := ctx.(interface{ GetCaller() authorization.Session })
		if !ok || host.GetCaller() == nil {
			return nil, nil, refuse(errors.New("authenticated caller required for IaC state"))
		}
		principal := host.GetCaller().Principal()
		owner := "principal:" + principal.ID()
		if tenant := principal.Attributes()["tenant"]; tenant != "" {
			owner = "tenant:" + tenant
		}
		root = filepath.Join(root, fmt.Sprintf("%x", sha256.Sum256([]byte(owner))))
	}
	state, stateErr := resolveStateDir(root, popPredicate(params, statePredicate))
	if stateErr != nil {
		return nil, nil, refuse(fmt.Errorf("relation '%s': '%s' %w", relation, statePredicate, stateErr))
	}
	runID := popPredicate(params, runIDPredicate)
	if runID == "" {
		runID = time.Now().UTC().Format(runIDLayout) + "-" + rand.Text()
	}
	return params, omnisdk.NewLedger(collection, state, runID), nil
}

// resolveStateDir places the named state directory under root, or returns root when no name is
// given. The name must be relative and stay inside root, lexically and once symlinks (and, on
// Windows, junctions) in its existing part are followed, so a link cannot carry it out either.
func resolveStateDir(root, name string) (string, error) {
	if name == "" {
		return root, nil
	}
	if !filepath.IsLocal(name) {
		return "", errStateEscapesRoot
	}
	resolvedRoot, err := resolveExisting(root)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, name)
	resolvedDir, err := resolveExisting(dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedDir)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errStateEscapesRoot
	}
	return dir, nil
}

// resolveExisting follows the links in the longest prefix of path that exists and appends the
// rest unchanged; a part that does not exist yet cannot be a link.
func resolveExisting(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var rest []string
	for {
		resolved, evalErr := filepath.EvalSymlinks(abs)
		if evalErr == nil {
			return filepath.Join(append([]string{resolved}, rest...)...), nil
		}
		if !errors.Is(evalErr, fs.ErrNotExist) {
			return "", evalErr
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return filepath.Join(append([]string{abs}, rest...)...), nil
		}
		rest = append([]string{filepath.Base(abs)}, rest...)
		abs = parent
	}
}

// countSupplied counts the non-empty values.
func countSupplied(values ...string) int {
	count := 0
	for _, value := range values {
		if value != "" {
			count++
		}
	}
	return count
}

// convergeResources renders what a run converges, from whichever of the three
// predicates was supplied.
func convergeResources(
	handle, spec, file string, params map[string]string) ([]omnisdk.ManagedResource, error) {
	switch {
	case handle != "":
		return blueprintResources(handle, params)
	case file != "":
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return specResources(string(data))
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
	resources, err := omnisdk.ParseResources([]byte(spec))
	if err != nil {
		return nil, fmt.Errorf("intrinsic: invalid resource specification: %w", err)
	}
	seen := map[string]bool{}
	for _, resource := range resources {
		if seen[resource.Key()] {
			return nil, fmt.Errorf("duplicate resource key %q", resource.Key())
		}
		seen[resource.Key()] = true
	}
	return resources, nil
}

// convergeCloud selects the initial provider credential. Additional providers
// currently use SDK credential resolution from their canonical environment.
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
func blueprintSelectFunc(resource string) func() internaldto.ExecutorOutput {
	if _, ok := blueprintFor(resource); !ok {
		return refuse(fmt.Errorf(
			"'%s.%s' has no relation '%s'; run SHOW RESOURCES IN %s.%s to list them",
			ProviderName, iacService, resource, ProviderName, iacService))
	}
	return refuse(fmt.Errorf(
		"'%s' renders a deployment rather than rows; "+
			"apply it with SELECT ... FROM %s WHERE %s = '%s' AND %s = '<name>'",
		relationAddress(iacService, resource),
		relationAddress(iacService, convergeRelation),
		blueprintPredicate, resource, collectionPredicate))
}

// iacBindings keeps typed values separate from provider execution scope.
func iacBindings(params map[string]string) (map[string]any, map[string]any, error) {
	var maps [2]map[string]any
	for index, name := range []string{"inputs", "environment"} {
		raw := popPredicate(params, name)
		if raw == "" {
			continue
		}
		decoder := json.NewDecoder(bytes.NewBufferString(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&maps[index]); err != nil {
			return nil, nil, fmt.Errorf("%s must be a JSON object: %w", name, err)
		}
		if maps[index] == nil {
			return nil, nil, fmt.Errorf("%s must be a JSON object", name)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, nil, fmt.Errorf("%s must contain one JSON object", name)
		}
	}
	return maps[0], maps[1], nil
}

// iacRows preserves diagnostic rows while turning unsuccessful outcomes into SQL errors.
type iacRows struct {
	omnisdk.Rows
	mu      sync.Mutex
	failure error
}

func (r *iacRows) Next() bool {
	if !r.Rows.Next() {
		return false
	}
	row := r.Rows.Row()
	status, _ := row["status"].(string)
	failed := strings.Contains(strings.ToLower(status), "fail") || strings.Contains(strings.ToLower(status), "outstanding")
	if value := row["error"]; value != nil && fmt.Sprint(value) != "" {
		failed = true
	}
	if value := row["outstanding"]; value != nil && fmt.Sprint(value) != "[]" && fmt.Sprint(value) != "" {
		failed = true
	}
	if failed {
		r.mu.Lock()
		r.failure = iacOutcomeError{key: fmt.Sprint(row["key"]), status: status}
		r.mu.Unlock()
	}
	return true
}
func (r *iacRows) Err() error {
	if err := r.Rows.Err(); err != nil {
		return iacOutcomeError{cause: err}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.failure
}

type iacOutcomeError struct {
	key, status string
	cause       error
}

func (e iacOutcomeError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return fmt.Sprintf("IaC operation did not complete successfully for key %s (status %s); inspect the journal",
		e.key, e.status)
}
func (iacOutcomeError) IaCFailure() bool { return true }

func (e iacOutcomeError) Unwrap() error { return e.cause }
