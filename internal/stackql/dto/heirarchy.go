package dto

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/iqlutil"

	"vitess.io/vitess/go/vt/sqlparser"
)

type HeirarchyIdentifiers struct {
	ProviderStr string
	ServiceStr  string
	ResourceStr string
	MethodStr   string
}

func NewHeirarchyIdentifiers(prov, svc, rsc, method string) *HeirarchyIdentifiers {
	return &HeirarchyIdentifiers{
		ProviderStr: prov,
		ServiceStr:  svc,
		ResourceStr: rsc,
		MethodStr:   method,
	}
}

func (hi *HeirarchyIdentifiers) GetTableName() string {
	if hi.ProviderStr != "" {
		return fmt.Sprintf("%s.%s.%s", hi.ProviderStr, hi.ServiceStr, hi.ResourceStr)
	}
	return fmt.Sprintf("%s.%s", hi.ServiceStr, hi.ResourceStr)
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
	discoveryID int
	hIDs        *HeirarchyIdentifiers
}

func NewDBTable(name string, discoveryID int, hIDs *HeirarchyIdentifiers) DBTable {
	return DBTable{
		name:        name,
		discoveryID: discoveryID,
		hIDs:        hIDs,
	}
}

func (dbt DBTable) GetName() string {
	return dbt.name
}

func (dbt DBTable) GetDiscoveryID() int {
	return dbt.discoveryID
}

func (dbt DBTable) GetHeirarchyIdentifiers() *HeirarchyIdentifiers {
	return dbt.hIDs
}
