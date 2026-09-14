package intrinsic //nolint:testpackage // tests unexported preview-service plumbing

import (
	"strings"
	"testing"
)

// withUnstable opts the document-driven surface in for one test and restores
// whatever the package had afterwards.
func withUnstable(t *testing.T, enabled bool) {
	t.Helper()
	previous := previewCfg
	previewCfg = newBackendInput(previewCfgDTO{Unstable: enabled})
	t.Cleanup(func() { previewCfg = previous })
}

func TestPreviewServiceRequiresOptIn(t *testing.T) {
	withUnstable(t, false)
	for _, name := range []string{dynamicGraphService, iacService} {
		if _, ok := previewService(ProviderName, name, ""); ok {
			t.Errorf("previewService(%q) = true without the unstable opt-in", name)
		}
	}
	// The catalogue must read exactly as it did before anyone asked for them.
	rows := map[string]map[string]interface{}{}
	appendPreviewServices(rows, false)
	if len(rows) != 0 {
		t.Errorf("appendPreviewServices added %d rows without the opt-in", len(rows))
	}
}

func TestPreviewServiceCanonicalises(t *testing.T) {
	withUnstable(t, true)
	cases := map[string]string{
		"dynamic_graph":  dynamicGraphService,
		"DYNAMIC_GRAPH":  dynamicGraphService,
		"  iac  ":        iacService,
		"IaC":            iacService,
		"audit":          "",
		"dynamic_graphs": "",
		"iac_extra":      "",
	}
	for input, want := range cases {
		got, ok := previewService(ProviderName, input, "")
		if want == "" {
			if ok {
				t.Errorf("previewService(%q) = %q, want no match", input, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("previewService(%q) = %q, %v; want %q, true", input, got, ok, want)
		}
	}
}

// The extended services belong to stackql_preview alone; a document-driven
// bundle that happens to ship a service of the same name is not one of them.
func TestPreviewServiceRejectsOtherProviders(t *testing.T) {
	withUnstable(t, true)
	for _, provider := range []string{UnstablePrefix + "aws", "github", ""} {
		if _, ok := previewService(provider, iacService, ""); ok {
			t.Errorf("previewService(%q, %q) matched a non-preview provider", provider, iacService)
		}
	}
	// It does resolve through the current provider when none is qualified.
	if _, ok := previewService("", iacService, ProviderName); !ok {
		t.Error("previewService did not resolve through the current provider")
	}
}

func TestPreviewTables(t *testing.T) {
	withUnstable(t, true)
	graph := previewTables(dynamicGraphService)
	if len(graph) != 1 || graph[0].name != graphRelation {
		t.Errorf("previewTables(dynamic_graph) = %v, want [%s]", graph, graphRelation)
	}
	iac := previewTables(iacService)
	if len(iac) < 2 || iac[0].name != convergeRelation {
		t.Fatalf("previewTables(iac) = %v, want %s first", iac, convergeRelation)
	}
	var sawBlueprint bool
	for _, tbl := range iac[1:] {
		if tbl.name == "aws_vpc_subnet" {
			sawBlueprint = true
		}
		if tbl.service != iacService {
			t.Errorf("blueprint %q sits under service %q", tbl.name, tbl.service)
		}
	}
	if !sawBlueprint {
		t.Error("previewTables(iac) omitted the aws_vpc_subnet blueprint")
	}
}

func TestPopPredicateRemovesControlKeys(t *testing.T) {
	params := map[string]string{
		collectionPredicate: "scratch",
		"region":            "us-east-1",
	}
	if got := popPredicate(params, collectionPredicate); got != "scratch" {
		t.Errorf("popPredicate = %q, want %q", got, "scratch")
	}
	if _, still := params[collectionPredicate]; still {
		t.Error("popPredicate left the control key in the scope")
	}
	if got := popPredicate(params, statePredicate); got != "" {
		t.Errorf("popPredicate of an absent key = %q, want empty", got)
	}
	if len(params) != 1 || params["region"] != "us-east-1" {
		t.Errorf("scope after popping = %v, want only the region", params)
	}
}

func TestBuildGraphMapsSpecOntoSDK(t *testing.T) {
	spec := `{
		"addresses": ["stackql_unstable_aws.ec2.vpcs", "stackql_unstable_aws.ec2.subnets"],
		"wirings": [{
			"to": "stackql_unstable_aws.ec2.subnets",
			"inbound": [{"from": "stackql_unstable_aws.ec2.vpcs", "src": "VpcId", "as": "vpc_id"}],
			"viaType": "golang_template_json_v0.1.0",
			"viaProgram": "{\"Filter.1.Name\":\"vpc-id\"}",
			"provides": ["Filter.1.Name", "Filter.1.Value.1"]
		}],
		"overrides": [{"address": "stackql_unstable_aws.ec2.vpcs", "objectKey": "$.items"}]
	}`
	graph, err := buildGraph(spec)
	if err != nil {
		t.Fatalf("buildGraph: %v", err)
	}
	if got := len(graph.Addresses()); got != 2 {
		t.Fatalf("addresses = %d, want 2", got)
	}
	wirings := graph.Wirings()
	if len(wirings) != 1 {
		t.Fatalf("wirings = %d, want 1", len(wirings))
	}
	wiring := wirings[0]
	if wiring.To() != "stackql_unstable_aws.ec2.subnets" {
		t.Errorf("wiring To = %q", wiring.To())
	}
	inbound := wiring.Inbound()
	if len(inbound) != 1 || inbound[0].Src() != "VpcId" || inbound[0].As() != "vpc_id" {
		t.Errorf("inbound = %+v, want one VpcId -> vpc_id", inbound)
	}
	viaType, viaProgram := wiring.Via()
	if viaType != "golang_template_json_v0.1.0" || viaProgram == "" {
		t.Errorf("via = %q, %q", viaType, viaProgram)
	}
	if got := len(wiring.Provides()); got != 2 {
		t.Errorf("provides = %d, want 2", got)
	}
	overrides := graph.Overrides()
	if len(overrides) != 1 || overrides[0].ObjectKey() != "$.items" {
		t.Errorf("overrides = %+v, want one $.items correction", overrides)
	}
}

// An identity wiring is the common case: a value already shaped for the
// consumer passes straight through, and NewInbound names it after its source.
func TestBuildGraphIdentityInboundTakesSourceName(t *testing.T) {
	graph, err := buildGraph(`{
		"addresses": ["a.b.c", "d.e.f"],
		"wirings": [{"to": "d.e.f", "inbound": [{"from": "a.b.c", "src": "VpcId"}]}]
	}`)
	if err != nil {
		t.Fatalf("buildGraph: %v", err)
	}
	inbound := graph.Wirings()[0].Inbound()[0]
	if inbound.As() != "VpcId" {
		t.Errorf("As = %q, want the source name %q", inbound.As(), "VpcId")
	}
}

func TestBuildGraphRejectsMalformedSpec(t *testing.T) {
	if _, err := buildGraph("{ not json"); err == nil {
		t.Fatal("buildGraph accepted a malformed specification")
	}
}

func TestSpecResourcesMapsOntoSDK(t *testing.T) {
	spec := `[{
		"key": "aws/ec2/vpc",
		"provider": "aws",
		"address": "ec2.vpcs",
		"desired": {"CidrBlock": "10.42.0.0/16"},
		"params": {"TagSpecification.1.ResourceType": "vpc"},
		"identity": "line_items.VpcId",
		"addressedBy": "VpcId",
		"correlationParam": "TagSpecification.1.Tag.1.Value"
	}, {
		"key": "aws/ec2/subnet",
		"provider": "aws",
		"address": "ec2.subnets",
		"desired": {"CidrBlock": "10.42.1.0/24"},
		"inbound": [{"From": "aws/ec2/vpc", "As": "VpcId"}],
		"identity": "line_items.SubnetId",
		"addressedBy": "SubnetId"
	}]`
	resources, err := specResources(spec)
	if err != nil {
		t.Fatalf("specResources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(resources))
	}
	vpc := resources[0]
	if vpc.Key() != "aws/ec2/vpc" || vpc.Provider() != "aws" || vpc.Address() != "ec2.vpcs" {
		t.Errorf("vpc addressing = %q %q %q", vpc.Key(), vpc.Provider(), vpc.Address())
	}
	if vpc.Identity() != "line_items.VpcId" || vpc.AddressedBy() != "VpcId" {
		t.Errorf("vpc identity = %q, addressedBy = %q", vpc.Identity(), vpc.AddressedBy())
	}
	if vpc.CorrelationParam() != "TagSpecification.1.Tag.1.Value" {
		t.Errorf("vpc correlation = %q", vpc.CorrelationParam())
	}
	// Desired is opaque: nothing between the predicate and the wire parses it,
	// so it must arrive as the bytes the caller wrote.
	if string(vpc.Desired()) != `{"CidrBlock": "10.42.0.0/16"}` {
		t.Errorf("vpc desired = %q, want the caller's bytes unaltered", vpc.Desired())
	}
	subnet := resources[1]
	arrivals := subnet.Inbound()
	if len(arrivals) != 1 || arrivals[0].From != "aws/ec2/vpc" || arrivals[0].As != "VpcId" {
		t.Errorf("subnet inbound = %+v, want one aws/ec2/vpc -> VpcId", arrivals)
	}
}

func TestSpecResourcesRejectsMalformedSpec(t *testing.T) {
	if _, err := specResources("nonsense"); err == nil {
		t.Fatal("specResources accepted a malformed specification")
	}
}

func TestBlueprintHandleIsSQLAddressable(t *testing.T) {
	if got := blueprintHandle("aws-vpc-subnet"); got != "aws_vpc_subnet" {
		t.Errorf("blueprintHandle = %q, want %q", got, "aws_vpc_subnet")
	}
}

// A blueprint is reachable by the handle as written and by the relation name it
// takes, so a caller who read SHOW RESOURCES can paste what they saw.
func TestBlueprintForAcceptsEitherSpelling(t *testing.T) {
	for _, name := range []string{"aws-vpc-subnet", "aws_vpc_subnet", "AWS_VPC_SUBNET"} {
		blueprint, ok := blueprintFor(name)
		if !ok {
			t.Errorf("blueprintFor(%q) found nothing", name)
			continue
		}
		if blueprint.Handle() != "aws-vpc-subnet" {
			t.Errorf("blueprintFor(%q).Handle() = %q", name, blueprint.Handle())
		}
	}
	if _, ok := blueprintFor("no-such-thing"); ok {
		t.Error("blueprintFor resolved a handle that does not exist")
	}
}

func TestBlueprintColumnsMarkRequiredInputs(t *testing.T) {
	blueprint, ok := blueprintFor("aws-vpc-subnet")
	if !ok {
		t.Fatal("aws-vpc-subnet blueprint is absent")
	}
	columns := blueprintColumns(blueprint)
	if len(columns) != len(blueprint.Params()) {
		t.Fatalf("columns = %d, want %d", len(columns), len(blueprint.Params()))
	}
	byName := make(map[string]column, len(columns))
	for _, col := range columns {
		byName[col.name] = col
	}
	region, present := byName["region"]
	if !present {
		t.Fatal("region column is absent")
	}
	if region.dataType != "string" {
		t.Errorf("region type = %q, want %q", region.dataType, "string")
	}
	if !strings.HasPrefix(region.description, "required; ") {
		t.Errorf("region description = %q, want it marked required", region.description)
	}
	tags, present := byName["vpc_tags"]
	if !present {
		t.Fatal("vpc_tags column is absent")
	}
	if strings.HasPrefix(tags.description, "required; ") {
		t.Errorf("vpc_tags description = %q, want it unmarked", tags.description)
	}
}
