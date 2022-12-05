package internaldto

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/iqlutil"

	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_ Heirarchy            = &standardHeirarchy{}
	_ HeirarchyIdentifiers = &standardHeirarchyIdentifiers{}
	_ DBTable              = &standardDBTable{}
)

type Heirarchy interface {
	GetServiceHdl() *openapistackql.Service
	GetResource() *openapistackql.Resource
	GetMethodSet() openapistackql.MethodSet
	GetMethod() *openapistackql.OperationStore
	SetServiceHdl(*openapistackql.Service)
	SetResource(*openapistackql.Resource)
	SetMethodSet(openapistackql.MethodSet)
	SetMethod(*openapistackql.OperationStore)
	SetMethodStr(string)
}

func NewHeirarchy(hIDs HeirarchyIdentifiers) Heirarchy {
	return &standardHeirarchy{
		hIDs: hIDs,
	}
}

type standardHeirarchy struct {
	hIDs       HeirarchyIdentifiers
	serviceHdl *openapistackql.Service
	resource   *openapistackql.Resource
	methodSet  openapistackql.MethodSet
	method     *openapistackql.OperationStore
}

func (hr *standardHeirarchy) SetServiceHdl(sh *openapistackql.Service) {
	hr.serviceHdl = sh
}

func (hr *standardHeirarchy) SetResource(r *openapistackql.Resource) {
	hr.resource = r
}

func (hr *standardHeirarchy) SetMethodSet(mSet openapistackql.MethodSet) {
	hr.methodSet = mSet
}

func (hr *standardHeirarchy) SetMethodStr(mStr string) {
	hr.hIDs.SetMethodStr(mStr)
}

func (hr *standardHeirarchy) SetMethod(ost *openapistackql.OperationStore) {
	hr.method = ost
}

func (hr *standardHeirarchy) GetServiceHdl() *openapistackql.Service {
	return hr.serviceHdl
}

func (hr *standardHeirarchy) GetResource() *openapistackql.Resource {
	return hr.resource
}

func (hr *standardHeirarchy) GetMethodSet() openapistackql.MethodSet {
	return hr.methodSet
}

func (hr *standardHeirarchy) GetMethod() *openapistackql.OperationStore {
	return hr.method
}

type HeirarchyIdentifiers interface {
	GetMethodStr() string
	GetProviderStr() string
	GetServiceStr() string
	GetResourceStr() string
	GetResponseSchemaStr() string
	GetStackQLTableName() string
	GetTableName() string
	IsView() bool
	SetMethodStr(string)
	WithIsView(bool) HeirarchyIdentifiers
	WithProviderStr(string) HeirarchyIdentifiers
	WithResponseSchemaStr(rss string) HeirarchyIdentifiers
}

type standardHeirarchyIdentifiers struct {
	providerStr       string
	serviceStr        string
	resourceStr       string
	responseSchemaStr string
	methodStr         string
	isView            bool
}

func (hi *standardHeirarchyIdentifiers) SetMethodStr(mStr string) {
	hi.methodStr = mStr
}

func (hi *standardHeirarchyIdentifiers) GetProviderStr() string {
	return hi.providerStr
}

func (hi *standardHeirarchyIdentifiers) GetServiceStr() string {
	return hi.serviceStr
}

func (hi *standardHeirarchyIdentifiers) IsView() bool {
	return hi.isView
}

func (hi *standardHeirarchyIdentifiers) GetResourceStr() string {
	return hi.resourceStr
}

func (hi *standardHeirarchyIdentifiers) GetResponseSchemaStr() string {
	return hi.responseSchemaStr
}

func (hi *standardHeirarchyIdentifiers) GetMethodStr() string {
	return hi.methodStr
}

func (hi *standardHeirarchyIdentifiers) WithProviderStr(ps string) HeirarchyIdentifiers {
	hi.providerStr = ps
	return hi
}

func (hi *standardHeirarchyIdentifiers) WithIsView(isView bool) HeirarchyIdentifiers {
	hi.isView = isView
	return hi
}

func NewHeirarchyIdentifiers(prov, svc, rsc, method string) HeirarchyIdentifiers {
	return &standardHeirarchyIdentifiers{
		providerStr: prov,
		serviceStr:  svc,
		resourceStr: rsc,
		methodStr:   method,
	}
}

