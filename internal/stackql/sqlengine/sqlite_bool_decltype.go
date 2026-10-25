package sqlengine

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
)

// boolDecltype is the declared column type that triggers boolean conversion;
// the retired cgo driver (mattn/go-sqlite3) compared the lowercased decltype
// against exactly "boolean".
const boolDecltype = "boolean"

var (
	_ driver.Driver = (*boolDecltypeDriver)(nil)
	_ moderncConn   = (*boolDecltypeConn)(nil)
	_ moderncStmt   = (*boolDecltypeStmt)(nil)
	_ moderncRows   = (*boolDecltypeRows)(nil)
)

// moderncConn is the optional-interface surface of the modernc.org/sqlite
// connection that database/sql consults; the wrapper delegates every one so
// pooling and cancellation behavior is unchanged.
type moderncConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.ExecerContext
	driver.QueryerContext
	driver.Pinger
	driver.SessionResetter
	driver.Validator
}

// moderncStmt is the optional-interface surface of the modernc.org/sqlite
// prepared statement.
type moderncStmt interface {
	driver.Stmt
	driver.StmtExecContext
	driver.StmtQueryContext
}

// moderncRows is the optional-interface surface of the modernc.org/sqlite
// rows iterator.
type moderncRows interface {
	driver.Rows
	driver.RowsColumnTypeDatabaseTypeName
	driver.RowsColumnTypeLength
	driver.RowsColumnTypeNullable
	driver.RowsColumnTypePrecisionScale
	driver.RowsColumnTypeScanType
}

// boolDecltypeDriver restores mattn/go-sqlite3 parity for declared-boolean
// columns: INTEGER values in a column whose declared type is boolean
// (case-insensitive) surface as Go bool via the retired driver's val > 0
// rule. modernc.org/sqlite performs decltype-based conversion for timestamps
// but not booleans, so such columns would otherwise surface as 0/1.
type boolDecltypeDriver struct {
	inner driver.Driver
}

func newBoolDecltypeDriver(inner driver.Driver) driver.Driver {
	return &boolDecltypeDriver{inner: inner}
}

func (d *boolDecltypeDriver) Open(name string) (driver.Conn, error) {
	c, err := d.inner.Open(name)
	if err != nil {
		return nil, err
	}
	mc, ok := c.(moderncConn)
	if !ok {
		_ = c.Close()
		return nil, fmt.Errorf("sqlite embedded: connection %T lacks the expected modernc.org/sqlite interfaces", c)
	}
	return &boolDecltypeConn{inner: mc}, nil
}

type boolDecltypeConn struct {
	inner moderncConn
}

func (c *boolDecltypeConn) Prepare(query string) (driver.Stmt, error) {
	return wrapBoolDecltypeStmt(c.inner.Prepare(query))
}

func (c *boolDecltypeConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return wrapBoolDecltypeStmt(c.inner.PrepareContext(ctx, query))
}

func (c *boolDecltypeConn) Close() error { return c.inner.Close() }

// Begin satisfies driver.Conn by delegating to BeginTx, so the deprecated
// inner method is never called; database/sql itself always prefers BeginTx.
func (c *boolDecltypeConn) Begin() (driver.Tx, error) {
	return c.inner.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *boolDecltypeConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.inner.BeginTx(ctx, opts)
}

func (c *boolDecltypeConn) ExecContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Result, error) {
	return c.inner.ExecContext(ctx, query, args)
}

func (c *boolDecltypeConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	return wrapBoolDecltypeRows(c.inner.QueryContext(ctx, query, args))
}

func (c *boolDecltypeConn) Ping(ctx context.Context) error { return c.inner.Ping(ctx) }

func (c *boolDecltypeConn) ResetSession(ctx context.Context) error { return c.inner.ResetSession(ctx) }

func (c *boolDecltypeConn) IsValid() bool { return c.inner.IsValid() }

