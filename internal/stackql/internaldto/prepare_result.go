package internaldto

import (
	"github.com/lib/pq/oid"
)

type PrepareResultSetDTO struct {
	OutputBody  map[string]interface{}
	Msg         *BackendMessages
	RawRows     map[int]map[int]interface{}
	RowMap      map[string]map[string]interface{}
	ColumnOrder []string
	ColumnOIDs  []oid.Oid
	RowSort     func(map[string]map[string]interface{}) []string
	Err         error
}

func NewPrepareResultSetDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg *BackendMessages,
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     map[int]map[int]interface{}{},
	}
}

func NewPrepareResultSetPlusRawDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg *BackendMessages,
	rawRows map[int]map[int]interface{},
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     rawRows,
	}
}

func NewPrepareResultSetPlusRawAndTypesDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	columnOIDs []oid.Oid,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg *BackendMessages,
	rawRows map[int]map[int]interface{},
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		ColumnOIDs:  columnOIDs,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     rawRows,
	}
}
