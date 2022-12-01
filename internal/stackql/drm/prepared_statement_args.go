package drm

var (
	_ PreparedStatementArgs = &standardPreparedStatementArgs{}
)

type PreparedStatementArgs interface {
	GetArgs() []interface{}
	GetChild(int) PreparedStatementArgs
	GetChildren() map[int]PreparedStatementArgs
	GetQuery() string
	SetArgs([]interface{})
	SetChild(int, PreparedStatementArgs)
}

type standardPreparedStatementArgs struct {
	query    string
	args     []interface{}
	children map[int]PreparedStatementArgs
}

func NewPreparedStatementArgs(query string) PreparedStatementArgs {
	return &standardPreparedStatementArgs{
		query:    query,
		children: make(map[int]PreparedStatementArgs),
	}
}

func (ca *standardPreparedStatementArgs) GetChild(k int) PreparedStatementArgs {
	return ca.children[k]
}

func (ca *standardPreparedStatementArgs) GetChildren() map[int]PreparedStatementArgs {
	return ca.children
}

func (ca *standardPreparedStatementArgs) GetArgs() []interface{} {
	return ca.args
}

func (ca *standardPreparedStatementArgs) GetQuery() string {
	return ca.query
}

func (ca *standardPreparedStatementArgs) SetChild(i int, a PreparedStatementArgs) {
	ca.children[i] = a
}

func (ca *standardPreparedStatementArgs) SetArgs(args []interface{}) {
	ca.args = args
}
