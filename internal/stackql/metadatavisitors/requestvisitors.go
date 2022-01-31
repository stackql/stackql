package metadatavisitors

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/iqlutil"

	"sort"
	"strings"

	"github.com/stackql/stackql/internal/pkg/prettyprint"

	"github.com/stackql/go-openapistackql/openapistackql"

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/vt/sqlparser"
)

type SchemaRequestTemplateVisitor struct {
	MaxDepth       int
	Strategy       string
	PrettyPrinter  *prettyprint.PrettyPrinter
	visitedObjects map[string]bool
	requiredOnly   bool
}

func NewSchemaRequestTemplateVisitor(maxDepth int, strategy string, prettyPrinter *prettyprint.PrettyPrinter, requiredOnly bool) *SchemaRequestTemplateVisitor {
	return &SchemaRequestTemplateVisitor{
		MaxDepth:       maxDepth,
		Strategy:       strategy,
		PrettyPrinter:  prettyPrinter,
		visitedObjects: make(map[string]bool),
		requiredOnly:   requiredOnly,
	}
}

func (sv *SchemaRequestTemplateVisitor) recordSchemaVisited(schemaKey string) {
	sv.visitedObjects[schemaKey] = true
}

func (sv *SchemaRequestTemplateVisitor) isVisited(schemaKey string, localVisited map[string]bool) bool {
	if localVisited != nil {
		if localVisited[schemaKey] {
			return true
		}
	}
	return sv.visitedObjects[schemaKey]
}

func checkAllColumnsPresent(columns sqlparser.Columns, toInclude map[string]bool) error {
	var missingColNames []string
	if columns != nil {
		for _, col := range columns {
			cName := col.GetRawVal()
			if !toInclude[cName] {
				missingColNames = append(missingColNames, cName)
			}
		}
		if len(missingColNames) > 0 {
			return fmt.Errorf("cannot find the following columns: %s", strings.Join(missingColNames, ", "))
		}
	}
	return nil
}

func getColsMap(columns sqlparser.Columns) map[string]bool {
	retVal := make(map[string]bool)
	for _, col := range columns {
		retVal[col.GetRawVal()] = true
	}
	return retVal

}

func isColIncludable(key string, columns sqlparser.Columns, colMap map[string]bool) bool {
	colOk := columns == nil
	if colOk {
		return colOk
	}
	return colMap[key]
}

func isBodyParam(paramName string) bool {
	return strings.HasPrefix(paramName, constants.RequestBodyBaseKey)
}

