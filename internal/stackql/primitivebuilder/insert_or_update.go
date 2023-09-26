package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type InsertOrUpdate struct {
	bldrInput builder_input.BuilderInput
	root      primitivegraph.PrimitiveNode
}

func NewInsertOrUpdate(
	bldrInput builder_input.BuilderInput,
) Builder {
	return &InsertOrUpdate{
		bldrInput: bldrInput,
	}
}

func (ss *InsertOrUpdate) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *InsertOrUpdate) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *InsertOrUpdate) Build() error {
	node, nodeExists := ss.bldrInput.GetParserNode()
	if !nodeExists {
		return fmt.Errorf("mutation executor: node does not exist")
	}
	mutableInput := ss.bldrInput.Clone()
	switch node := node.(type) {
	case *sqlparser.Insert:
		mutableInput.SetVerb("insert")
	case *sqlparser.Update:
		mutableInput.SetVerb("update")
	default:
		return fmt.Errorf("mutation executor: cannnot accomodate node of type '%T'", node)
	}
	var genericBldr Builder
	var genericBldrSetupErr error
	//nolint:nestif,gocritic // tactical
	if mutableInput.IsTargetPhysicalTable() {
		if mutableInput.GetVerb() == "insert" {
			genericBldr, genericBldrSetupErr = newInsertIntoPhysicalTable(
				mutableInput,
			)
		} else if mutableInput.GetVerb() == "update" {
			graphHolder, graphHolderExists := mutableInput.GetGraphHolder()
			handlerCtx, handlerCtxExists := mutableInput.GetHandlerContext()
			tcc, tccExists := mutableInput.GetTxnCtrlCtrs()
			if !graphHolderExists || !handlerCtxExists || !tccExists {
				return fmt.Errorf("mutation executor: cannot accomodate verb '%s'", mutableInput.GetVerb())
			}
			genericBldr = NewRawNativeExec(
				graphHolder,
				handlerCtx,
				tcc,
				handlerCtx.GetQuery(),
				mutableInput,
			)
		} else {
			return fmt.Errorf("mutation executor: cannot accomodate verb '%s'", mutableInput.GetVerb())
		}
	} else {
		genericBldr, genericBldrSetupErr = newGenericHTTPStreamInput(
			mutableInput,
		)
	}

	if genericBldrSetupErr != nil {
		return genericBldrSetupErr
	}
	genericBldrErr := genericBldr.Build()
	if genericBldrErr != nil {
		return genericBldrErr
	}
	ss.root = genericBldr.GetRoot()

	return nil
}
