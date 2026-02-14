package metadatavisitors

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/public/formulation"

	"sort"
	"strings"

	"github.com/stackql/stackql/pkg/prettyprint"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ TemplatedProduct = &standardTemplatedProduct{}
)

const (
	strType    = "string"
	objectType = "object"
	arrayType  = "array"
	symLHS     = "<< "
	symRHS     = " >>"
)

type TemplatedProduct interface {
	GetBody() string
	GetPlaceholder() string
}

type standardTemplatedProduct struct {
	body        string
	placeholder string
}

func NewTemplatedProduct(body, placeholder string) TemplatedProduct {
	return &standardTemplatedProduct{
		body:        body,
		placeholder: placeholder,
	}
}

func (tp *standardTemplatedProduct) GetBody() string {
	return tp.body
}

func (tp *standardTemplatedProduct) GetPlaceholder() string {
	return tp.placeholder
}

type SchemaRequestTemplateVisitor struct {
	MaxDepth                 int
	Strategy                 string
	PrettyPrinter            *prettyprint.PrettyPrinter
	PlaceholderPrettyPrinter *prettyprint.PrettyPrinter
	visitedObjects           map[string]bool
	requiredOnly             bool
}

func NewSchemaRequestTemplateVisitor(
	maxDepth int,
	strategy string,
	prettyPrinter, placeHolderPrettyPrinter *prettyprint.PrettyPrinter,
	requiredOnly bool) *SchemaRequestTemplateVisitor {
	return &SchemaRequestTemplateVisitor{
		MaxDepth:                 maxDepth,
		Strategy:                 strategy,
		PrettyPrinter:            prettyPrinter,
		PlaceholderPrettyPrinter: placeHolderPrettyPrinter,
		visitedObjects:           make(map[string]bool),
		requiredOnly:             requiredOnly,
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

func isRequestBodyParam(paramName string, m anysdk.OperationStore) bool {
	return m.IsRequestBodyAttributeRenamed(paramName)
}

//nolint:funlen,gocognit,revive // acceptable
func ToInsertStatement(
	columns sqlparser.Columns,
	m anysdk.OperationStore,
	svc anysdk.Service,
	extended bool, prettyPrinter,
	placeHolderPrettyPrinter *prettyprint.PrettyPrinter,
	requiredOnly bool,
) (string, error) {
	rawParamsToInclude := m.GetNonBodyParameters()
	paramsToInclude := make(map[string]formulation.Addressable)
	for k, v := range rawParamsToInclude {
		paramsToInclude[k] = v
	}
	if requiredOnly {
		rawParamsToInclude = m.GetRequiredNonBodyParameters()
		paramsToInclude = make(map[string]formulation.Addressable)
		for k, v := range rawParamsToInclude {
			paramsToInclude[k] = v
		}
	}
	successfullyIncludedCols := make(map[string]bool)
	if !extended {
		rawParamsToInclude = m.GetRequiredParameters()
		paramsToInclude = make(map[string]formulation.Addressable)
		for k, v := range rawParamsToInclude {
			paramsToInclude[k] = v
		}
		for k := range paramsToInclude {
			if m.IsRequestBodyAttributeRenamed(k) {
				delete(paramsToInclude, k)
			}
		}
	}
	if columns != nil {
		paramsToInclude = make(map[string]formulation.Addressable)
		for _, col := range columns {
			cName := col.GetRawVal()
			if !isRequestBodyParam(cName, m) {
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
	for k := range paramsToInclude {
		includedParamNames = append(includedParamNames, k)
	}
	sort.Strings(includedParamNames)
	var columnList, exprList, jsonnetPlaholderList []string
	for _, s := range includedParamNames {
		p, ok := m.GetParameter(s)
		if !ok {
			return "", fmt.Errorf("'%s'", s)
		}
		columnList = append(columnList, prettyPrinter.RenderColumnName(s))
		switch p.GetType() {
		case strType:
			exprList = append(exprList, prettyPrinter.RenderTemplateVarAndDelimit(s))
		default:
			exprList = append(exprList, prettyPrinter.RenderTemplateVarNoDelimit(s))
		}
		jsonnetPlaholderList = append(jsonnetPlaholderList, placeHolderPrettyPrinter.RenderTemplateVarPlaceholderNoDelimit(s))
	}

	sch, err := m.GetRequestBodySchema()

	if err != nil {
		return "", err
	}

	if sch == nil {
		err = checkAllColumnsPresent(columns, successfullyIncludedCols)
		//nolint:lll // acceptable
		return "<<<jsonnet\n{\n" + strings.Join(jsonnetPlaholderList, ",\n") + "\n}\n>>>\nINSERT INTO %s" + "(\n" + strings.Join(columnList, ",\n") +
			"\n)\n" + "SELECT\n" + strings.Join(exprList, ",\n") + "\n;\n", err
	}

	//nolint:mnd // acceptable
	schemaVisitor := NewSchemaRequestTemplateVisitor(2, "", prettyPrinter, placeHolderPrettyPrinter, requiredOnly)

	tVal, _ := schemaVisitor.RetrieveTemplate(sch, m, extended)

	logging.GetLogger().Infoln(fmt.Sprintf("tVal = %v", tVal))

	colMap := getColsMap(columns)

	if columns != nil { //nolint:nestif // acceptable
		for _, c := range columns {
			cName := c.GetRawVal()
			if !isRequestBodyParam(cName, m) {
				continue
			}
			cNameSuffix, revertErr := m.RevertRequestBodyAttributeRename(cName)
			if revertErr != nil {
				return "", revertErr
			}
			if v, ok := tVal[cNameSuffix]; ok {
				columnList = append(columnList, prettyPrinter.RenderColumnName(cName))
				exprList = append(exprList, v.GetBody())
				jsonnetPlaholderList = append(jsonnetPlaholderList, v.GetPlaceholder())
				successfullyIncludedCols[cName] = true
			}
		}
	} else {
		tValKeysSorted := getSortedKeysTmplMap(tVal)
		for _, k := range tValKeysSorted {
			v := tVal[k]
			if isColIncludable(k, columns, colMap) {
				renamedPropertyKey, renamedPropertyErr := m.RenameRequestBodyAttribute(k)
				if renamedPropertyErr != nil {
					return "", renamedPropertyErr
				}
				columnList = append(columnList, prettyPrinter.RenderColumnName(
					renamedPropertyKey,
				))
				exprList = append(exprList, v.GetBody())
				jsonnetPlaholderList = append(jsonnetPlaholderList, v.GetPlaceholder())
			}
		}
	}

	err = checkAllColumnsPresent(columns, successfullyIncludedCols)
	//nolint:lll // acceptable
	retVal := "<<<jsonnet\n{\n" + strings.Join(jsonnetPlaholderList, ",\n") + "\n}\n>>>\nINSERT INTO %s" + "(\n" + strings.Join(columnList, ",\n") +
		"\n)\n" + "SELECT\n" + strings.Join(exprList, ",\n") + "\n;\n"
	return retVal, err
}

//nolint:gocognit // acceptable
func (sv *SchemaRequestTemplateVisitor) processSubSchemasMap(
	sc anysdk.Schema,
	method anysdk.OperationStore,
	properties map[string]anysdk.Schema,
) (map[string]TemplatedProduct, error) {
	retVal := make(map[string]TemplatedProduct)
	for k, ss := range properties {
		logging.GetLogger().Infoln(fmt.Sprintf("RetrieveTemplate() k = '%s', ss is nil ? '%t'", k, ss == nil))
		if ss != nil && (k == "" || !sv.isVisited(k, nil)) { //nolint:nestif // acceptable
			localSchemaVisitedMap := make(map[string]bool)
			localSchemaVisitedMap[k] = true
			if !method.IsRequiredRequestBodyProperty(k) && (ss.IsReadOnly() || (sv.requiredOnly && !sc.IsRequired(k))) {
				logging.GetLogger().Infoln(fmt.Sprintf("property = '%s' will be skipped", k))
				continue
			}
			renamedPropertyKey, renamedPropertyErr := method.RenameRequestBodyAttribute(k)
			if renamedPropertyErr != nil {
				return nil, renamedPropertyErr
			}
			rv, err := sv.retrieveTemplateVal(
				ss,
				method.GetService(),
				fmt.Sprintf(".values.%s", renamedPropertyKey),
				localSchemaVisitedMap)
			if err != nil {
				return nil, err
			}
			pl, err := sv.retrieveJsonnetPlaceholderVal(
				ss, method.GetService(), renamedPropertyKey,
				localSchemaVisitedMap)
			if err != nil {
				return nil, err
			}
			var bodyBytes, placeholderBytes string
			switch rvt := rv.(type) {
			case map[string]interface{}, []interface{}, string:
				bodyBytes, err = sv.PrettyPrinter.PrintTemplatedJSON(rvt)
				if err != nil {
					return nil, err
				}
			case nil:
				continue
			default:
				return nil, fmt.Errorf("error processing template key '%s' with disallowed type '%T'", k, rvt)
			}
			switch plt := pl.(type) {
			case map[string]interface{}, []interface{}, string:
				placeholderBytes, err = sv.PlaceholderPrettyPrinter.PrintPlaceholderJSON(plt)
				if err != nil {
					return nil, err
				}
				colName := sv.PlaceholderPrettyPrinter.RenderTemplateVarPlaceholderKeyNoDelimit(
					renamedPropertyKey,
				)
				placeholderBytes = fmt.Sprintf("%s: %s", colName, placeholderBytes)
			case nil:
				continue
			default:
				return nil, fmt.Errorf("error processing placeholder template key '%s' with disallowed type '%T'", k, plt)
			}
			retVal[k] = NewTemplatedProduct(bodyBytes, placeholderBytes)
		}
	}
	return retVal, nil
}

func (sv *SchemaRequestTemplateVisitor) RetrieveTemplate(
	sc anysdk.Schema,
	method anysdk.OperationStore,
	extended bool, //nolint:revive // TODO: review
) (map[string]TemplatedProduct, error) {
	retVal := make(map[string]TemplatedProduct) //nolint:ineffassign,staticcheck,wastedassign // TODO: review
	sv.recordSchemaVisited(sc.GetName())
	//nolint:gocritic // TODO: review
	switch sc.GetType() {
	case objectType:
		prop, err := sc.GetProperties()
		if err != nil {
			return nil, err
		}
		retVal, err = sv.processSubSchemasMap(sc, method, prop)
		if len(retVal) != 0 || err != nil {
			return retVal, err
		}
		additionalProperties, additionalProprtiesExist := sc.GetAdditionalProperties()
		if additionalProprtiesExist {
			additionalProperties.SetKey("k1")
			retVal, err = sv.processSubSchemasMap(sc, method, map[string]anysdk.Schema{"k1": additionalProperties})
		}
		if len(retVal) == 0 {
			return nil, nil //nolint:nilnil // TODO: review
		}
		return retVal, err
	}
	return nil, fmt.Errorf("templating of request body only supported for object type payload")
}

//nolint:funlen,gocognit // acceptable
func (sv *SchemaRequestTemplateVisitor) retrieveTemplateVal(
	sc anysdk.Schema,
	svc anysdk.Service, //nolint:unparam // TODO: review
	objectKey string,
	localSchemaVisitedMap map[string]bool,
) (interface{}, error) {
	sSplit := strings.Split(objectKey, ".")
	oKey := sSplit[len(sSplit)-1]
	oPrefix := objectKey //nolint:ineffassign,wastedassign // TODO: review
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
	switch sc.GetType() {
	case objectType:
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
			if ss != nil && ((ss.GetType() != arrayType) || !sv.isVisited(ss.GetTitle(), propertyLocalSchemaVisitedMap)) {
				if propertyLocalSchemaVisitedMap[ss.GetTitle()] {
					return "\"{{ " + templateValName + " }}\"", nil
				}
				propertyLocalSchemaVisitedMap[ss.GetTitle()] = true
				sv, svErr := sv.retrieveTemplateVal(ss, svc, templateValName+"."+k, propertyLocalSchemaVisitedMap)
				if svErr != nil {
					return nil, svErr
				}
				if sv != nil {
					rv[k] = sv
				}
			}
		}
		if len(rv) == 0 { //nolint:nestif // acceptable
			additionalProperties, additionalProprtiesExist := sc.GetAdditionalProperties()
			if additionalProprtiesExist {
				if additionalProperties != nil {
					hasProperties := false
					propz, pErr := additionalProperties.GetProperties()
					if pErr != nil {
						return nil, pErr
					}
					for k, v := range propz {
						hasProperties = true
						if k == "" {
							k = "key"
						}
						key := fmt.Sprintf("{{ %s[0].%s }}", templateValName, k)
						rv[key] = getAdditionalStuff(v, templateValName)
					}
					if !hasProperties {
						key := fmt.Sprintf("{{ %s[0].%s }}", templateValName, "key")
						rv[key] = getAdditionalStuff(additionalProperties, templateValName)
					}
				}
			}
		}
		if len(rv) == 0 {
			return nil, nil //nolint:nilnil // TODO: review
		}
		return rv, nil
	case arrayType:
		var arr []interface{}
		iSch, err := sc.GetItemsSchema()
		if err != nil {
			return nil, err
		}
		itemLocalSchemaVisitedMap := make(map[string]bool)
		for k, v := range initialLocalSchemaVisitedMap {
			itemLocalSchemaVisitedMap[k] = v
		}
		itemS, err := sv.retrieveTemplateVal(iSch, svc, templateValName+"[0]", itemLocalSchemaVisitedMap)
		arr = append(arr, itemS)
		if err != nil {
			return nil, err
		}
		return arr, nil
	case strType:
		return "\"{{ " + templateValName + " }}\"", nil
	default:
		return "{{ " + templateValName + " }}", nil
	}
}

//nolint:funlen,gocognit // acceptable
func (sv *SchemaRequestTemplateVisitor) retrieveJsonnetPlaceholderVal(
	sc anysdk.Schema,
	svc anysdk.Service, //nolint:unparam // TODO: review
	objectKey string,
	localSchemaVisitedMap map[string]bool,
) (interface{}, error) {
	sSplit := strings.Split(objectKey, ".")
	oKey := sSplit[len(sSplit)-1]
	oPrefix := objectKey //nolint:ineffassign,wastedassign // TODO: review
	if len(sSplit) > 1 {
		oPrefix = strings.TrimSuffix(objectKey, "."+oKey)
	} else {
		oPrefix = ""
	}
	templateValSuffix := oKey
	templateValName := oPrefix + "." + templateValSuffix
	if oPrefix == "" || true {
		templateValName = templateValSuffix
	}
	initialLocalSchemaVisitedMap := make(map[string]bool)
	for k, v := range localSchemaVisitedMap {
		initialLocalSchemaVisitedMap[k] = v
	}
	switch sc.GetType() {
	case objectType:
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
			if ss != nil && ((ss.GetType() != arrayType) || !sv.isVisited(ss.GetTitle(), propertyLocalSchemaVisitedMap)) {
				if propertyLocalSchemaVisitedMap[ss.GetTitle()] {
					return symLHS + templateValName + symRHS, nil
				}
				propertyLocalSchemaVisitedMap[ss.GetTitle()] = true
				sv, svErr := sv.retrieveJsonnetPlaceholderVal(ss, svc, templateValName+"."+k, propertyLocalSchemaVisitedMap)
				if svErr != nil {
					return nil, svErr
				}
				if sv != nil {
					rv[k] = sv
				}
			}
		}
		if len(rv) == 0 { //nolint:nestif // acceptable
			additionalProperties, additionalProprtiesExist := sc.GetAdditionalProperties()
			if additionalProprtiesExist {
				hasProperties := false
				propz, pErr := additionalProperties.GetProperties()
				if pErr != nil {
					return nil, pErr
				}
				for k, v := range propz {
					hasProperties = true
					if k == "" {
						k = "key"
					}
					key := fmt.Sprintf("<< %s[0].%s >>", templateValName, k)
					rv[key] = getAdditionalStuffPlaceholder(v, templateValName)
				}
				if !hasProperties {
					key := fmt.Sprintf("<< %s[0].%s >>", templateValName, "key")
					rv[key] = getAdditionalStuffPlaceholder(additionalProperties, templateValName)
				}
			}
		}
		if len(rv) == 0 {
			return nil, nil //nolint:nilnil // TODO: review
		}
		return rv, nil
	case arrayType:
		var arr []interface{}
		iSch, err := sc.GetItemsSchema()
		if err != nil {
			return nil, err
		}
		itemLocalSchemaVisitedMap := make(map[string]bool)
		for k, v := range initialLocalSchemaVisitedMap {
			itemLocalSchemaVisitedMap[k] = v
		}
		itemS, err := sv.retrieveJsonnetPlaceholderVal(iSch, svc, templateValName+"[0]", itemLocalSchemaVisitedMap)
		arr = append(arr, itemS)
		if err != nil {
			return nil, err
		}
		return arr, nil
	case strType:
		return symLHS + templateValName + symRHS, nil
	default:
		return symLHS + templateValName + symRHS, nil
	}
}

func getAdditionalStuff(ss anysdk.Schema, templateValName string) string {
	valBase := fmt.Sprintf("{{ %s[0].val }}", templateValName)
	switch ss.GetType() {
	case strType:
		return fmt.Sprintf(`"%s"`, valBase)
	case "number", "int", "int32", "int64":
		return valBase
	default:
		return valBase
	}
}

func getAdditionalStuffPlaceholder(ss anysdk.Schema, templateValName string) string {
	valBase := fmt.Sprintf("<< %s[0].val >>", templateValName)
	switch ss.GetType() {
	case strType:
		return valBase
	case "number", "int", "int32", "int64":
		return valBase
	default:
		return valBase
	}
}

func getSortedKeysTmplMap(m map[string]TemplatedProduct) []string {
	var retVal []string
	for k := range m {
		retVal = append(retVal, k)
	}
	sort.Strings(retVal)
	return retVal
}