func ToInsertStatement(columns sqlparser.Columns, m *openapistackql.OperationStore, svc *openapistackql.Service, extended bool, prettyPrinter *prettyprint.PrettyPrinter, requiredOnly bool) (string, error) {
	paramsToInclude := m.GetParameters()
	successfullyIncludedCols := make(map[string]bool)
	if !extended {
		paramsToInclude = m.GetRequiredParameters()
	}
	if columns != nil {
		paramsToInclude = make(map[string]*openapistackql.Parameter)
		for _, col := range columns {
			cName := col.GetRawVal()
			if !isBodyParam(cName) {
				p, ok := m.GetParameter(cName)
				if !ok {
					return "", fmt.Errorf("cannot generate insert statement: column '%s' not present", cName)
				}
				paramsToInclude[cName] = p
				successfullyIncludedCols[cName] = true
			}
		}
	}
	var includedParamNames []string
	for k, _ := range paramsToInclude {
		includedParamNames = append(includedParamNames, k)
	}
	sort.Strings(includedParamNames)
	var columnList, exprList []string
	for _, s := range includedParamNames {
		p, ok := m.GetParameter(s)
		if !ok {
			return "", fmt.Errorf("'%s'", s)
		}
		columnList = append(columnList, prettyPrinter.RenderColumnName(s))
		switch p.GetType() {
		case "string":
			exprList = append(exprList, prettyPrinter.RenderTemplateVarAndDelimit(s))
		default:
			exprList = append(exprList, prettyPrinter.RenderTemplateVarNoDelimit(s))
		}
	}

	sch, err := m.GetRequestBodySchema()

	if err != nil {
		return "", err
	}

	if sch == nil {
		err := checkAllColumnsPresent(columns, successfullyIncludedCols)
		return "INSERT INTO %s" + "(\n" + strings.Join(columnList, ",\n") +
			"\n)\n" + "SELECT\n" + strings.Join(exprList, ",\n") + "\n;\n", err
	}

	schemaVisitor := NewSchemaRequestTemplateVisitor(2, "", prettyPrinter, requiredOnly)

	tVal, _ := schemaVisitor.RetrieveTemplate(sch, m, extended)

	log.Infoln(fmt.Sprintf("tVal = %v", tVal))

	colMap := getColsMap(columns)

	if columns != nil {
		for _, c := range columns {
			cName := c.GetRawVal()
			if !isBodyParam(cName) {
				continue
			}
			cNameSuffix := strings.TrimPrefix(cName, constants.RequestBodyBaseKey)
			if v, ok := tVal[cNameSuffix]; ok {
				columnList = append(columnList, prettyPrinter.RenderColumnName(cName))
				exprList = append(exprList, v)
				successfullyIncludedCols[cName] = true
			}
		}
	} else {
		tValKeysSorted := iqlutil.GetSortedKeysStringMap(tVal)
		for _, k := range tValKeysSorted {
			v := tVal[k]
			if isColIncludable(k, columns, colMap) {
				columnList = append(columnList, prettyPrinter.RenderColumnName(constants.RequestBodyBaseKey+k))
				exprList = append(exprList, v)
			}
		}
	}

	err = checkAllColumnsPresent(columns, successfullyIncludedCols)
	retVal := "INSERT INTO %s" + "(\n" + strings.Join(columnList, ",\n") +
		"\n)\n" + "SELECT\n" + strings.Join(exprList, ",\n") + "\n;\n"
	return retVal, err
}

func (sv *SchemaRequestTemplateVisitor) processSubSchemasMap(sc *openapistackql.Schema, method *openapistackql.OperationStore, properties map[string]*openapistackql.Schema) (map[string]string, error) {
	retVal := make(map[string]string)
	for k, ss := range properties {
		log.Infoln(fmt.Sprintf("RetrieveTemplate() k = '%s', ss is nil ? '%t'", k, ss == nil))
		if ss != nil && (k == "" || !sv.isVisited(k, nil)) {
			localSchemaVisitedMap := make(map[string]bool)
			localSchemaVisitedMap[k] = true
			if !method.IsRequiredRequestBodyProperty(k) && (ss.ReadOnly || (sv.requiredOnly && !sc.IsRequired(k))) {
				log.Infoln(fmt.Sprintf("property = '%s' will be skipped", k))
				continue
			}
			rv, err := sv.retrieveTemplateVal(ss, ".values."+constants.RequestBodyBaseKey+k, localSchemaVisitedMap)
			if err != nil {
				return nil, err
			}
			switch rvt := rv.(type) {
			case map[string]interface{}, []interface{}, string:
				bytes, err := sv.PrettyPrinter.PrintTemplatedJSON(rvt)
				if err != nil {
					return nil, err
				}
				retVal[k] = string(bytes)
			case nil:
				continue
			default:
				return nil, fmt.Errorf("error processing template key '%s' with disallowed type '%T'", k, rvt)
			}
		}
	}
	return retVal, nil
}

func (sv *SchemaRequestTemplateVisitor) RetrieveTemplate(sc *openapistackql.Schema, method *openapistackql.OperationStore, extended bool) (map[string]string, error) {
	retVal := make(map[string]string)
	sv.recordSchemaVisited(sc.GetName())
	switch sc.Type {
	case "object":
		prop, err := sc.GetProperties()
		if err != nil {
			return nil, err
		}
		retVal, err = sv.processSubSchemasMap(sc, method, prop)
		if len(retVal) != 0 || err != nil {
			return retVal, err
		}
		if sc.AdditionalProperties != nil && sc.AdditionalProperties.Value != nil {
			retVal, err = sv.processSubSchemasMap(sc, method, map[string]*openapistackql.Schema{"k1": openapistackql.NewSchema(sc.AdditionalProperties.Value, "k1")})
		}
		if len(retVal) == 0 {
			return nil, nil
		}
		return retVal, err
	}
	return nil, fmt.Errorf("templating of request body only supported for object type payload")
}

