package docparser

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"
)

const (
	SchemaDelimiter            string = "."
	googleServiceKeyDelimiter  string = ":"
	stackqlServiceKeyDelimiter string = "__"
)

func TranslateServiceKeyGenericProviderToIql(serviceKey string) string {
	return strings.Replace(serviceKey, googleServiceKeyDelimiter, stackqlServiceKeyDelimiter, -1)
}

func TranslateServiceKeyIqlToGenericProvider(serviceKey string) string {
	return strings.Replace(serviceKey, stackqlServiceKeyDelimiter, googleServiceKeyDelimiter, -1)
}

func OpenapiStackQLTabulationsPersistor(
	m *openapistackql.OperationStore,
	tabluationsAnnotated []util.AnnotatedTabulation,
	dbEngine sqlengine.SQLEngine,
	prefix string,
	namespaceCollection tablenamespace.TableNamespaceCollection,
	controlAttributes sqlcontrol.ControlAttributes,
	sqlDialect sqldialect.SQLDialect,
) (int, error) {
	drmCfg, err := drm.GetDRMConfig(sqlDialect, namespaceCollection, controlAttributes)
	if err != nil {
		return 0, err
	}
	discoveryGenerationId, err := dbEngine.GetCurrentDiscoveryGenerationId(prefix)
	if err != nil {
		discoveryGenerationId, err = dbEngine.GetNextDiscoveryGenerationId(prefix)
		if err != nil {
			return discoveryGenerationId, err
		}
	}
	db, err := dbEngine.GetDB()
	if err != nil {
		return discoveryGenerationId, err
	}
	txn, err := db.Begin()
	if err != nil {
		return discoveryGenerationId, err
	}
	for _, tblt := range tabluationsAnnotated {
		ddl, err := drmCfg.GenerateDDL(tblt, m, discoveryGenerationId, false)
		if err != nil {
			displayErr := fmt.Errorf("error generating DDL: %s", err.Error())
			logging.GetLogger().Infoln(displayErr.Error())
			txn.Rollback()
			return discoveryGenerationId, displayErr
		}
		for _, q := range ddl {
			_, err = db.Exec(q)
			if err != nil {
				displayErr := fmt.Errorf("aborting DDL run for query '''%s''' with error: %s", q, err.Error())
				logging.GetLogger().Infof("aborting DDL run for query '''%s''' with error: %s\n", q, err.Error())
				txn.Rollback()
				return discoveryGenerationId, displayErr
			}
		}
	}
	err = txn.Commit()
	if err != nil {
		return discoveryGenerationId, err
	}
	return discoveryGenerationId, nil
}
