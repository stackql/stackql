package parserutil

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type ParamSourceType int

const (
	UnknownParam ParamSourceType = iota
	WhereParam
	JoinOnParam
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

type standardColumnarReference struct {
	k          interface{}
	sourceType ParamSourceType
}

func (cr standardColumnarReference) Abbreviate() (string, bool) {
	switch kv := cr.Value().(type) {
	case *sqlparser.ColName:
		return kv.Name.GetRawVal(), true
	default:
		return cr.GetStringKey(), true
	}
}

func (cr standardColumnarReference) SourceType() ParamSourceType {
	return cr.sourceType
}

func (cr standardColumnarReference) Value() interface{} {
	return cr.k
}

func (cr standardColumnarReference) Alias() string {
	switch t := cr.k.(type) {
	case *sqlparser.ColName:
		return t.Qualifier.GetRawVal()
	case *sqlparser.ColIdent:
		return t.GetRawVal()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (cr standardColumnarReference) Name() string {
	switch t := cr.k.(type) {
	case *sqlparser.ColName:
		return t.Name.GetRawVal()
	case *sqlparser.ColIdent:
		return t.GetRawVal()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (cr standardColumnarReference) String() string {
	switch t := cr.k.(type) {
	case *sqlparser.ColName:
		return t.GetRawVal()
	case *sqlparser.ColIdent:
		return t.GetRawVal()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (cr standardColumnarReference) iColumnarReference() {}

func NewUnknownTypeColumnarReference(k interface{}) (ColumnarReference, error) {
	return newColumnarReference(k, UnknownParam)
}

// Enforces supported underlying data invariant.
func newColumnarReference(k interface{}, sourceType ParamSourceType) (ColumnarReference, error) {
	switch k := k.(type) {
	case *sqlparser.ColName:
		return standardColumnarReference{k: k, sourceType: sourceType}, nil
	case sqlparser.ColIdent:
		kp := &k
		return standardColumnarReference{k: kp, sourceType: sourceType}, nil
	default:
		return nil, fmt.Errorf("cannot accomodate columnar reference for type = '%T'", k)
	}
}

func (cr standardColumnarReference) GetStringKey() string {
	switch k := cr.k.(type) {
	case *sqlparser.ColName:
		return k.GetRawVal()
	case *sqlparser.ColIdent:
		return k.GetRawVal()
	default:
		return fmt.Sprintf("%v", k)
	}
}
