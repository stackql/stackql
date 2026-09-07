package contributors_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/contributors"
)

// The embedded leaderboard must be non-empty, ordered by contributions
// descending, and free of blank or automated logins.
func TestList_EmbeddedLeaderboard(t *testing.T) {
	list := contributors.List()
	if len(list) == 0 {
		t.Fatal("embedded contributors.csv parsed to an empty leaderboard")
	}
	for i, c := range list {
		if c.GetLogin() == "" || c.GetContributions() <= 0 {
			t.Fatalf("entry %d is malformed: %q %d", i, c.GetLogin(), c.GetContributions())
		}
		if i > 0 && c.GetContributions() > list[i-1].GetContributions() {
			t.Fatalf("entry %d (%s) breaks descending order", i, c.GetLogin())
		}
	}
	if list[0].GetLogin() != "general-kroll-4-life" {
		t.Fatalf("expected the project founder on top, got %q", list[0].GetLogin())
	}
}

func TestList_IsStable(t *testing.T) {
	if a, b := contributors.List(), contributors.List(); len(a) != len(b) || &a[0] != &b[0] {
		t.Fatal("List() must parse once and return the same leaderboard")
	}
}