func (sv *SchemaRequestTemplateVisitor) retrieveTemplateVal(sc *openapistackql.Schema, objectKey string, localSchemaVisitedMap map[string]bool) (interface{}, error) {
	sSplit := strings.Split(objectKey, ".")
	oKey := sSplit[len(sSplit)-1]
	oPrefix := objectKey
	if len(sSplit) > 1 {
		oPrefix = strings.TrimSuffix(objectKey, "."+oKey)
	} else {
		oPrefix = ""
	}
	templateValSuffix := oKey
	templateValName := oPrefix + "." + templateValSuffix
	if oPrefix == "" {
		templateValName = templateValSuffix
	}
	initialLocalSchemaVisitedMap := make(map[string]bool)
	for k, v := range localSchemaVisitedMap {
		initialLocalSchemaVisitedMap[k] = v
	}
	switch sc.Type {
	case "object":
		rv := make(map[string]interface{})
		props, err := sc.GetProperties()
		if err != nil {
			return nil, err
		}
		for k, ss := range props {
			propertyLocalSchemaVisitedMap := make(map[string]bool)
			for k, v := range initialLocalSchemaVisitedMap {
				propertyLocalSchemaVisitedMap[k] = v
			}
			if ss != nil && ((ss.Type != "array") || !sv.isVisited(ss.Title, propertyLocalSchemaVisitedMap)) {
				propertyLocalSchemaVisitedMap[ss.Title] = true
				sv, err := sv.retrieveTemplateVal(ss, templateValName+"."+k, propertyLocalSchemaVisitedMap)
				if err != nil {
					return nil, err
				}
				if sv != nil {
					rv[k] = sv
				}
			}
		}
		if len(rv) == 0 {
			if sc.AdditionalProperties != nil {
				if aps := sc.AdditionalProperties.Value; aps != nil {
					aps := openapistackql.NewSchema(aps, "additionalProperties")
					hasProperties := false
					for k, v := range aps.Properties {
						hasProperties = true
						ss := openapistackql.NewSchema(v.Value, k)
						if k == "" {
							k = "key"
						}
						key := fmt.Sprintf("{{ %s[0].%s }}", templateValName, k)
						rv[key] = getAdditionalStuff(ss, templateValName)
					}
					if !hasProperties {
						key := fmt.Sprintf("{{ %s[0].%s }}", templateValName, "key")
						rv[key] = getAdditionalStuff(aps, templateValName)
					}
				}
			}
		}
		if len(rv) == 0 {
			return nil, nil
		}
		return rv, nil
	case "array":
		var arr []interface{}
		iSch, err := sc.GetItemsSchema()
		if err != nil {
			return nil, err
		}
		itemLocalSchemaVisitedMap := make(map[string]bool)
		for k, v := range initialLocalSchemaVisitedMap {
			itemLocalSchemaVisitedMap[k] = v
		}
		itemS, err := sv.retrieveTemplateVal(iSch, templateValName+"[0]", itemLocalSchemaVisitedMap)
		arr = append(arr, itemS)
		if err != nil {
			return nil, err
		}
		return arr, nil
	case "string":
		return "\"{{ " + templateValName + " }}\"", nil
	default:
		return "{{ " + templateValName + " }}", nil
	}
	return nil, nil
}

func getAdditionalStuff(ss *openapistackql.Schema, templateValName string) string {
	valBase := fmt.Sprintf("{{ %s[0].val }}", templateValName)
	switch ss.Type {
	case "string":
		return fmt.Sprintf(`"%s"`, valBase)
	case "number", "int", "int32", "int64":
		return valBase
	default:
		return valBase
	}
}
