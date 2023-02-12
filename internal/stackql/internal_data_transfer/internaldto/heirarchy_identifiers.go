package internaldto

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
)

var (
	_ HeirarchyIdentifiers = &standardHeirarchyIdentifiers{}
)

type HeirarchyIdentifiers interface {
	GetMethodStr() string
	GetProviderStr() string
	GetServiceStr() string
	GetResourceStr() string
	GetResponseSchemaStr() string
	GetSQLDataSourceTableName() string
	GetStackQLTableName() string
	GetTableName() string
	GetSubquery() (SubqueryDTO, bool)
	GetView() (ViewDTO, bool)
	GetSubAST() sqlparser.Statement
	ContainsNativeDBMSTable() bool
	SetContainsNativeDBMSTable(bool)
	SetSubAST(sqlparser.Statement)
	SetMethodStr(string)
	WithView(ViewDTO) HeirarchyIdentifiers
	withSubquery(SubqueryDTO) HeirarchyIdentifiers
	WithProviderStr(string) HeirarchyIdentifiers
	WithResponseSchemaStr(rss string) HeirarchyIdentifiers
}

type standardHeirarchyIdentifiers struct {
	providerStr       string
	serviceStr        string
	resourceStr       string
	responseSchemaStr string
	methodStr         string
	viewDTO           ViewDTO
	subqueryDTO       SubqueryDTO
	viewAST           sqlparser.Statement
	containsDBMSTable bool
}

func (hi *standardHeirarchyIdentifiers) SetMethodStr(mStr string) {
	hi.methodStr = mStr
}

func (hi *standardHeirarchyIdentifiers) ContainsNativeDBMSTable() bool {
	return hi.containsDBMSTable
}

func (hi *standardHeirarchyIdentifiers) SetContainsNativeDBMSTable(containsDBMSTable bool) {
	hi.containsDBMSTable = containsDBMSTable
}

func (hi *standardHeirarchyIdentifiers) SetSubAST(viewAST sqlparser.Statement) {
	hi.viewAST = viewAST
}

func (hi *standardHeirarchyIdentifiers) GetSubAST() sqlparser.Statement {
	return hi.viewAST
}

func (hi *standardHeirarchyIdentifiers) GetProviderStr() string {
	return hi.providerStr
}

func (hi *standardHeirarchyIdentifiers) GetServiceStr() string {
	return hi.serviceStr
}

func (hi *standardHeirarchyIdentifiers) GetView() (ViewDTO, bool) {
	return hi.viewDTO, hi.viewDTO != nil
}

func (hi *standardHeirarchyIdentifiers) GetSubquery() (SubqueryDTO, bool) {
	return hi.subqueryDTO, hi.subqueryDTO != nil
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

func (hi *standardHeirarchyIdentifiers) WithView(viewDTO ViewDTO) HeirarchyIdentifiers {
	hi.viewDTO = viewDTO
	return hi
}

func (hi *standardHeirarchyIdentifiers) withSubquery(subQuery SubqueryDTO) HeirarchyIdentifiers {
	hi.subqueryDTO = subQuery
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

func (hi *standardHeirarchyIdentifiers) GetSQLDataSourceTableName() string {
	baseStr := hi.serviceStr
	if hi.resourceStr != "" {
		return fmt.Sprintf("%s.%s", baseStr, hi.resourceStr)
	}
	return baseStr
}

func ResolveMethodTerminalHeirarchyIdentifiers(node sqlparser.TableName) HeirarchyIdentifiers {
	return NewHeirarchyIdentifiers(
		iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierThird.String()),
		iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierSecond.String()),
		iqlutil.SanitisePossibleTickEscapedTerm(node.Qualifier.String()),
		iqlutil.SanitisePossibleTickEscapedTerm(node.Name.String()),
	)
}

func ResolveResourceTerminalHeirarchyIdentifiers(node sqlparser.TableName) HeirarchyIdentifiers {
	return NewHeirarchyIdentifiers(
		iqlutil.SanitisePossibleTickEscapedTerm(node.QualifierSecond.String()),
		iqlutil.SanitisePossibleTickEscapedTerm(node.Qualifier.String()),
		iqlutil.SanitisePossibleTickEscapedTerm(node.Name.String()),
		"",
	)
}

func ObtainSubqueryHeirarchyIdentifiers(subQuery SubqueryDTO) HeirarchyIdentifiers {
	return NewHeirarchyIdentifiers(
		"",
		"",
		"",
		"",
	).withSubquery(subQuery)
}
