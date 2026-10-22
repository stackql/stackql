//nolint:testpackage // These tests exercise the unexported read-hint planner.
package intrinsic

import (
	"strings"
	"testing"

	"github.com/stackql/stackql/pkg/astformat"
)

func TestStagedOffsetAcknowledgmentIsAppliedOnce(t *testing.T) {
	withUnstable(t, true)
	node := parseSelect(t, "select id from stackql_unstable_p.s.items order by id limit 1 offset 2")
	staged, err := planStagedSelect(node, newDocTranslator("", sqliteDialect{}), astformat.SQLiteSelectExprsFormatter)
	if err != nil {
		t.Fatal(err)
	}
	acknowledgment, ok := staged.(interface{ renderAfterOffset(string, int) string })
	if !ok {
		t.Fatal("staged query cannot acknowledge a remote offset")
	}
	original := staged.renderOuter("items")
	if !strings.HasSuffix(original, "LIMIT 1 OFFSET 2") {
		t.Fatal(original)
	}
	if got := acknowledgment.renderAfterOffset("items", 2); strings.Contains(got, "OFFSET") || !strings.HasSuffix(got, "LIMIT 1") {
		t.Fatal(got)
	}
	if got := acknowledgment.renderAfterOffset("items", 1); !strings.HasSuffix(got, "LIMIT 1 OFFSET 1") {
		t.Fatal(got)
	}
	if got := staged.renderOuter("items"); got != original {
		t.Fatalf("acknowledgment mutated reusable plan: %s", got)
	}
}

func TestAggregateReadHintsDisableEarlyRowControls(t *testing.T) {
	node := parseSelect(t, "select count(*) from stackql_unstable_p.s.items limit 1 offset 2")
	hint := readHints(node)["items"]
	if !hint.DisableRowControls || !hint.Count || hint.Limit != 1 || hint.Offset != 2 {
		t.Fatalf("hints = %+v", hint)
	}
}