func wrapBoolDecltypeStmt(s driver.Stmt, err error) (driver.Stmt, error) {
	if err != nil || s == nil {
		return s, err
	}
	ms, ok := s.(moderncStmt)
	if !ok {
		_ = s.Close()
		return nil, fmt.Errorf("sqlite embedded: statement %T lacks the expected modernc.org/sqlite interfaces", s)
	}
	return &boolDecltypeStmt{inner: ms}, nil
}

type boolDecltypeStmt struct {
	inner moderncStmt
}

func (s *boolDecltypeStmt) Close() error { return s.inner.Close() }

func (s *boolDecltypeStmt) NumInput() int { return s.inner.NumInput() }

// Exec satisfies driver.Stmt by delegating to ExecContext, so the deprecated
// inner method is never called.
func (s *boolDecltypeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.inner.ExecContext(context.Background(), valuesToNamedValues(args))
}

// Query satisfies driver.Stmt by delegating to QueryContext, so the
// legacy inner method is never called.
func (s *boolDecltypeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return wrapBoolDecltypeRows(s.inner.QueryContext(context.Background(), valuesToNamedValues(args)))
}

func (s *boolDecltypeStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.inner.ExecContext(ctx, args)
}

func (s *boolDecltypeStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return wrapBoolDecltypeRows(s.inner.QueryContext(ctx, args))
}

func valuesToNamedValues(args []driver.Value) []driver.NamedValue {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return named
}

// wrapBoolDecltypeRows wraps a result set that declares at least one boolean
// column; result sets without one pass through untouched.
func wrapBoolDecltypeRows(r driver.Rows, err error) (driver.Rows, error) {
	if err != nil || r == nil {
		return r, err
	}
	mr, ok := r.(moderncRows)
	if !ok {
		_ = r.Close()
		return nil, fmt.Errorf("sqlite embedded: rows %T lack the expected modernc.org/sqlite interfaces", r)
	}
	var boolCols []bool
	for i := range mr.Columns() {
		if strings.EqualFold(mr.ColumnTypeDatabaseTypeName(i), boolDecltype) {
			if boolCols == nil {
				boolCols = make([]bool, len(mr.Columns()))
			}
			boolCols[i] = true
		}
	}
	if boolCols == nil {
		return mr, nil
	}
	return &boolDecltypeRows{inner: mr, boolCols: boolCols}, nil
}

type boolDecltypeRows struct {
	inner moderncRows
	// boolCols marks the columns whose declared type is boolean.
	boolCols []bool
}

func (r *boolDecltypeRows) Columns() []string { return r.inner.Columns() }

func (r *boolDecltypeRows) Close() error { return r.inner.Close() }

// Next converts INTEGER values in declared-boolean columns to Go bool with
// the retired driver's val > 0 rule; NULL and every other storage class or
// column pass through unchanged.
func (r *boolDecltypeRows) Next(dest []driver.Value) error {
	if err := r.inner.Next(dest); err != nil {
		return err
	}
	for i, isBool := range r.boolCols {
		if !isBool || i >= len(dest) {
			continue
		}
		if v, ok := dest[i].(int64); ok {
			dest[i] = v > 0
		}
	}
	return nil
}

func (r *boolDecltypeRows) ColumnTypeDatabaseTypeName(index int) string {
	return r.inner.ColumnTypeDatabaseTypeName(index)
}

func (r *boolDecltypeRows) ColumnTypeLength(index int) (int64, bool) {
	return r.inner.ColumnTypeLength(index)
}

func (r *boolDecltypeRows) ColumnTypeNullable(index int) (bool, bool) {
	return r.inner.ColumnTypeNullable(index)
}

func (r *boolDecltypeRows) ColumnTypePrecisionScale(index int) (int64, int64, bool) {
	return r.inner.ColumnTypePrecisionScale(index)
}

func (r *boolDecltypeRows) ColumnTypeScanType(index int) reflect.Type {
	return r.inner.ColumnTypeScanType(index)
}
