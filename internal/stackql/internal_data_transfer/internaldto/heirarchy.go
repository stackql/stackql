package internaldto

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ Heirarchy = &standardHeirarchy{}
)

type Heirarchy interface {
	GetServiceHdl() anysdk.Service
	GetResource() anysdk.Resource
	GetMethodSet() anysdk.MethodSet
	GetMethod() anysdk.OperationStore
	SetServiceHdl(anysdk.Service)
	SetResource(anysdk.Resource)
	SetMethodSet(anysdk.MethodSet)
	SetMethod(anysdk.OperationStore)
	SetMethodStr(string)
}

func NewHeirarchy(hIDs HeirarchyIdentifiers) Heirarchy {
	return &standardHeirarchy{
		hIDs: hIDs,
	}
}

type standardHeirarchy struct {
	hIDs       HeirarchyIdentifiers
	serviceHdl anysdk.Service
	resource   anysdk.Resource
	methodSet  anysdk.MethodSet
	method     anysdk.OperationStore
}

func (hr *standardHeirarchy) SetServiceHdl(sh anysdk.Service) {
	hr.serviceHdl = sh
}

func (hr *standardHeirarchy) SetResource(r anysdk.Resource) {
	hr.resource = r
}

func (hr *standardHeirarchy) SetMethodSet(mSet anysdk.MethodSet) {
	hr.methodSet = mSet
}

func (hr *standardHeirarchy) SetMethodStr(mStr string) {
	hr.hIDs.SetMethodStr(mStr)
}

func (hr *standardHeirarchy) SetMethod(ost anysdk.OperationStore) {
	hr.method = ost
}

func (hr *standardHeirarchy) GetServiceHdl() anysdk.Service {
	return hr.serviceHdl
}

func (hr *standardHeirarchy) GetResource() anysdk.Resource {
	return hr.resource
}

func (hr *standardHeirarchy) GetMethodSet() anysdk.MethodSet {
	return hr.methodSet
}

func (hr *standardHeirarchy) GetMethod() anysdk.OperationStore {
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
