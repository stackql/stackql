package provider

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/registry"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

// GenericProvider retains host registration metadata, never request execution.
type GenericProvider struct {
	document  registry.ProviderDocument
	sqlSystem sql_system.SQLSystem
}

func (p *GenericProvider) GetProviderString() string                       { return p.document.GetName() }
func (p *GenericProvider) GetProvider() (registry.ProviderDocument, error) { return p.document, nil }
func (p *GenericProvider) PersistStaticExternalSQLDataSource(_ dto.RuntimeCtx) error {
	tables, err := p.document.ExternalTables()
	if err != nil {
		return err
	}
	if len(tables) == 0 {
		return fmt.Errorf("no external tables supplied")
	}
	for _, table := range tables {
		if err = p.sqlSystem.RegisterExternalTable(p.GetProviderString(), table); err != nil {
			return err
		}
	}
	return nil
}
