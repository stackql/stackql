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

type TableParameterCoupling interface {
	ColumnKeyedDatastore
	getParameterMap() ParameterMap
	AbbreviateMap() (map[string]interface{}, error)
	Add(ColumnarReference, ParameterMetadata, ParamSourceType) error
	Clone() TableParameterCoupling
	GetOnCoupling() TableParameterCoupling
	GetNotOnCoupling() TableParameterCoupling
	GetAllParameters() []ParameterMapKeyVal
	Minus(rhs TableParameterCoupling) TableParameterCoupling
	ReconstituteConsumedParams(map[string]interface{}) (TableParameterCoupling, error)
}

type standardTableParameterCoupling struct {
	paramMap    ParameterMap
	colMappings map[string]ColumnarReference
}

func NewTableParameterCoupling() TableParameterCoupling {
	return &standardTableParameterCoupling{
		paramMap:    NewParameterMap(),
		colMappings: make(map[string]ColumnarReference),
	}
}

func (tpc *standardTableParameterCoupling) AndStringMap(rhs map[string]interface{}) ColumnKeyedDatastore {
	tpc.paramMap.AndStringMap(rhs)
	return tpc
}

func (tpc *standardTableParameterCoupling) DeleteStringMap(rhs map[string]interface{}) ColumnKeyedDatastore {
	tpc.paramMap.DeleteStringMap(rhs)
	return tpc
}

func (tpc *standardTableParameterCoupling) Clone() TableParameterCoupling {
	colMappings := make(map[string]ColumnarReference)
	for k, v := range tpc.colMappings {
		colMappings[k] = v
	}
	return &standardTableParameterCoupling{
		paramMap:    tpc.paramMap.Clone(),
		colMappings: colMappings,
	}
}

func (tpc *standardTableParameterCoupling) getParameterMap() ParameterMap {
	return tpc.paramMap
}

func (tpc *standardTableParameterCoupling) Minus(rhs TableParameterCoupling) TableParameterCoupling {
	difference := tpc.Clone()
	rhsParamMap := rhs.getParameterMap()
	for _, k := range rhsParamMap.GetAll() {
		difference.Delete(k.K)
	}
	return difference
}

func (tpc *standardTableParameterCoupling) GetAllParameters() []ParameterMapKeyVal {
	return tpc.paramMap.GetAll()
}

func (tpc *standardTableParameterCoupling) Add(
	col ColumnarReference,
	val ParameterMetadata,
	paramType ParamSourceType,
) error {
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

func (tpc *standardTableParameterCoupling) Delete(col ColumnarReference) bool {
	return tpc.paramMap.Delete(col)
}

func (tpc *standardTableParameterCoupling) Contains(col ColumnarReference) bool {
	return tpc.paramMap.Delete(col)
}

func (tpc *standardTableParameterCoupling) GetStringified() map[string]interface{} {
	return tpc.paramMap.GetAbbreviatedStringified()
}

func (tpc *standardTableParameterCoupling) DeleteByString(k string) bool {
	return tpc.paramMap.DeleteByString(k)
}

func (tpc *standardTableParameterCoupling) ContainsString(k string) bool {
	return tpc.paramMap.ContainsString(k)
}

func (tpc *standardTableParameterCoupling) AbbreviateMap() (map[string]interface{}, error) {
	return tpc.paramMap.GetAbbreviatedStringified(), nil
}

func (tpc *standardTableParameterCoupling) GetOnCoupling() TableParameterCoupling {
	retVal := NewTableParameterCoupling()
	m := tpc.paramMap.GetMap()
	for k, v := range m {
		if k.SourceType() == JoinOnParam {
			retVal.Add(k, v, k.SourceType()) //nolint:errcheck // TODO: review
		}
	}
	return retVal
}

func (tpc *standardTableParameterCoupling) GetNotOnCoupling() TableParameterCoupling {
	retVal := NewTableParameterCoupling()
	m := tpc.paramMap.GetMap()
	for k, v := range m {
		if k.SourceType() != JoinOnParam {
			retVal.Add(k, v, k.SourceType()) //nolint:errcheck // TODO: review
		}
	}
	return retVal
}

func (tpc *standardTableParameterCoupling) clone() TableParameterCoupling {
	retVal := NewTableParameterCoupling()
	m := tpc.paramMap.GetMap()
	for k, v := range m {
		retVal.Add(k, v, k.SourceType()) //nolint:errcheck // TODO: review
	}
	return retVal
}

//nolint:gocritic // TODO: review
func (tpc *standardTableParameterCoupling) ReconstituteConsumedParams(
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
