package internaldto

var (
	_ PrepStmt = &standardPrepStmt{}
)

type PrepStmt interface {
	GetRawQuery() string
	GetArgs() []any
}

func NewPrepStmt(
	rawQuery string,
	args []any,
) PrepStmt {
	return &standardPrepStmt{
		rawQuery: rawQuery,
		args:     args,
	}
}

type standardPrepStmt struct {
	rawQuery string
	args     []any
}

func (s *standardPrepStmt) GetRawQuery() string {
	return s.rawQuery
}

func (s *standardPrepStmt) GetArgs() []any {
	return s.args
}
