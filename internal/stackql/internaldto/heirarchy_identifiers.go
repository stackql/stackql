package internaldto

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/iqlutil"
	"vitess.io/vitess/go/vt/sqlparser"
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
	GetStackQLTableName() string
	GetTableName() string
	GetView() (ViewDTO, bool)
	GetSubAST() sqlparser.Statement
	SetSubAST(sqlparser.Statement)
	SetMethodStr(string)
	WithView(ViewDTO) HeirarchyIdentifiers
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
	viewAST           sqlparser.Statement
}

func (hi *standardHeirarchyIdentifiers) SetMethodStr(mStr string) {
	hi.methodStr = mStr
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
