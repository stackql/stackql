package provider

import (
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql/internal/stackql/registry"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

// DummyAuth is retained for the existing test harness; execution authentication is owned by Omni.
var DummyAuth bool //nolint:gochecknoglobals // legacy test harness

type IProvider interface {
	CheckCredentialFile(*dto.AuthCtx) error
	ShowAuth(*dto.AuthCtx) (*omnisdk.AuthMetadata, error)
	GetProviderString() string
	GetProvider() (registry.ProviderDocument, error)
	PersistStaticExternalSQLDataSource(dto.RuntimeCtx) error
}

func GenerateProvider(
	_ dto.RuntimeCtx, name, version string, reg registry.RegistryAPI, system sql_system.SQLSystem,
) (IProvider, error) {
	doc, err := reg.LoadProviderByName(name, version)
	if err != nil {
		return nil, err
	}
	return &GenericProvider{document: doc, sqlSystem: system}, nil
}