func (hi *standardHeirarchyIdentifiers) WithResponseSchemaStr(rss string) HeirarchyIdentifiers {
	hi.responseSchemaStr = rss
	return hi
}

func (hi *standardHeirarchyIdentifiers) GetTableName() string {
	if hi.providerStr != "" {
		if hi.responseSchemaStr == "" {
			return fmt.Sprintf("%s.%s.%s", hi.providerStr, hi.serviceStr, hi.resourceStr)
		}
		return fmt.Sprintf("%s.%s.%s.%s", hi.providerStr, hi.serviceStr, hi.resourceStr, hi.responseSchemaStr)
	}
	if hi.responseSchemaStr == "" {
		if hi.serviceStr == "" {
			return hi.resourceStr
		}
		return fmt.Sprintf("%s.%s", hi.serviceStr, hi.resourceStr)
	}
	return fmt.Sprintf("%s.%s.%s", hi.serviceStr, hi.resourceStr, hi.responseSchemaStr)
}

func (hi *standardHeirarchyIdentifiers) GetStackQLTableName() string {
	baseStr := fmt.Sprintf("%s.%s", hi.serviceStr, hi.resourceStr)
	if hi.providerStr != "" {
		baseStr = fmt.Sprintf("%s.%s", hi.providerStr, baseStr)
	}
	if hi.methodStr != "" {
		return fmt.Sprintf("%s.%s", baseStr, hi.methodStr)
	}
	return baseStr
}

func ResolveMethodTerminalHeirarchyIdentifiers(node sqlparser.TableName) HeirarchyIdentifiers {
	var retVal standardHeirarchyIdentifiers
	// all will default to empty string
	retVal.providerStr = iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierThird.String())
	retVal.serviceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierSecond.String())
	retVal.resourceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Qualifier.String())
	retVal.methodStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Name.String())
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

func ResolveResourceTerminalHeirarchyIdentifiers(node sqlparser.TableName) HeirarchyIdentifiers {
	var retVal standardHeirarchyIdentifiers
	// all will default to empty string
	retVal.providerStr = iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierSecond.String())
	retVal.serviceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Qualifier.String())
	retVal.resourceStr = iqlutil.SanitisePossibleTickEscapedTerm(node.Name.String())
	return &retVal
}

type DBTable interface {
	GetBaseName() string
	GetDiscoveryID() int
	GetHeirarchyIdentifiers() HeirarchyIdentifiers
	IsAnalytics() bool
	GetName() string
	GetNameStump() string
}

type standardDBTable struct {
	name        string
	nameStump   string
	baseName    string
	discoveryID int
	hIDs        HeirarchyIdentifiers
	namespace   string
}

func NewDBTable(name string, nameStump string, baseName string, discoveryID int, hIDs HeirarchyIdentifiers) DBTable {
	return newDBTable(name, nameStump, baseName, discoveryID, hIDs, "")
}

func NewDBTableAnalytics(name string, discoveryID int, hIDs HeirarchyIdentifiers) DBTable {
	return newDBTable(name, name, name, discoveryID, hIDs, constants.AnalyticsPrefix)
}

func newDBTable(name string, nameStump string, baseName string, discoveryID int, hIDs HeirarchyIdentifiers, namespace string) DBTable {
	return &standardDBTable{
		name:        name,
		nameStump:   nameStump,
		baseName:    baseName,
		discoveryID: discoveryID,
		hIDs:        hIDs,
		namespace:   namespace,
	}
}

func (dbt *standardDBTable) GetName() string {
	return dbt.name
}

func (dbt *standardDBTable) GetNameStump() string {
	return dbt.nameStump
}

func (dbt *standardDBTable) GetBaseName() string {
	return dbt.baseName
}

func (dbt *standardDBTable) GetDiscoveryID() int {
	return dbt.discoveryID
}

func (dbt *standardDBTable) GetHeirarchyIdentifiers() HeirarchyIdentifiers {
	return dbt.hIDs
}

func (dbt *standardDBTable) IsAnalytics() bool {
	return dbt.namespace == constants.AnalyticsPrefix
}
