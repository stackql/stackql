package parserutil

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// An abstract data type where the
// underlying datum should in some way
// represent a column or parameter.
// Purposes include parameterisation
// and request routing.
type ColumnarReference interface {
	iColumnarReference()
	GetStringKey() string
	Value() interface{}
	Alias() string
	String() string
	Name() string
	Abbreviate() (string, bool)
	SourceType() ParamSourceType
}

type StandardColumnarReference struct {
	k          interface{}
	sourceType ParamSourceType
}

func (cr StandardColumnarReference) Abbreviate() (string, bool) {
	switch kv := cr.Value().(type) {
	case *sqlparser.ColName:
		return kv.Name.GetRawVal(), true
	default:
		return cr.GetStringKey(), true
	}
}

func (cr StandardColumnarReference) SourceType() ParamSourceType {
	return cr.sourceType
}

func (cr StandardColumnarReference) Value() interface{} {
	return cr.k
}

func (cr StandardColumnarReference) Alias() string {
	switch t := cr.k.(type) {
	case *sqlparser.ColName:
		return t.Qualifier.GetRawVal()
	case *sqlparser.ColIdent:
		return t.GetRawVal()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (cr StandardColumnarReference) Name() string {
	switch t := cr.k.(type) {
	case *sqlparser.ColName:
		return t.Name.GetRawVal()
	case *sqlparser.ColIdent:
		return t.GetRawVal()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (cr StandardColumnarReference) String() string {
	switch t := cr.k.(type) {
	case *sqlparser.ColName:
		return t.GetRawVal()
	case *sqlparser.ColIdent:
		return t.GetRawVal()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (cr StandardColumnarReference) iColumnarReference() {}

func NewUnknownTypeColumnarReference(k interface{}) (ColumnarReference, error) {
	return newColumnarReference(k, UnknownParam)
}

func NewColumnarReference(k interface{}, sourceType ParamSourceType) (ColumnarReference, error) {
	return newColumnarReference(k, sourceType)
}

// Enforces supported underlying data invariant.
func newColumnarReference(k interface{}, sourceType ParamSourceType) (ColumnarReference, error) {
	switch k := k.(type) {
	case *sqlparser.ColName:
		return StandardColumnarReference{k: k, sourceType: sourceType}, nil
	case sqlparser.ColIdent:
		kp := &k
		return StandardColumnarReference{k: kp, sourceType: sourceType}, nil
	default:
		return nil, fmt.Errorf("cannot accomodate columnar reference for type = '%T'", k)
	}
}

func (cr StandardColumnarReference) GetStringKey() string {
	switch k := cr.k.(type) {
	case *sqlparser.ColName:
		return k.GetRawVal()
	case *sqlparser.ColIdent:
		return k.GetRawVal()
	default:
		return fmt.Sprintf("%v", k)
	}
}
