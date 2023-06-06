package internaldto

import (
	"github.com/lib/pq/oid"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type PrepareResultSetDTO struct {
	OutputBody  map[string]interface{}
	Msg         BackendMessages
	RawRows     map[int]map[int]interface{}
	RowMap      map[string]map[string]interface{}
	ColumnOrder []string
	ColumnOIDs  []oid.Oid
	RowSort     func(map[string]map[string]interface{}) []string
	Err         error
	TypCfg      typing.Config
}

func NewPrepareResultSetDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg BackendMessages,
	typCfg typing.Config,
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     map[int]map[int]interface{}{},
		TypCfg:      typCfg,
	}
}

func NewPrepareResultSetPlusRawDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg BackendMessages,
	rawRows map[int]map[int]interface{},
	typCfg typing.Config,
) PrepareResultSetDTO {
	return PrepareResultSetDTO{
		OutputBody:  body,
		RowMap:      rowMap,
		ColumnOrder: columnOrder,
		RowSort:     rowSort,
		Err:         err,
		Msg:         msg,
		RawRows:     rawRows,
		TypCfg:      typCfg,
	}
}

func NewPrepareResultSetPlusRawAndTypesDTO(
	body map[string]interface{},
	rowMap map[string]map[string]interface{},
	columnOrder []string,
	columnOIDs []oid.Oid,
	rowSort func(map[string]map[string]interface{}) []string,
	err error,
	msg BackendMessages,
	rawRows map[int]map[int]interface{},
	typCfg typing.Config,
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
		TypCfg:      typCfg,
	}
}
