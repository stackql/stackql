package parserutil

import (
	"fmt"

	"vitess.io/vitess/go/vt/sqlparser"
)

type ParamSourceType int

const (
	UnknownParam ParamSourceType = iota
	WhereParam
	JoinOnParam
)

type TableParameterCoupling interface {
	AbbreviateMap() (map[string]interface{}, error)
	Add(ColumnarReference, ParameterMetadata, ParamSourceType) error
	Delete(ColumnarReference) bool
	GetOnCoupling() TableParameterCoupling
	GetNotOnCoupling() TableParameterCoupling
	GetAllParameters() []ParameterMapKeyVal
	GetStringified() map[string]interface{}
	ReconstituteConsumedParams(map[string]interface{}) (TableParameterCoupling, error)
}

type StandardTableParameterCoupling struct {
	paramMap    ParameterMap
	colMappings map[string]ColumnarReference
}

func NewTableParameterCoupling() TableParameterCoupling {
	return &StandardTableParameterCoupling{
		paramMap:    NewParameterMap(),
		colMappings: make(map[string]ColumnarReference),
	}
}

func (tpc *StandardTableParameterCoupling) GetAllParameters() []ParameterMapKeyVal {
	return tpc.paramMap.GetAll()
}

func (tpc *StandardTableParameterCoupling) Add(col ColumnarReference, val ParameterMetadata, paramType ParamSourceType) error {
	colTyped, err := NewColumnarReference(col.Value(), paramType)
	if err != nil {
		return err
	}
	err = tpc.paramMap.Set(colTyped, val)
	if err != nil {
		return err
	}
	_, ok := tpc.colMappings[col.Name()]
	if ok {
		return fmt.Errorf("parameter '%s' already present", col.Name())
	}
	tpc.colMappings[col.Name()] = col
	return nil
}

func (tpc *StandardTableParameterCoupling) Delete(col ColumnarReference) bool {
	return tpc.paramMap.Delete(col)
}

func (tpc *StandardTableParameterCoupling) GetStringified() map[string]interface{} {
	return tpc.paramMap.GetAbbreviatedStringified()
}

func (tpc *StandardTableParameterCoupling) AbbreviateMap() (map[string]interface{}, error) {
	return tpc.paramMap.GetAbbreviatedStringified(), nil
}

func (tpc *StandardTableParameterCoupling) GetOnCoupling() TableParameterCoupling {
	retVal := NewTableParameterCoupling()
	m := tpc.paramMap.GetMap()
	for k, v := range m {
		if k.SourceType() == JoinOnParam {
			retVal.Add(k, v, k.SourceType())
		}
	}
	return retVal
}

func (tpc *StandardTableParameterCoupling) GetNotOnCoupling() TableParameterCoupling {
	retVal := NewTableParameterCoupling()
	m := tpc.paramMap.GetMap()
	for k, v := range m {
		if k.SourceType() != JoinOnParam {
			retVal.Add(k, v, k.SourceType())
		}
	}
	return retVal
}

func (tpc *StandardTableParameterCoupling) clone() TableParameterCoupling {
	retVal := NewTableParameterCoupling()
	m := tpc.paramMap.GetMap()
	for k, v := range m {
		retVal.Add(k, v, k.SourceType())
	}
	return retVal
}

func (tpc *StandardTableParameterCoupling) ReconstituteConsumedParams(
	returnedMap map[string]interface{},
) (TableParameterCoupling, error) {
	retVal := tpc.clone()
	for k, v := range returnedMap {
		key, ok := tpc.colMappings[k]
		if !ok || v == nil {
			return nil, fmt.Errorf("no reconstitution mapping for key = '%s'", k)
		}
		switch kv := key.Value().(type) {
		case *sqlparser.ColName:
			kv.Metadata = true
		}
		kv, ok := tpc.paramMap.GetByString(key.String())
		if !ok {
			return nil, fmt.Errorf("cannot process consumed params: attempt to locate non existing key")
		}
		for _, kt := range kv {
			ok = retVal.Delete(kt.K)
			if !ok {
				return nil, fmt.Errorf("cannot process consumed params: attempt to delete non existing key")
			}
		}

	}
	return retVal, nil
}
