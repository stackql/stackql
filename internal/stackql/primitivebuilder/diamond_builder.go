package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

type DiamondBuilder struct {
	SubTreeBuilder
	parentBuilder            Builder
	graphHolder              primitivegraph.PrimitiveGraphHolder
	root, tailRoot, tailTail primitivegraph.PrimitiveNode
	sqlSystem                sql_system.SQLSystem
	shouldCollectGarbage     bool
}

func NewDiamondBuilder(
	parent Builder,
	children []Builder,
	graphHolder primitivegraph.PrimitiveGraphHolder,
	sqlSystem sql_system.SQLSystem,
	shouldCollectGarbage bool,
) Builder {
	return &DiamondBuilder{
		SubTreeBuilder:       SubTreeBuilder{children: children},
		parentBuilder:        parent,
		graphHolder:          graphHolder,
		sqlSystem:            sqlSystem,
		shouldCollectGarbage: shouldCollectGarbage,
	}
}

func (db *DiamondBuilder) Build() error {
	for _, child := range db.children {
		err := child.Build()
		if err != nil {
			return err
		}
	}
	db.root = db.graphHolder.CreatePrimitiveNode(
		primitive.NewPassThroughPrimitive(
			db.sqlSystem,
			db.graphHolder.GetTxnControlCounterSlice(),
			false,
		),
	)
	if db.parentBuilder != nil {
		err := db.parentBuilder.Build()
		if err != nil {
			return err
		}
		db.tailRoot = db.parentBuilder.GetRoot()
		db.tailTail = db.parentBuilder.GetTail()
	} else {
		db.tailRoot = db.graphHolder.CreatePrimitiveNode(
			primitive.NewPassThroughPrimitive(
				db.sqlSystem,
				db.graphHolder.GetTxnControlCounterSlice(),
				db.shouldCollectGarbage,
			),
		)
		db.tailTail = db.tailRoot
	}
	for _, child := range db.children {
		root := child.GetRoot()
		tail := child.GetTail()
		db.graphHolder.NewDependency(db.root, root, 1.0)
		db.graphHolder.NewDependency(tail, db.tailRoot, 1.0)
		// db.tail.Primitive = child.GetTail().Primitive
	}
	return nil
}

func (db *DiamondBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return db.root
}

func (db *DiamondBuilder) GetTail() primitivegraph.PrimitiveNode {
	return db.tailTail
}
