package nativedb

import "github.com/stackql/stackql/internal/stackql/streaming"

type Select interface {
	GetColumns() []Column
	GetRows() streaming.MapReader
}

func NewSelect(columns []Column) Select {
	return &StandardSelect{
		columns: columns,
	}
}

func NewSelectWithRows(columns []Column, rows streaming.MapReader) Select {
	return &StandardSelect{
		columns: columns,
		rows:    rows,
	}
}

type StandardSelect struct {
	columns []Column
	rows    streaming.MapReader
}

func (sc *StandardSelect) GetColumns() []Column {
	return sc.columns
}

func (sc *StandardSelect) GetRows() streaming.MapReader {
	return sc.rows
}
