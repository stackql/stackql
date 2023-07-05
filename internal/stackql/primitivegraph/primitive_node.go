package primitivegraph

import (
	"github.com/stackql/stackql/internal/stackql/acid/operation"
)

var (
	_ PrimitiveNode = (*standardPrimitiveNode)(nil)
)

type PrimitiveNode interface {
	GetOperation() operation.Operation
	ID() int64
	IsDone() chan (bool)
	GetError() (error, bool)
	SetError(error)
	SetInputAlias(alias string, id int64) error
	SetIsDone(bool)
}

type standardPrimitiveNode struct {
	op     operation.Operation
	id     int64
	isDone chan bool
	err    error
}

func (pn *standardPrimitiveNode) ID() int64 {
	return pn.id
}

//nolint:revive // TODO: consider API change
func (pn *standardPrimitiveNode) GetError() (error, bool) {
	return pn.err, pn.err != nil
}

func (pn *standardPrimitiveNode) GetOperation() operation.Operation {
	return pn.op
}

func (pn *standardPrimitiveNode) IsDone() chan bool {
	return pn.isDone
}

func (pn *standardPrimitiveNode) SetInputAlias(alias string, id int64) error {
	op := pn.GetOperation()
	return op.SetInputAlias(alias, id)
}

func (pn *standardPrimitiveNode) SetIsDone(isDone bool) {
	pn.isDone <- isDone
}

func (pn *standardPrimitiveNode) SetError(err error) {
	pn.err = err
}
