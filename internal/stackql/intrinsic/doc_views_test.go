//nolint:testpackage // These tests exercise the unexported view-expansion helper.
package intrinsic

import (
	"strings"
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestInlineDocumentViewPreservesBusinessFiltersAndCallerScope(t *testing.T) {
	outer := parseSelect(t, "select bucket_name from aws.pseudo_s3.buckets where region = 'ap-southeast-2' and bucket_name like 'prod%' order by bucket_name")
	view := parseSelect(t, "select region,json_extract(Properties, '$.BucketName') as bucket_name from aws.cloud_control.resources where region = 'us-east-1' and data__TypeName = 'AWS::S3::Bucket'")
	original := sqlparser.String(outer)
	// The public entry point clones before inlining; exercise that same boundary.
	cloned, err := sqlparser.Parse(original)
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := inlineDocumentView(cloned.(*sqlparser.Select), view, "buckets")
	if err != nil {
		t.Fatal(err)
	}
	result := strings.ReplaceAll(sqlparser.String(expanded), "\"", "")
	for _, want := range []string{"json_extract", "as bucket_name", "aws.cloud_control.resources", "ap-southeast-2", "AWS::S3::Bucket", "prod%"} {
		if !strings.Contains(result, want) {
			t.Fatalf("expanded query lacks %q: %s", want, result)
		}
	}
	if strings.Contains(result, "us-east-1") || strings.Contains(result, "aws.pseudo_s3.buckets") {
		t.Fatal(result)
	}
	if sqlparser.String(outer) != original {
		t.Fatal("view expansion mutated the caller's AST")
	}
}

func TestViewVariantRequiresExplicitInboxInputs(t *testing.T) {
	predicate := `sqlDialect == "sqlite3" && requiredParams == [ "data__Identifier", region ]`
	for _, tc := range []struct {
		query string
		want  bool
	}{
		{"select * from v where data__Identifier = 'one' and region in ('east','west')", true},
		{"select * from v where region = 'east'", false},
		{"select * from v where data__Identifier = 'one' or region = 'east'", false},
		{"select * from v where other.data__Identifier = 'one' and region = 'east'", false},
	} {
		node := parseSelect(t, tc.query)
		got, err := matchViewPredicate(predicate, "sqlite3", viewLiteralBindings(node.Where, "v", "v"))
		if err != nil || got != tc.want {
			t.Fatalf("%s: match=%v err=%v", tc.query, got, err)
		}
	}
}

func TestViewFanoutOverridesLiteralInboxButRetainsDependency(t *testing.T) {
	outer := parseSelect(t, "select region, id from v where region in ('east','west')")
	view := parseSelect(t, "select detail.region, detail.id from p.s.listing listing join p.s.detail detail on detail.id = listing.id and detail.region = listing.region where listing.region in ('old','older') and listing.region = detail.region and listing.data__TypeName = 'kind'")
	expanded, err := inlineDocumentView(outer, view, "v")
	if err != nil {
		t.Fatal(err)
	}
	result := strings.ReplaceAll(sqlparser.String(expanded), "\"", "")
	for _, want := range []string{"detail.region = listing.region", "listing.region = detail.region", "listing.region in ('east', 'west')", "listing.data__TypeName = 'kind'"} {
		if !strings.Contains(result, want) {
			t.Fatalf("missing %s: %s", want, result)
		}
	}
	if strings.Contains(result, "older") {
		t.Fatal(result)
	}
}
