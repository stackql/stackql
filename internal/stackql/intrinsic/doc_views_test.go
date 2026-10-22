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
	result := sqlparser.String(expanded)
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
