package intrinsic //nolint:testpackage // Verifies host metadata and execution remain behind read handles.

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
)

type externalTestContext struct {
	*stagingTestCtx
	source sql_datasource.SQLDataSource
}

func (c *externalTestContext) GetSQLDataSource(name string) (sql_datasource.SQLDataSource, bool) {
	return c.source, name == "pgi"
}

type externalTestSource struct {
	sql_datasource.SQLDataSource
	db       *sql.DB
	metadata atomic.Int32
	data     atomic.Int32
}

func (s *externalTestSource) QueryContext(ctx context.Context, statement string, args ...any) (*sql.Rows, error) {
	if strings.HasSuffix(statement, "WHERE 1=0") {
		s.metadata.Add(1)
		if _, bounded := ctx.Deadline(); !bounded {
			return nil, errors.New("unbounded metadata read")
		}
	} else {
		s.data.Add(1)
	}
	return s.db.QueryContext(ctx, statement, args...)
}

func TestConfiguredExternalSourceBecomesLazyOmniRelation(t *testing.T) {
	base := newStagingTestCtx(t, "")
	db, err := base.GetSQLEngine().GetDB()
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"CREATE TABLE external_rows (id INTEGER, name TEXT)", "INSERT INTO external_rows VALUES (1,'a'),(2,'b')"} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	source := &externalTestSource{db: db}
	ctx := &externalTestContext{stagingTestCtx: base, source: source}
	node := parseSelect(t, "select name from pgi.main.external_rows where id = 2")
	if execution, owned := storedSelectFunc(ctx, node, ""); !owned || execution == nil {
		t.Fatal("configured SQL source fell through to provider discovery")
	}
	prepared, relations, err := prepareStoredRelations(ctx, node)
	if err != nil {
		t.Fatal(err)
	}
	if !hasExternalRelations(relations) || source.metadata.Load() == 0 || source.data.Load() != 0 {
		t.Fatalf("relations=%d metadata=%d data=%d", len(relations), source.metadata.Load(), source.data.Load())
	}
	translated, err := newDocTranslator("", sqliteDialect{}).selectQuery(prepared)
	if err != nil {
		t.Fatal(err)
	}
	translated = withRelations(translated, relations)
	rows, _, err := openDocQueryIn(ctx, translated, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() || rows.Row()["name"] != "b" {
		t.Fatalf("row=%v err=%v", rows.Row(), rows.Err())
	}
	if rows.Next() || rows.Err() != nil || source.data.Load() != 1 {
		t.Fatalf("continuation=%v data=%d", rows.Err(), source.data.Load())
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err = openDocQueryIn(ctx, translated, canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read=%v", err)
	}
	if source.data.Load() != 1 {
		t.Fatal("canceled plan queried the data source")
	}
}
