package intrinsic

// stackql_preview.dynamic_graph presents omnisdk's multi-exchange query as a
// single relation.
// A document describes one provider and cannot state a relationship spanning
// two, so the caller states it: which exchanges take part, and what flows
// between them. That statement is the 'spec' predicate, and it maps onto the
// SDK's own constructors one field at a time.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// graphSpecDTO is the wire shape of the 'spec' predicate. It is omnisdk's Graph
// with the interfaces flattened into data, and nothing else: every field here
// is handed to a constructor unaltered.
type graphSpecDTO struct {
	Addresses []string      `json:"addresses"`
	Wirings   []wiringDTO   `json:"wirings"`
	Overrides []overrideDTO `json:"overrides"`
}

// wiringDTO is everything arriving at one consumer, and how its inbox becomes
// its inputs. ViaType and ViaProgram are T_in; empty is identity, which is the
// common case.
type wiringDTO struct {
	To         string       `json:"to"`
	Inbound    []inboundDTO `json:"inbound"`
	ViaType    string       `json:"viaType"`
	ViaProgram string       `json:"viaProgram"`
	Provides   []string     `json:"provides"`
}

// inboundDTO is one value arriving at a consumer: an attribute a producer
// emits, landing in the consumer's inbox under a name of the caller's choosing.
type inboundDTO struct {
	From string `json:"from"`
	Src  string `json:"src"`
	As   string `json:"as"`
}

// overrideDTO corrects what a document says about one exchange's response,
// where it is wrong for this engine.
type overrideDTO struct {
	Address     string `json:"address"`
	ObjectKey   string `json:"objectKey"`
	MediaType   string `json:"mediaType"`
	ProgramType string `json:"programType"`
	ProgramBody string `json:"programBody"`
}

// dynamicSelectFunc routes a SELECT over stackql_preview.dynamic_graph.query.
func dynamicSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	resource string,
) (func() internaldto.ExecutorOutput, bool) {
	relation := relationAddress(dynamicGraphService, graphRelation)
	if !strings.EqualFold(resource, graphRelation) {
		return refuse(fmt.Errorf(
			"'%s' has no relation '%s'; the graph query is '%s'",
			ProviderName+"."+dynamicGraphService, resource, relation)), true
	}
	params, refusal := previewPredicates(node, relation)
	if refusal != nil {
		return refusal, true
	}
	spec := popPredicate(params, specPredicate)
	if strings.TrimSpace(spec) == "" {
		return refuse(fmt.Errorf(
			"relation '%s' needs a '%s' predicate naming the exchanges and their wiring",
			relation, specPredicate)), true
	}
	return func() internaldto.ExecutorOutput {
		graph, err := buildGraph(spec)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		plan, planErr := omnisdk.NewGraphQuery(
			registryRoot(ctx), graph, previewArgs(ctx, graphCloud(graph.Addresses()), params))
		if planErr != nil {
			return internaldto.NewErroneousExecutorOutput(planErr)
		}
		return streamPlan(ctx, plan, graphRelation, node.SelectExprs)
	}, true
}

// buildGraph turns the 'spec' predicate into the SDK's graph. Validation is the
// SDK's: an edge naming an exchange the query does not run is caught there,
// where it can name the address.
func buildGraph(spec string) (omnisdk.Graph, error) {
	var dtoSpec graphSpecDTO
	if err := json.Unmarshal([]byte(spec), &dtoSpec); err != nil {
		return nil, fmt.Errorf("intrinsic: '%s' is not a valid graph specification: %w", specPredicate, err)
	}
	wirings := make([]omnisdk.Wiring, 0, len(dtoSpec.Wirings))
	for _, wiring := range dtoSpec.Wirings {
		inbound := make([]omnisdk.Inbound, 0, len(wiring.Inbound))
		for _, arrival := range wiring.Inbound {
			inbound = append(inbound, omnisdk.NewInbound(arrival.From, arrival.Src, arrival.As))
		}
		wirings = append(wirings, omnisdk.NewWiring(
			wiring.To, inbound, wiring.ViaType, wiring.ViaProgram, wiring.Provides...))
	}
	overrides := make([]omnisdk.Override, 0, len(dtoSpec.Overrides))
	for _, override := range dtoSpec.Overrides {
		overrides = append(overrides, omnisdk.NewOverride(
			override.Address, override.ObjectKey, override.MediaType,
			override.ProgramType, override.ProgramBody))
	}
	return omnisdk.NewGraph(dtoSpec.Addresses, wirings, overrides...)
}

// graphCloud is the cloud whose credential a graph runs under. omnisdk takes
// one credential per run, so a graph spanning two clouds carries the first
// address's and leaves the rest to the canonical environment variables, which
// is the SDK's own fallback.
func graphCloud(addresses []string) string {
	if len(addresses) == 0 {
		return ""
	}
	cloud, _, _ := strings.Cut(strings.TrimPrefix(addresses[0], UnstablePrefix), ".")
	return cloud
}
