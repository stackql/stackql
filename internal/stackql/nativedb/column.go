package nativedb

type Column interface {
	GetName() string
	GetType() string
	GetWidth() (int, bool)
	SetWidth(width int)
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
	width      int
}

func (sc *StandardColumn) GetName() string {
	return sc.columnName
}

func (sc *StandardColumn) GetType() string {
	return sc.columnType
}

func (sc *StandardColumn) GetWidth() (int, bool) {
	return sc.width, sc.width != 0
}

func (sc *StandardColumn) SetWidth(width int) {
	sc.width = width
}
