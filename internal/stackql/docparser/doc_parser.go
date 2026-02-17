package docparser

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/db/sqlcontrol"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/internal/stackql/util"

	"strings"
)

const (
	SchemaDelimiter            string = "."
	googleServiceKeyDelimiter  string = ":"
	stackqlServiceKeyDelimiter string = "__"
)

func TranslateServiceKeyGenericProviderToIql(serviceKey string) string {
	//nolint:gocritic // TODO: review
	return strings.Replace(serviceKey, googleServiceKeyDelimiter, stackqlServiceKeyDelimiter, -1)
}

func TranslateServiceKeyIqlToGenericProvider(serviceKey string) string {
	//nolint:gocritic // TODO: review
	return strings.Replace(serviceKey, stackqlServiceKeyDelimiter, googleServiceKeyDelimiter, -1)
}

func OpenapiStackQLTabulationsPersistor(
	prov formulation.Provider,
	svc formulation.Service,
	resource formulation.Resource,
	m formulation.StandardOperationStore,
	isAwait bool,
	tabluationsAnnotated []util.AnnotatedTabulation,
	dbEngine sqlengine.SQLEngine,
	prefix string,
	namespaceCollection tablenamespace.Collection,
	controlAttributes sqlcontrol.ControlAttributes,
	sqlSystem sql_system.SQLSystem,
	persistenceSystem formulation.PersistenceSystem,
	typCfg typing.Config,
) (int, error) {
	drmCfg, err := drm.GenerateDRMConfig(sqlSystem, persistenceSystem,
		typCfg, namespaceCollection, controlAttributes)
	if err != nil {
		return 0, err
	}
	discoveryGenerationID, err := dbEngine.GetCurrentDiscoveryGenerationID(prefix)
	if err != nil {
		discoveryGenerationID, err = dbEngine.GetNextDiscoveryGenerationID(prefix)
		if err != nil {
			return discoveryGenerationID, err
		}
	}
	db, err := dbEngine.GetDB()
	if err != nil {
		return discoveryGenerationID, err
	}
	txn, err := db.Begin()
	if err != nil {
		return discoveryGenerationID, err
	}
	for _, tblt := range tabluationsAnnotated {
		ddl, ddlErr := drmCfg.GenerateDDL(tblt, prov, svc, resource, m, isAwait, discoveryGenerationID, false, true)
		if ddlErr != nil {
			displayErr := fmt.Errorf("error generating DDL: %w", ddlErr)
			logging.GetLogger().Infoln(displayErr.Error())
			txn.Rollback() //nolint:errcheck // TODO: investigate
			return discoveryGenerationID, displayErr
		}
		for _, q := range ddl {
			logging.GetLogger().Infof("DDL about to run: '''%q'''", q)
			_, err = txn.Exec(q)
			if err != nil {
				displayErr := fmt.Errorf("aborting DDL run for query '''%s''' with error: %w", q, err)
				logging.GetLogger().Infof("aborting DDL run for query '''%s''' with error: %s", q, err.Error())
				txn.Rollback() //nolint:errcheck // TODO: investigate
				return discoveryGenerationID, displayErr
			}
		}
	}
	err = txn.Commit()
	if err != nil {
		return discoveryGenerationID, err
	}
	return discoveryGenerationID, nil
}
