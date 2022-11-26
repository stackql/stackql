package relationaldto

import (
	"github.com/stackql/stackql/internal/stackql/dto"
)

var (
	_ RelationalTable     = &standardRelationalTable{}
	_ NomenclatureContext = &standardNomenclatureContext{}
)

type TableNomenclatureEncodingAlgorithm int

const (
	VerbatimNomenclatureEncoding TableNomenclatureEncodingAlgorithm = iota
	BasicCondenseNomenclatureEncoding
)

type NomenclatureContext interface {
	GetAlgorithm() TableNomenclatureEncodingAlgorithm
	GetMaxWidth() int
	WithMaxWidth(int) NomenclatureContext
}

func NewNomenclatureContext(algo TableNomenclatureEncodingAlgorithm) (NomenclatureContext, error) {
	return &standardNomenclatureContext{
		algo: algo,
	}, nil
}

func (nc *standardNomenclatureContext) WithMaxWidth(maxWidth int) NomenclatureContext {
	nc.maxWidth = maxWidth
	return nc
}

type standardNomenclatureContext struct {
	algo     TableNomenclatureEncodingAlgorithm
	maxWidth int
}

func (nc *standardNomenclatureContext) GetAlgorithm() TableNomenclatureEncodingAlgorithm {
	return nc.algo
}

func (nc *standardNomenclatureContext) GetMaxWidth() int {
	return nc.maxWidth
}

type RelationalTable interface {
	GetAlias() string
	GetBaseName() string
	GetColumns() []RelationalColumn
	GetName(NomenclatureContext) (string, error)
	PushBackColumn(RelationalColumn)
	WithAlias(alias string) RelationalTable
}

func NewRelationalTable(hIDs *dto.HeirarchyIdentifiers, discoveryID int, name, baseName string) RelationalTable {
	return &standardRelationalTable{
		hIDs:        hIDs,
		name:        name,
		baseName:    baseName,
		discoveryID: discoveryID,
	}
}

type standardRelationalTable struct {
	alias       string
	name        string
	baseName    string
	discoveryID int
	hIDs        *dto.HeirarchyIdentifiers
	columns     []RelationalColumn
}

func (rt *standardRelationalTable) GetName(ctx NomenclatureContext) (string, error) {
	return rt.name, nil
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
