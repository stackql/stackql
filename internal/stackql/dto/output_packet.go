package dto

import (
	"github.com/lib/pq/oid"
)

type OutputPacket interface {
	GetRows() map[string]map[string]interface{}
	GetRawRows() map[int]map[int]interface{}
	GetColumnNames() []string
	GetColumnOIDs() []oid.Oid
}

func NewStandardOutputPacket(
	rowMaps map[string]map[string]interface{},
	rawRows map[int]map[int]interface{},
	columnNames []string,
	columnOIDs []oid.Oid,
) OutputPacket {
	return &standardOutputPacket{
		rowMaps:     rowMaps,
		rawRows:     rawRows,
		columnNames: columnNames,
		columnOIDs:  columnOIDs,
	}
}

type standardOutputPacket struct {
	rowMaps     map[string]map[string]interface{}
	rawRows     map[int]map[int]interface{}
	columnNames []string
	columnOIDs  []oid.Oid
}

func (op *standardOutputPacket) GetRows() map[string]map[string]interface{} {
	return op.rowMaps
}

func (op *standardOutputPacket) GetRawRows() map[int]map[int]interface{} {
	return op.rawRows
}

func (op *standardOutputPacket) GetColumnNames() []string {
	return op.columnNames
}

func (op *standardOutputPacket) GetColumnOIDs() []oid.Oid {
	return op.columnOIDs
}
