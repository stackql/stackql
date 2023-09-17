package drm

import "github.com/stackql/stackql/internal/stackql/typing"

var (
	_ PreparedStatementParameterized = &standardPreparedStatementParameterized{}
)

type PreparedStatementParameterized interface {
	AddChild(int, PreparedStatementParameterized)
	GetArgs() map[string]interface{}
	GetChildren() map[int]PreparedStatementParameterized
	GetCtx() PreparedStatementCtx
	GetRequestEncoding() string
	IsControlArgsRequired() bool
	WithRequestEncoding(string) PreparedStatementParameterized
	GetNonControlColumns() []typing.ColumnMetadata
}

type standardPreparedStatementParameterized struct {
	ctx                 PreparedStatementCtx
	args                map[string]interface{}
	controlArgsRequired bool
	requestEncoding     string
	children            map[int]PreparedStatementParameterized
}

func (ps *standardPreparedStatementParameterized) WithRequestEncoding(reqEnc string) PreparedStatementParameterized {
	ps.requestEncoding = reqEnc
	return ps
}

func (ps *standardPreparedStatementParameterized) GetRequestEncoding() string {
	return ps.requestEncoding
}

func (ps *standardPreparedStatementParameterized) GetNonControlColumns() []typing.ColumnMetadata {
	return ps.ctx.GetNonControlColumns()
}

func (ps *standardPreparedStatementParameterized) IsControlArgsRequired() bool {
	return ps.controlArgsRequired
}

func (ps *standardPreparedStatementParameterized) GetArgs() map[string]interface{} {
	return ps.args
}

func (ps *standardPreparedStatementParameterized) AddChild(key int, val PreparedStatementParameterized) {
	ps.children[key] = val
}

func (ps *standardPreparedStatementParameterized) GetChildren() map[int]PreparedStatementParameterized {
	return ps.children
}

func (ps *standardPreparedStatementParameterized) GetCtx() PreparedStatementCtx {
	return ps.ctx
}

func NewPreparedStatementParameterized(
	ctx PreparedStatementCtx,
	args map[string]interface{},
	controlArgsRequired bool,
) PreparedStatementParameterized {
	children := make(map[int]PreparedStatementParameterized)
	for i, ctx := range ctx.GetIndirectContexts() {
		children[i] = NewPreparedStatementParameterized(ctx, nil, true)
	}
	return &standardPreparedStatementParameterized{
		ctx:                 ctx,
		args:                args,
		controlArgsRequired: controlArgsRequired,
		children:            children,
	}
}
