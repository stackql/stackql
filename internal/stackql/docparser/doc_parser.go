package docparser

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"
)

const (
	SchemaDelimiter            string = "."
	googleServiceKeyDelimiter  string = ":"
	stackqlServiceKeyDelimiter string = "__"
)

var (
	drmConfig drm.DRMConfig = drm.GetGoogleV1SQLiteConfig()
)

func TranslateServiceKeyGenericProviderToIql(serviceKey string) string {
	return strings.Replace(serviceKey, googleServiceKeyDelimiter, stackqlServiceKeyDelimiter, -1)
}

func TranslateServiceKeyIqlToGenericProvider(serviceKey string) string {
	return strings.Replace(serviceKey, stackqlServiceKeyDelimiter, googleServiceKeyDelimiter, -1)
}

func OpenapiStackQLTabulationsPersistor(m *openapistackql.OperationStore, tabluationsAnnotated []util.AnnotatedTabulation, dbEngine sqlengine.SQLEngine, prefix string) (int, error) {
	// replace := false
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
		ddl := drmConfig.GenerateDDL(tblt, m, discoveryGenerationId, false)
		for _, q := range ddl {
			// logging.GetLogger().Infoln(q)
			_, err = db.Exec(q)
			if err != nil {
				errStr := fmt.Sprintf("aborting DDL run on query = %s, err = %v", q, err)
				logging.GetLogger().Infoln(errStr)
				txn.Rollback()
				return discoveryGenerationId, err
			}
		}
	}
	err = txn.Commit()
	if err != nil {
		return discoveryGenerationId, err
	}
	return discoveryGenerationId, nil
}
