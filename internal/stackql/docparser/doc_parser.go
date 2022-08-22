package docparser

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	SchemaDelimiter            string = "."
	googleServiceKeyDelimiter  string = ":"
	stackqlServiceKeyDelimiter string = "__"
)

var (
	drmConfig        drm.DRMConfig  = drm.GetGoogleV1SQLiteConfig()
	outputOnlyRegexp *regexp.Regexp = regexp.MustCompile(`(?i)^\[Output.Only\].*$`)
	requiredRegexpV2 *regexp.Regexp = regexp.MustCompile(`(?i)^\[Required\].*$`)
)

func TranslateServiceKeyGenericProviderToIql(serviceKey string) string {
	return strings.Replace(serviceKey, googleServiceKeyDelimiter, stackqlServiceKeyDelimiter, -1)
}

func TranslateServiceKeyIqlToGenericProvider(serviceKey string) string {
	return strings.Replace(serviceKey, stackqlServiceKeyDelimiter, googleServiceKeyDelimiter, -1)
}

func OpenapiStackQLServiceDiscoveryDocPersistor(prov *openapistackql.Provider, svc *openapistackql.Service, dbEngine sqlengine.SQLEngine, prefix string) error {
	// replace := false
	discoveryGenerationId, err := dbEngine.GetCurrentDiscoveryGenerationId(prefix)
	if err != nil {
		discoveryGenerationId, err = dbEngine.GetNextDiscoveryGenerationId(prefix)
		if err != nil {
			return err
		}
	}
	version := svc.Info.Version
	var tabluationsAnnotated []util.AnnotatedTabulation
	for name, s := range svc.Components.Schemas {
		v := openapistackql.NewSchema(s.Value, svc, name)
		if v.IsArrayRef() {
			continue
		}
		// tableName := fmt.Sprintf("%s.%s", prefix, k)
		switch v.Type {
		case "object":
			tabulation := v.Tabulate(false)
			annTab := util.NewAnnotatedTabulation(tabulation, dto.NewHeirarchyIdentifiers(prov.Name, svc.GetName(), tabulation.GetName(), ""), "")
			tabluationsAnnotated = append(tabluationsAnnotated, annTab)
			if version == "v2" {
				for pr, prVal := range v.Properties {
					prValSc := openapistackql.NewSchema(prVal.Value, svc, pr)
					if prValSc != nil {
						if prValSc.IsArrayRef() {
							iSc := openapistackql.NewSchema(prValSc.Items.Value, svc, fmt.Sprintf("%s.%s.Items", v.Title, pr))
							tb := iSc.Tabulate(false)
							log.Infoln(fmt.Sprintf("tb = %v", tb))
							if tb != nil {
								annTab := util.NewAnnotatedTabulation(tb, dto.NewHeirarchyIdentifiers(prov.Name, svc.GetName(), tb.GetName(), ""), "")
								tabluationsAnnotated = append(tabluationsAnnotated, annTab)
							}
						}
					}
				}
			}
			// create table
		case "array":
			itemsSchema, _ := v.GetItemsSchema()
			if len(itemsSchema.Properties) > 0 {
				// create "inline" table
				tabulation := v.Tabulate(false)
				annTab := util.NewAnnotatedTabulation(tabulation, dto.NewHeirarchyIdentifiers(prov.Name, svc.GetName(), tabulation.GetName(), ""), "")
				tabluationsAnnotated = append(tabluationsAnnotated, annTab)
			}
		}
	}
	db, err := dbEngine.GetDB()
	if err != nil {
		return err
	}
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	for _, tblt := range tabluationsAnnotated {
		ddl := drmConfig.GenerateDDL(tblt, discoveryGenerationId, true)
		for _, q := range ddl {
			// log.Infoln(q)
			_, err = db.Exec(q)
			if err != nil {
				errStr := fmt.Sprintf("aborting DDL run on query = %s, err = %v", q, err)
				log.Infoln(errStr)
				txn.Rollback()
				return err
			}
		}
	}
	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}

func OpenapiStackQLTabulationsPersistor(prov *openapistackql.Provider, svc *openapistackql.Service, tabluationsAnnotated []util.AnnotatedTabulation, dbEngine sqlengine.SQLEngine, prefix string) (int, error) {
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
		ddl := drmConfig.GenerateDDL(tblt, discoveryGenerationId, false)
		for _, q := range ddl {
			// log.Infoln(q)
			_, err = db.Exec(q)
			if err != nil {
				errStr := fmt.Sprintf("aborting DDL run on query = %s, err = %v", q, err)
				log.Infoln(errStr)
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

func isAlwaysRequired(item interface{}) bool {
	if rMap, ok := item.(map[string]interface{}); ok {
		if desc, ok := rMap["description"]; ok {
			if descStr, ok := desc.(string); ok {
				return requiredRegexpV2.MatchString(descStr)
			}
		}
	}
	return false
}

func getRequiredIfPresent(item interface{}) map[string]bool {
	var retVal map[string]bool
	if item != nil {
		if rMap, ok := item.(map[string]interface{}); ok {
			if ref, ok := rMap["annotations"]; ok {
				if ann, ok := ref.(map[string]interface{}); ok {
					if req, ok := ann["required"]; ok {
						switch req := req.(type) {
						case []interface{}:
							retVal = make(map[string]bool)
							for _, s := range req {
								switch v := s.(type) {
								case string:
									retVal[v] = true
								}
							}
						}
					}
				}
			}
		}
	}
	return retVal
}
