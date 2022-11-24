package relationaldto

var (
	_ RelationalTable = &standardRelationalTable{}
)

type RelationalTable interface {
	GetAlias() string
	GetBaseName() string
	GetColumns() []RelationalColumn
	GetName() string
	PushBackColumn(RelationalColumn)
	WithAlias(alias string) RelationalTable
}

func NewRelationalTable(name, baseName string) RelationalTable {
	return &standardRelationalTable{
		name:     name,
		baseName: baseName,
	}
}

type standardRelationalTable struct {
	alias    string
	name     string
	baseName string
	columns  []RelationalColumn
}

func (rt *standardRelationalTable) GetName() string {
	return rt.name
}

func (rt *standardRelationalTable) GetBaseName() string {
	return rt.baseName
}

func (rt *standardRelationalTable) GetAlias() string {
	return rt.alias
}

func (rt *standardRelationalTable) WithAlias(alias string) RelationalTable {
	rt.alias = alias
	return rt
}

func (rt *standardRelationalTable) GetColumns() []RelationalColumn {
	return rt.columns
}

func (rt *standardRelationalTable) PushBackColumn(col RelationalColumn) {
	rt.columns = append(rt.columns, col)
}
