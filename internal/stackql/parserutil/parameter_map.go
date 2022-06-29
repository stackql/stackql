package parserutil

import (
	"fmt"

	"vitess.io/vitess/go/vt/sqlparser"
)

// ParameterMap type is an abstraction
// for mapping a "Columnar Reference" (an abstract data type)
// to some supplied or inferred value.
type ParameterMap interface {
	iParameterMap()
	Set(ColumnarReference, ParameterMetadata) error
	Get(ColumnarReference) (ParameterMetadata, bool)
	Delete(ColumnarReference) bool
	GetAll() []ParameterMapKeyVal
	GetByString(string) ([]ParameterMapKeyVal, bool)
	GetMap() map[ColumnarReference]ParameterMetadata
	GetStringified() map[string]interface{}
	GetAbbreviatedStringified() map[string]interface{}
}

type StandardParameterMap struct {
	m map[ColumnarReference]ParameterMetadata
}

func NewParameterMap() ParameterMap {
	return StandardParameterMap{
		m: make(map[ColumnarReference]ParameterMetadata),
	}
}

func (pm StandardParameterMap) iParameterMap() {}

func (pm StandardParameterMap) GetByString(s string) ([]ParameterMapKeyVal, bool) {
	var retVal []ParameterMapKeyVal
	for k, v := range pm.m {
		if k.GetStringKey() == s {
			retVal = append(retVal, ParameterMapKeyVal{K: k, V: v})
		}
	}
	return retVal, true
}

func (pm StandardParameterMap) GetAll() []ParameterMapKeyVal {
	var retVal []ParameterMapKeyVal
	for k, v := range pm.m {
		retVal = append(retVal, ParameterMapKeyVal{K: k, V: v})
	}
	return retVal
}

func (pm StandardParameterMap) Delete(k ColumnarReference) bool {
	_, ok := pm.m[k]
	if ok {
		delete(pm.m, k)
		return true
	}
	return false
}

func (pm StandardParameterMap) GetMap() map[ColumnarReference]ParameterMetadata {
	return pm.m
}

func (pm StandardParameterMap) GetStringified() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range pm.m {
		rv[k.GetStringKey()] = v
	}
	return rv
}

func (pm StandardParameterMap) GetAbbreviatedStringified() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range pm.m {
		if k.SourceType() == JoinOnParam {
			// continue
		}
		switch kv := k.Value().(type) {
		case *sqlparser.ColName:
			rv[kv.Name.GetRawVal()] = v
		default:
			rv[k.GetStringKey()] = v
		}
	}
	return rv
}

func (pm StandardParameterMap) Set(k ColumnarReference, v ParameterMetadata) error {
	switch t := k.Value().(type) {
	case *sqlparser.ColName:
		pm.m[k] = v
	case *sqlparser.ColIdent:
		pm.m[k] = v
	default:
		return fmt.Errorf("parameter map cannot support key type = '%T'", t)
	}
	return nil
}

func (pm StandardParameterMap) Get(k ColumnarReference) (ParameterMetadata, bool) {
	switch k.Value().(type) {
	case *sqlparser.ColName:
		rv, ok := pm.m[k]
		return rv, ok
	case *sqlparser.ColIdent:
		rv, ok := pm.m[k]
		return rv, ok
	default:
		return nil, false
	}
}

func (tm StandardParameterMap) ToStringMap() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range tm.m {
		rv[k.GetStringKey()] = v
	}
	return rv
}
