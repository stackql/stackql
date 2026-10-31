// Package accessmanagement owns StackQL's persistent caller directory. Provider credentials
// and their storage are independent of these tables.
package accessmanagement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stackql-labs/omnisdk/public/principal"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

type directory struct{ db *sql.DB }

// NewDirectory uses the configured StackQL backend; it neither connects to a second database nor
// initializes schema implicitly. Missing schema or identities fail closed.
func NewDirectory(engine sqlengine.SQLEngine) (principal.Directory, error) {
	if engine == nil {
		return nil, errors.New("access directory requires the configured SQL engine")
	}
	db, err := engine.GetDB()
	if err != nil {
		return nil, err
	}
	return newDirectory(db)
}
func newDirectory(db *sql.DB) (principal.Directory, error) {
	if db == nil {
		return nil, errors.New("access directory requires a database")
	}
	return directory{db}, nil
}

// InitializeSchema is an explicit administration operation, not part of identity resolution.
// The statements are shared by SQLite and PostgreSQL; table names never come from caller input.
func InitializeSchema(ctx context.Context, engine sqlengine.SQLEngine) error {
	if engine == nil {
		return errors.New("access directory requires the configured SQL engine")
	}
	db, err := engine.GetDB()
	if err != nil {
		return err
	}
	return initializeSchema(ctx, db)
}
func initializeSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("access directory requires a database")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // Rollback after commit is harmless.
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS stackql_access_principals (id TEXT PRIMARY KEY, attributes TEXT NOT NULL DEFAULT '{}')`,
		`CREATE TABLE IF NOT EXISTS stackql_access_groups (id TEXT PRIMARY KEY)`,
		`CREATE TABLE IF NOT EXISTS stackql_access_identities (issuer TEXT NOT NULL, subject TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES stackql_access_principals(id), PRIMARY KEY (issuer, subject))`,
		`CREATE TABLE IF NOT EXISTS stackql_access_memberships (
 principal_id TEXT NOT NULL REFERENCES stackql_access_principals(id),
 group_id TEXT NOT NULL REFERENCES stackql_access_groups(id), PRIMARY KEY (principal_id, group_id))`,
	} {
		if _, err = tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("access directory schema: %w", err)
		}
	}
	return tx.Commit()
}
func (d directory) Resolve(ctx context.Context, i principal.Identity) (principal.Principal, error) {
	if i == nil || i.Issuer() == "" || i.Subject() == "" {
		return nil, errors.New("verified identity is required")
	}
	// A single SELECT supplies one consistent membership/attribute snapshot per session.
	rows, err := d.db.QueryContext(ctx, `SELECT p.id, p.attributes, m.group_id, g.id
 FROM stackql_access_identities i
 JOIN stackql_access_principals p ON p.id = i.principal_id
 LEFT JOIN stackql_access_memberships m ON m.principal_id = p.id
 LEFT JOIN stackql_access_groups g ON g.id = m.group_id
 WHERE i.issuer = $1 AND i.subject = $2 ORDER BY m.group_id`, i.Issuer(), i.Subject())
	if err != nil {
		return nil, fmt.Errorf("resolve access identity: %w", err)
	}
	defer rows.Close()
	var id, raw string
	var groups []string
	for rows.Next() {
		var nextID, nextRaw string
		var group, definedGroup sql.NullString
		if err = rows.Scan(&nextID, &nextRaw, &group, &definedGroup); err != nil {
			return nil, err
		}
		if id != "" && (id != nextID || raw != nextRaw) {
			return nil, errors.New("ambiguous access identity")
		}
		id, raw = nextID, nextRaw
		if group.Valid {
			if !definedGroup.Valid {
				return nil, errors.New("directory membership references an undefined group")
			}
			groups = append(groups, group.String)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.New("identity is not in the directory")
	}
	var attributes map[string]string
	if err = json.Unmarshal([]byte(raw), &attributes); err != nil {
		return nil, fmt.Errorf("invalid principal attributes: %w", err)
	}
	if attributes == nil {
		return nil, errors.New("principal attributes must be a JSON object")
	}
	return principal.NewPrincipal(id, groups, attributes)
}
