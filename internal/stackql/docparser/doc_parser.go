package docparser

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
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
	m anysdk.OperationStore,
	tabluationsAnnotated []util.AnnotatedTabulation,
	dbEngine sqlengine.SQLEngine,
	prefix string,
	namespaceCollection tablenamespace.Collection,
	controlAttributes sqlcontrol.ControlAttributes,
	sqlSystem sql_system.SQLSystem,
	typCfg typing.Config,
) (int, error) {
	drmCfg, err := drm.GetDRMConfig(sqlSystem, typCfg, namespaceCollection, controlAttributes)
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
		ddl, ddlErr := drmCfg.GenerateDDL(tblt, m, discoveryGenerationID, false)
		if ddlErr != nil {
			displayErr := fmt.Errorf("error generating DDL: %w", err)
			logging.GetLogger().Infoln(displayErr.Error())
			txn.Rollback() //nolint:errcheck // TODO: investigate
			return discoveryGenerationID, displayErr
		}
		for _, q := range ddl {
			_, err = txn.Exec(q)
			if err != nil {
				displayErr := fmt.Errorf("aborting DDL run for query '''%s''' with error: %w", q, err)
				logging.GetLogger().Infof("aborting DDL run for query '''%s''' with error: %s\n", q, err.Error())
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
