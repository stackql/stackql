package intrinsic

import (
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql-labs/omnisdk/pkg/sqlfn"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
)

// sqlDialect is the SQL of the session's backend as the translator writes it: the omnisdk function
// catalogue a query's calls resolve in, and how SQL that is an operator in the parse is spelled as a
// call in that catalogue.
type sqlDialect interface {
	// catalogue is the omnisdk function catalogue of the backend.
	catalogue() sqlfn.Dialect
	// like is value LIKE pattern [ESCAPE escape]; escape is nil without the clause.
	like(value, pattern, escape query.Expr) query.Expr
}

// sqliteDialect is the embedded SQLite's: its function like(pattern, value[, escape]).
type sqliteDialect struct{}

func (sqliteDialect) catalogue() sqlfn.Dialect { return sqlfn.SQLite }

func (sqliteDialect) like(value, pattern, escape query.Expr) query.Expr {
	args := []query.Expr{pattern, value}
	if escape != nil {
		args = append(args, escape)
	}
	return query.NewCall("like", args...)
}

// postgresDialect is Postgres's: like(value, pattern), an ESCAPE clause rewriting the pattern
// through like_escape(pattern, escape) as Postgres's parser does.
type postgresDialect struct{}

func (postgresDialect) catalogue() sqlfn.Dialect { return sqlfn.Postgres }

func (postgresDialect) like(value, pattern, escape query.Expr) query.Expr {
	if escape != nil {
		pattern = query.NewCall("like_escape", pattern, escape)
	}
	return query.NewCall("like", value, pattern)
}

// backendDialect is the dialect of the session's backend, read from its configuration; a backend
// omnisdk has no catalogue for is refused.
func backendDialect(ctx queryContext) (sqlDialect, error) {
	cfg, err := dto.GetSQLBackendCfg(ctx.GetRuntimeContext().SQLBackendCfgRaw)
	if err != nil {
		return nil, err
	}
	switch cfg.GetSQLDialect() {
	case constants.SQLDialectSQLite3:
		return sqliteDialect{}, nil
	case constants.SQLDialectPostgres:
		return postgresDialect{}, nil
	}
	return nil, fmt.Errorf("SQL backend %q has no omnisdk function catalogue (want %q or %q)",
		cfg.GetSQLDialect(), constants.SQLDialectSQLite3, constants.SQLDialectPostgres)
}
