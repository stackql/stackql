package nativedb

type Column interface {
	GetName() string
	GetType() string
}

func NewColumn(columnName, columnType string) Column {
	return &StandardColumn{
		columnName: columnName,
		columnType: columnType,
	}
}

type StandardColumn struct {
	columnName string
	columnType string
}

func (sc *StandardColumn) GetName() string {
	return sc.columnName
}

func (sc *StandardColumn) GetType() string {
	return sc.columnType
}
