package sqlengine

import "gopkg.in/yaml.v3"

var (
	_ SQLExternalConnection = standardSQLExternalConnection{}
	_ SQLExternalTable      = standardSQLExternalTable{}
	_ SQLExternalColumn     = standardSQLExternalColumn{}
)

type SQLExternalConnection interface {
	GetConnectionName() string
	GetTables() map[string]SQLExternalTable
}

type standardSQLExternalConnection struct {
	ConnectionName string                      `json:"alias" yaml:"alias"`
	Tables         map[string]SQLExternalTable `json:"tables" yaml:"tables"`
}

func (c standardSQLExternalConnection) GetConnectionName() string {
	return c.ConnectionName
}

func (c standardSQLExternalConnection) GetTables() map[string]SQLExternalTable {
	return c.Tables
}

type SQLExternalTable interface {
	GetCatalogName() string
	GetSchemaName() string
	GetName() string
	GetColumns() []SQLExternalColumn
}

type standardSQLExternalTable struct {
	CatalogName string                       `json:"catalogName" yaml:"catalogName"`
	SchemaName  string                       `json:"schemaName" yaml:"schemaName"`
	Name        string                       `json:"name" yaml:"name"`
	Columns     []*standardSQLExternalColumn `json:"columns" yaml:"columns"`
}

func (t standardSQLExternalTable) GetCatalogName() string {
	return t.CatalogName
}

func (t standardSQLExternalTable) GetSchemaName() string {
	return t.SchemaName
}

func (t standardSQLExternalTable) GetName() string {
	return t.Name
}

func (t standardSQLExternalTable) GetColumns() []SQLExternalColumn {
	var rv []SQLExternalColumn
	for _, c := range t.Columns {
		rv = append(rv, c)
	}
	return rv
}

type SQLExternalColumn interface {
	GetName() string
	GetType() string
	GetOid() uint32
	GetWidth() int
	GetPrecision() int
}

type standardSQLExternalColumn struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"`
	Oid       uint32 `json:"oid" yaml:"oid"`
	Width     int    `json:"width" yaml:"width"`
	Precision int    `json:"precision" yaml:"precision"`
	// OrdinalPosition int    `json:"ordinalPosition" yaml:"ordinalPosition"`
}

func (c standardSQLExternalColumn) GetName() string {
	return c.Name
}

func (c standardSQLExternalColumn) GetType() string {
	return c.Type
}

func (c standardSQLExternalColumn) GetOid() uint32 {
	return c.Oid
}

func (c standardSQLExternalColumn) GetWidth() int {
	return c.Width
}

func (c standardSQLExternalColumn) GetPrecision() int {
	return c.Precision
}

// ExternalTable decodes a table declaration owned by the host SQL backend.
func ExternalTable(data []byte) (SQLExternalTable, error) {
	var table standardSQLExternalTable
	if err := yaml.Unmarshal(data, &table); err != nil {
		return nil, err
	}
	return table, nil
}
