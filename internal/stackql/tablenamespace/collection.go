package tablenamespace

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
)

type Collection interface {
	GetAnalyticsCacheTableNamespaceConfigurator() Configurator
	GetViewsTableNamespaceConfigurator() Configurator
	WithSQLSystem(sql_system.SQLSystem) (Collection, error)
}

func NewStandardTableNamespaceCollection(
	cfg map[string]dto.NamespaceCfg,
	sqlEngine sqlengine.SQLEngine,
) (Collection, error) {
	// nil dereference protect
	if cfg == nil {
		cfg = map[string]dto.NamespaceCfg{}
	}
	analyticsCfgDirector := getAnalyticsCacheTableNamespaceConfiguratorBuilderDirector(cfg["analytics"], sqlEngine)
	viewsCfgDirector := getViewsTableNamespaceConfiguratorBuilderDirector(cfg["views"], sqlEngine)
	err := analyticsCfgDirector.Construct()
	if err != nil {
		return nil, err
	}
	err = viewsCfgDirector.Construct()
	if err != nil {
		return nil, err
	}
	rv := &StandardTableNamespaceCollection{
		analyticsCfg: analyticsCfgDirector.GetResult(),
		viewCfg:      viewsCfgDirector.GetResult(),
		sqlEngine:    sqlEngine,
	}
	return rv, nil
}

type StandardTableNamespaceCollection struct {
	analyticsCfg Configurator
	viewCfg      Configurator
	sqlEngine    sqlengine.SQLEngine
}

func (col *StandardTableNamespaceCollection) GetAnalyticsCacheTableNamespaceConfigurator() Configurator {
	return col.analyticsCfg
}

func (col *StandardTableNamespaceCollection) GetViewsTableNamespaceConfigurator() Configurator {
	return col.viewCfg
}

func (col *StandardTableNamespaceCollection) WithSQLSystem(
	sqlSystem sql_system.SQLSystem,
) (Collection, error) {
	_, err := col.analyticsCfg.WithSQLSystem(sqlSystem)
	if err != nil {
		return nil, err
	}
	_, err = col.viewCfg.WithSQLSystem(sqlSystem)
	if err != nil {
		return nil, err
	}
	return col, nil
}
