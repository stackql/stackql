package internaldto

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_ Heirarchy = &standardHeirarchy{}
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
