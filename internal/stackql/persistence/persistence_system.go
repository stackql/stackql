package persistence

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/name_mangle"
	"github.com/stackql/any-sdk/public/discovery"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

var (
	_ discovery.PersistenceSystem = &SQLPersistenceSystem{}
)

type SQLPersistenceSystem struct {
	sqlSystem       sql_system.SQLSystem
	viewNameMangler name_mangle.NameMangler
}

func NewSQLPersistenceSystem(sqlSystem sql_system.SQLSystem) *SQLPersistenceSystem {
	return &SQLPersistenceSystem{
		sqlSystem:       sqlSystem,
		viewNameMangler: name_mangle.NewViewNameMangler(),
	}
}

func (s *SQLPersistenceSystem) GetSystemName() string {
	return s.sqlSystem.GetName()
}

func (s *SQLPersistenceSystem) HandleExternalTables(
	providerName string, externalTables map[string]anysdk.SQLExternalTable) error {
	for _, tbl := range externalTables {
		err := s.sqlSystem.RegisterExternalTable(
			providerName,
			tbl,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLPersistenceSystem) HandleViewCollection(viewCollection []anysdk.View) error {
	for i, view := range viewCollection {
		viewNameNaive := view.GetNameNaive()
		viewName := s.viewNameMangler.MangleName(viewNameNaive, i)
		_, viewExists := s.sqlSystem.GetViewByName(viewName)
		if !viewExists {
			// TODO: resolve any possible data race
			err := s.sqlSystem.CreateView(viewName, view.GetDDL(), true, view.GetRequiredParamNames())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SQLPersistenceSystem) CacheStoreGet(key string) ([]byte, error) {
	return s.sqlSystem.GetSQLEngine().CacheStoreGet(key)
}

func (s *SQLPersistenceSystem) CacheStorePut(key string, value []byte, expiration string, ttl int) error {
	return s.sqlSystem.GetSQLEngine().CacheStorePut(key, value, expiration, ttl)
}
