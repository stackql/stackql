package internaldto

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stackql/any-sdk/public/formulation"
)

var (
	_ Heirarchy = &standardHeirarchy{}
)

type Heirarchy interface {
	GetServiceHdl() formulation.Service
	GetResource() formulation.Resource
	GetMethodSet() formulation.MethodSet
	GetMethod() formulation.StandardOperationStore
	SetServiceHdl(formulation.Service)
	SetResource(formulation.Resource)
	SetMethodSet(formulation.MethodSet)
	SetMethod(formulation.StandardOperationStore)
	SetMethodStr(string)
}

func NewHeirarchy(hIDs HeirarchyIdentifiers) Heirarchy {
	return &standardHeirarchy{
		hIDs: hIDs,
	}
}

type standardHeirarchy struct {
	hIDs       HeirarchyIdentifiers
	serviceHdl formulation.Service
	resource   formulation.Resource
	methodSet  formulation.MethodSet
	method     formulation.StandardOperationStore
}

func (hr *standardHeirarchy) SetServiceHdl(sh formulation.Service) {
	hr.serviceHdl = sh
}

func (hr *standardHeirarchy) SetResource(r formulation.Resource) {
	hr.resource = r
}

func (hr *standardHeirarchy) SetMethodSet(mSet formulation.MethodSet) {
	hr.methodSet = mSet
}

func (hr *standardHeirarchy) SetMethodStr(mStr string) {
	hr.hIDs.SetMethodStr(mStr)
}

func (hr *standardHeirarchy) SetMethod(ost formulation.StandardOperationStore) {
	hr.method = ost
}

func (hr *standardHeirarchy) GetServiceHdl() formulation.Service {
	return hr.serviceHdl
}

func (hr *standardHeirarchy) GetResource() formulation.Resource {
	return hr.resource
}

func (hr *standardHeirarchy) GetMethodSet() formulation.MethodSet {
	return hr.methodSet
}

func (hr *standardHeirarchy) GetMethod() formulation.StandardOperationStore {
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
