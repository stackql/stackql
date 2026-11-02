package intrinsic //nolint:testpackage // Private provider-view rewriting is tested here.

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestProviderViewRetainsTenancyAlias(t *testing.T) {
	statement, err := sqlparser.Parse("select a.id from widgets.things.widgets a join other.things.widgets b on a.id = b.id join things.widgets c on c.id = a.id")
	if err != nil {
		t.Fatal(err)
	}
	scoped := scopeProviderView(statement.(*sqlparser.Select), "east", "widgets")
	providers := []string{}
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		if table, ok := node.(sqlparser.TableName); ok && !table.Qualifier.IsEmpty() {
			providers = append(providers, table.QualifierSecond.GetRawVal())
		}
		return true, nil
	}, scoped)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 3 || providers[0] != UnstablePrefix+"east" || providers[1] != "other" || providers[2] != UnstablePrefix+"east" {
		t.Fatalf("view lost its account scope: %v", providers)
	}
}

func TestAuthProviderNameNormalizesSQLProviderPrefix(t *testing.T) {
	for input, want := range map[string]string{"google": "google", "stackql_unstable_google": "google", "stackql_unstable_gcp": "google", "stackql_unstable_aws": "aws", "stackql_unstable_widgets": "widgets"} {
		if got := authProviderName(input); got != want {
			t.Fatalf("%s: got %s, want %s", input, got, want)
		}
	}
}
