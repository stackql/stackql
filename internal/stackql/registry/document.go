package registry

import (
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"gopkg.in/yaml.v3"
)

// ProviderDocument is the host's registration metadata; provider requests use Omni.
type ProviderDocument interface {
	GetName() string
	GetMinStackQLVersion() string
	GetAuth() (*omnisdk.ProviderAuth, bool)
	ExternalTables() (map[string]sqlengine.SQLExternalTable, error)
}

type providerDocument struct {
	Name   string `yaml:"name"`
	Config struct {
		Auth    *omnisdk.ProviderAuth `yaml:"auth"`
		Minimum string                `yaml:"minStackQLVersion"`
		Tables  map[string]yaml.Node  `yaml:"sqlExternalTables"`
	} `yaml:"config"`
}

func parseProviderDocument(data []byte) (ProviderDocument, error) {
	var doc providerDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
func (d *providerDocument) GetName() string              { return d.Name }
func (d *providerDocument) GetMinStackQLVersion() string { return d.Config.Minimum }
func (d *providerDocument) GetAuth() (*omnisdk.ProviderAuth, bool) {
	return d.Config.Auth, d.Config.Auth != nil
}
func (d *providerDocument) ExternalTables() (map[string]sqlengine.SQLExternalTable, error) {
	out := make(map[string]sqlengine.SQLExternalTable, len(d.Config.Tables))
	for name, node := range d.Config.Tables {
		data, err := yaml.Marshal(&node)
		if err != nil {
			return nil, err
		}
		table, err := sqlengine.ExternalTable(data)
		if err != nil {
			return nil, err
		}
		out[name] = table
	}
	return out, nil
}
