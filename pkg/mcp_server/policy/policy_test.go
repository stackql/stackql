package policy_test

import (
	"testing"

	"github.com/stackql/stackql/pkg/mcp_server/policy"
)

func TestClassifyQuery(t *testing.T) {
	cases := []struct {
		sql  string
		want policy.QueryClass
	}{
		{"", policy.QueryClassUnknown},
		{"   ", policy.QueryClassUnknown},
		{"select * from t", policy.QueryClassSelect},
		{"  SELECT 1", policy.QueryClassSelect},
		{"Show methods in x.y.z;", policy.QueryClassSelect},
		{"describe x.y", policy.QueryClassSelect},
		{"EXPLAIN select 1", policy.QueryClassSelect},
		{"insert into t values (1)", policy.QueryClassMutationCreate},
		{"UPDATE t SET a=1", policy.QueryClassMutationCreate},
		{"replace into t (a) values (1)", policy.QueryClassMutationCreate},
		{"merge into t using s on (...) when matched ...", policy.QueryClassMutationCreate},
		{"UPSERT into t ...", policy.QueryClassMutationCreate},
		{"delete from t", policy.QueryClassMutationDelete},
		{"EXEC google.compute.instances.start @project='x'", policy.QueryClassLifecycle},
		{"   exec aws.foo.bar @a='1'", policy.QueryClassLifecycle},
		{"vacuum foo", policy.QueryClassUnknown},
		{"-- comment then select", policy.QueryClassUnknown}, // first token is the comment marker
	}
	for _, c := range cases {
		t.Run(c.sql, func(t *testing.T) {
			got := policy.ClassifyQuery(c.sql)
			if got != c.want {
				t.Errorf("ClassifyQuery(%q) = %v, want %v", c.sql, got, c.want)
			}
		})
	}
}

func TestGateDecision_FullAccessAllowsEverything(t *testing.T) {
	for _, cls := range []policy.QueryClass{
		policy.QueryClassSelect, policy.QueryClassMutationCreate,
		policy.QueryClassMutationDelete, policy.QueryClassLifecycle,
	} {
		d, _ := policy.GateDecision(policy.ModeFullAccess, cls)
		if d != policy.DecisionAllow {
			t.Errorf("full_access should allow %v, got %v", cls, d)
		}
	}
}

func TestGateDecision_ReadOnlyRefusesMutations(t *testing.T) {
	d, _ := policy.GateDecision(policy.ModeReadOnly, policy.QueryClassSelect)
	if d != policy.DecisionAllow {
		t.Errorf("read_only should allow select, got %v", d)
	}
	for _, cls := range []policy.QueryClass{
		policy.QueryClassMutationCreate, policy.QueryClassMutationDelete, policy.QueryClassLifecycle,
	} {
		d, reason := policy.GateDecision(policy.ModeReadOnly, cls)
		if d != policy.DecisionRefuseImmediate {
			t.Errorf("read_only should refuse %v, got %v", cls, d)
		}
		if reason == "" || !contains(reason, "read_only") {
			t.Errorf("reason should mention read_only, got %q", reason)
		}
	}
}

func TestGateDecision_DeleteSafeAllowsCreateRefusesDeleteAndLifecycle(t *testing.T) {
	for _, cls := range []policy.QueryClass{policy.QueryClassSelect, policy.QueryClassMutationCreate} {
		d, _ := policy.GateDecision(policy.ModeDeleteSafe, cls)
		if d != policy.DecisionAllow {
			t.Errorf("delete_safe should allow %v, got %v", cls, d)
		}
	}
	for _, cls := range []policy.QueryClass{policy.QueryClassMutationDelete, policy.QueryClassLifecycle} {
		d, reason := policy.GateDecision(policy.ModeDeleteSafe, cls)
		if d != policy.DecisionNeedsApproval {
			t.Errorf("delete_safe should need approval for %v, got %v", cls, d)
		}
		if !contains(reason, "delete_safe") {
			t.Errorf("reason should mention delete_safe, got %q", reason)
		}
	}
}

func TestGateDecision_SafeNeedsApprovalForAllMutations(t *testing.T) {
	d, _ := policy.GateDecision(policy.ModeSafe, policy.QueryClassSelect)
	if d != policy.DecisionAllow {
		t.Errorf("safe should allow select")
	}
	for _, cls := range []policy.QueryClass{
		policy.QueryClassMutationCreate, policy.QueryClassMutationDelete, policy.QueryClassLifecycle,
	} {
		d, reason := policy.GateDecision(policy.ModeSafe, cls)
		if d != policy.DecisionNeedsApproval {
			t.Errorf("safe should need approval for %v, got %v", cls, d)
		}
		if !contains(reason, "safe") {
			t.Errorf("reason should mention safe, got %q", reason)
		}
	}
}

func TestGateDecision_EmptyAndUnknownDefaultToSafe(t *testing.T) {
	for _, mode := range []string{"", "bogus"} {
		d, _ := policy.GateDecision(mode, policy.QueryClassSelect)
		if d != policy.DecisionAllow {
			t.Errorf("mode %q + select should allow, got %v", mode, d)
		}
		d, reason := policy.GateDecision(mode, policy.QueryClassMutationCreate)
		if d != policy.DecisionNeedsApproval {
			t.Errorf("mode %q + create should need approval, got %v", mode, d)
		}
		if !contains(reason, "safe") {
			t.Errorf("reason should fall back to safe, got %q", reason)
		}
	}
}

func TestIsLegalMode(t *testing.T) {
	for _, m := range []string{"", "read_only", "safe", "delete_safe", "full_access"} {
		if !policy.IsLegalMode(m) {
			t.Errorf("%q should be legal", m)
		}
	}
	for _, m := range []string{"readonly", "Safe", "FULL_ACCESS", "bogus"} {
		if policy.IsLegalMode(m) {
			t.Errorf("%q should not be legal", m)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
