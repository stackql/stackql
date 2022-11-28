package dto

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/iqlutil"

	"vitess.io/vitess/go/vt/sqlparser"
)

type Heirarchy struct {
	ServiceHdl *openapistackql.Service
	Resource   *openapistackql.Resource
	MethodSet  openapistackql.MethodSet
	Method     *openapistackql.OperationStore
}

type HeirarchyIdentifiers struct {
	ProviderStr       string
	ServiceStr        string
	ResourceStr       string
	ResponseSchemaStr string
	MethodStr         string
}

func NewHeirarchyIdentifiers(prov, svc, rsc, method string) *HeirarchyIdentifiers {
	return &HeirarchyIdentifiers{
		ProviderStr: prov,
		ServiceStr:  svc,
		ResourceStr: rsc,
		MethodStr:   method,
	}
}

func (hi *HeirarchyIdentifiers) WithResponseSchemaStr(rss string) *HeirarchyIdentifiers {
	hi.ResponseSchemaStr = rss
	return hi
}

func (hi *HeirarchyIdentifiers) GetTableName() string {
	if hi.ProviderStr != "" {
		if hi.ResponseSchemaStr == "" {
			return fmt.Sprintf("%s.%s.%s", hi.ProviderStr, hi.ServiceStr, hi.ResourceStr)
		}
		return fmt.Sprintf("%s.%s.%s.%s", hi.ProviderStr, hi.ServiceStr, hi.ResourceStr, hi.ResponseSchemaStr)
	}
	if hi.ResponseSchemaStr == "" {
		return fmt.Sprintf("%s.%s", hi.ServiceStr, hi.ResourceStr)
	}
	return fmt.Sprintf("%s.%s.%s", hi.ServiceStr, hi.ResourceStr, hi.ResponseSchemaStr)
}

func (hi *HeirarchyIdentifiers) GetCompressedTableName() string {
	if hi.ProviderStr != "" {
		if hi.ResponseSchemaStr == "" {
			return fmt.Sprintf("%s.%s.%s", hi.ProviderStr, hi.ServiceStr, hi.ResourceStr)
		}
		return fmt.Sprintf("%s.%s.%s.%s", hi.ProviderStr, hi.ServiceStr, hi.ResourceStr, hi.ResponseSchemaStr)
	}
	if hi.ResponseSchemaStr == "" {
		return fmt.Sprintf("%s.%s", hi.ServiceStr, hi.ResourceStr)
	}
	return fmt.Sprintf("%s.%s.%s", hi.ServiceStr, hi.ResourceStr, hi.ResponseSchemaStr)
}

func (hi *HeirarchyIdentifiers) GetStackQLTableName() string {
	baseStr := fmt.Sprintf("%s.%s", hi.ServiceStr, hi.ResourceStr)
	if hi.ProviderStr != "" {
		baseStr = fmt.Sprintf("%s.%s", hi.ProviderStr, baseStr)
	}
	if hi.MethodStr != "" {
		return fmt.Sprintf("%s.%s", baseStr, hi.MethodStr)
	}
	return baseStr
}

func ResolveMethodTerminalHeirarchyIdentifiers(node sqlparser.TableName) *HeirarchyIdentifiers {
	var retVal HeirarchyIdentifiers
	// all will default to empty string
	retVal.ProviderStr = iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierThird.String())
	retVal.ServiceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierSecond.String())
	retVal.ResourceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Qualifier.String())
	retVal.MethodStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Name.String())
	return &retVal
}

func generatePutativelyUniqueTableName(node sqlparser.TableName) string {
	if node.IsEmpty() {
		return ""
	}
	retVal := ""
	if !node.QualifierThird.IsEmpty() {
		retVal += fmt.Sprintf("%s.", node.QualifierThird.GetRawVal())
	}
	if !node.QualifierSecond.IsEmpty() {
		retVal += fmt.Sprintf("%s.", node.QualifierSecond.GetRawVal())
	}
	if !node.Qualifier.IsEmpty() {
		retVal += fmt.Sprintf("%s.", node.Qualifier.GetRawVal())
	}
	retVal += node.Name.GetRawVal()
	return retVal
}

func GeneratePutativelyUniqueColumnID(node sqlparser.TableName, colName string) string {
	tableID := generatePutativelyUniqueTableName(node)
	if tableID == "" {
		return colName
	}
	return fmt.Sprintf("%s.%s", tableID, colName)
}

func ResolveResourceTerminalHeirarchyIdentifiers(node sqlparser.TableName) *HeirarchyIdentifiers {
	var retVal HeirarchyIdentifiers
	// all will default to empty string
	retVal.ProviderStr = iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierSecond.String())
	retVal.ServiceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Qualifier.String())
	retVal.ResourceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Name.String())
	return &retVal
}

type DBTable struct {
	name        string
	nameStump   string
	baseName    string
	discoveryID int
	hIDs        *HeirarchyIdentifiers
	namespace   string
}

func NewDBTable(name string, nameStump string, baseName string, discoveryID int, hIDs *HeirarchyIdentifiers) DBTable {
	return newDBTable(name, nameStump, baseName, discoveryID, hIDs, "")
}

func NewDBTableAnalytics(name string, discoveryID int, hIDs *HeirarchyIdentifiers) DBTable {
	return newDBTable(name, name, name, discoveryID, hIDs, constants.AnalyticsPrefix)
}

func newDBTable(name string, nameStump string, baseName string, discoveryID int, hIDs *HeirarchyIdentifiers, namespace string) DBTable {
	return DBTable{
		name:        name,
		nameStump:   nameStump,
		baseName:    baseName,
		discoveryID: discoveryID,
		hIDs:        hIDs,
		namespace:   namespace,
	}
}

func (dbt DBTable) GetName() string {
	return dbt.name
}

func (dbt DBTable) GetNameStump() string {
	return dbt.nameStump
}

func (dbt DBTable) GetBaseName() string {
	return dbt.baseName
}

func (dbt DBTable) GetDiscoveryID() int {
	return dbt.discoveryID
}

func (dbt DBTable) GetHeirarchyIdentifiers() *HeirarchyIdentifiers {
	return dbt.hIDs
}

func (dbt DBTable) IsAnalytics() bool {
	return dbt.namespace == constants.AnalyticsPrefix
}
