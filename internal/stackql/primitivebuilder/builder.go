package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type Builder interface {
	Build() error

	GetRoot() primitivegraph.PrimitiveNode

	GetTail() primitivegraph.PrimitiveNode
}
