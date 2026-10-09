package intrinsic //nolint:testpackage // tests the unexported dialect selection

import (
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/sqlfn"
	"github.com/stackql/any-sdk/pkg/dto"
)

// backendCtx is a queryContext whose only state is the SQL backend's configuration.
type backendCtx struct {
	queryContext
	raw string
}

func (c backendCtx) GetRuntimeContext() dto.RuntimeCtx {
	return dto.RuntimeCtx{SQLBackendCfgRaw: c.raw}
}

// The dialect is the backend's: the default and sqlite3 are SQLite, postgres is Postgres, and a
// backend with no catalogue is refused rather than given another's functions.
func TestBackendDialect(t *testing.T) {
	cases := map[string]sqlfn.Dialect{
		``:                          sqlfn.SQLite,
		`{"sqlDialect": "sqlite3"}`: sqlfn.SQLite,
		`{"dbEngine": "postgres_tcp", "sqlDialect": "postgres", "dsn": "postgres://u:p@h:5432/db"}`: sqlfn.Postgres,
	}
	for raw, want := range cases {
		got, err := backendDialect(backendCtx{raw: raw})
		if err != nil || got.catalogue() != want {
			t.Errorf("%s: got %v, %v; want %q", raw, got, err, want)
		}
	}
	if _, err := backendDialect(backendCtx{raw: `{"sqlDialect": "snowflake"}`}); err == nil {
		t.Error("snowflake: want a refusal, got none")
	}
}
