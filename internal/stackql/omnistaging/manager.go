package omnistaging

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Manager owns query identities and their staged tables.
//
// Release is idempotent: it drops every table owned by the query and its
// descendants, then forgets them. On error the registry is retained so the
// release can be retried.
type Manager interface {
	Begin(ctx context.Context) (QueryID, error)
	BeginChild(ctx context.Context, parent QueryID) (QueryID, error)
	Parent(id QueryID) (QueryID, bool)
	Children(id QueryID) []QueryID
	TableName(id QueryID) string
	// Stage creates the query table and writes every batch from source,
	// returning the number of rows written.
	Stage(ctx context.Context, id QueryID, columns []Column, source BatchSource) (int64, error)
	// Owned returns the IDs with staged tables in the subtree rooted at id.
	Owned(id QueryID) []QueryID
	Release(ctx context.Context, id QueryID) error
}

// writer renders and executes staging DDL and batch inserts.
type writer interface {
	Create(ctx context.Context, id QueryID, columns []Column) error
	Insert(ctx context.Context, id QueryID, columns []Column, rows [][]any) error
	Drop(ctx context.Context, id QueryID) error
}

type sqlWriter struct {
	db      Executor
	dialect Dialect
}

func newSQLWriter(db Executor, dialect Dialect) writer {
	return &sqlWriter{db: db, dialect: dialect}
}

func (w *sqlWriter) Create(ctx context.Context, id QueryID, columns []Column) error {
	stmt, err := w.dialect.CreateTableStatement(id, columns)
	if err != nil {
		return err
	}
	if _, err = w.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("omnistaging: cannot create %s: %w", w.dialect.TableName(id), err)
	}
	return nil
}

// Insert writes rows in chunks bounded by the dialect bind parameter limit.
func (w *sqlWriter) Insert(ctx context.Context, id QueryID, columns []Column, rows [][]any) error {
	width := len(columns)
	if width == 0 {
		return fmt.Errorf("omnistaging: insert into %s requires columns", w.dialect.TableName(id))
	}
	chunkSize := w.dialect.MaxBindParameters() / width
	for start := 0; start < len(rows); start += chunkSize {
		end := min(start+chunkSize, len(rows))
		chunk := rows[start:end]
		args := make([]any, 0, len(chunk)*width)
		for _, row := range chunk {
			if len(row) != width {
				return fmt.Errorf(
					"omnistaging: row width %d does not match %d columns of %s",
					len(row), width, w.dialect.TableName(id),
				)
			}
			args = append(args, row...)
		}
		stmt, err := w.dialect.InsertStatement(id, columns, len(chunk))
		if err != nil {
			return err
		}
		if _, err = w.db.ExecContext(ctx, stmt, args...); err != nil {
			return fmt.Errorf("omnistaging: cannot insert into %s: %w", w.dialect.TableName(id), err)
		}
	}
	return nil
}

func (w *sqlWriter) Drop(ctx context.Context, id QueryID) error {
	if _, err := w.db.ExecContext(ctx, w.dialect.DropTableStatement(id)); err != nil {
		return fmt.Errorf("omnistaging: cannot drop %s: %w", w.dialect.TableName(id), err)
	}
	return nil
}

type standardManager struct {
	dialect  Dialect
	counter  counter
	registry registry
	writer   writer
}

// NewManager runs the dialect setup statements and returns a manager whose
// query IDs are allocated by the backend.
func NewManager(ctx context.Context, db Executor, dialect Dialect) (Manager, error) {
	for _, stmt := range dialect.SetupStatements() {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return nil, fmt.Errorf("omnistaging: setup failed: %w", err)
		}
	}
	return &standardManager{
		dialect:  dialect,
		counter:  newSQLCounter(db, dialect),
		registry: newRegistry(),
		writer:   newSQLWriter(db, dialect),
	}, nil
}

func (m *standardManager) Begin(ctx context.Context) (QueryID, error) {
	return m.begin(ctx, nil)
}

func (m *standardManager) BeginChild(ctx context.Context, parent QueryID) (QueryID, error) {
	if parent == nil {
		return nil, fmt.Errorf("omnistaging: child query requires a parent")
	}
	return m.begin(ctx, parent)
}

func (m *standardManager) begin(ctx context.Context, parent QueryID) (QueryID, error) {
	id, err := m.counter.Next(ctx)
	if err != nil {
		return nil, err
	}
	if err = m.registry.Add(id, parent); err != nil {
		return nil, err
	}
	return id, nil
}

func (m *standardManager) Parent(id QueryID) (QueryID, bool) {
	return m.registry.Parent(id)
}

func (m *standardManager) Children(id QueryID) []QueryID {
	return m.registry.Children(id)
}

func (m *standardManager) TableName(id QueryID) string {
	return m.dialect.TableName(id)
}

func (m *standardManager) Stage(
	ctx context.Context,
	id QueryID,
	columns []Column,
	source BatchSource,
) (int64, error) {
	if !m.registry.Contains(id) {
		return 0, fmt.Errorf("omnistaging: query %s not registered", id)
	}
	// Ownership is recorded before DDL so a partial failure is still released.
	if err := m.registry.MarkStaged(id); err != nil {
		return 0, err
	}
	if err := m.writer.Create(ctx, id, columns); err != nil {
		return 0, err
	}
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		rows, err := source.Next(ctx)
		if errors.Is(err, io.EOF) {
			return written, nil
		}
		if err != nil {
			return written, fmt.Errorf("omnistaging: batch source failed for query %s: %w", id, err)
		}
		if err = m.writer.Insert(ctx, id, columns, rows); err != nil {
			return written, err
		}
		written += int64(len(rows))
	}
}

func (m *standardManager) Owned(id QueryID) []QueryID {
	var owned []QueryID
	for _, member := range m.registry.Subtree(id) {
		if m.registry.IsStaged(member) {
			owned = append(owned, member)
		}
	}
	return owned
}

func (m *standardManager) Release(ctx context.Context, id QueryID) error {
	subtree := m.registry.Subtree(id)
	for _, member := range subtree {
		if !m.registry.IsStaged(member) {
			continue
		}
		if err := m.writer.Drop(ctx, member); err != nil {
			return err
		}
	}
	m.registry.Remove(subtree)
	return nil
}
