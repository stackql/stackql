package internaldto

import (
	"fmt"
	"regexp"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
)

var (
	_                     HeirarchyIdentifiers = &standardHeirarchyIdentifiers{}
	pgInternalObjectRegex *regexp.Regexp       = regexp.MustCompile(`^pg_.*`) //nolint:revive // prefer declarative
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
	GetView() (RelationDTO, bool)
	GetSubAST() sqlparser.Statement
	ContainsNativeDBMSTable() bool
	SetContainsNativeDBMSTable(bool)
	SetSubAST(sqlparser.Statement)
	SetMethodStr(string)
	WithView(RelationDTO) HeirarchyIdentifiers
	withSubquery(SubqueryDTO) HeirarchyIdentifiers
	WithProviderStr(string) HeirarchyIdentifiers
	WithResponseSchemaStr(rss string) HeirarchyIdentifiers
	IsPgInternalObject() bool
	IsPhysicalTable() bool
	SetIsPhysicalTable(isPhysical bool)
	SetIsMaterializedView(isMaterialized bool)
	IsMaterializedView() bool
}

type standardHeirarchyIdentifiers struct {
	providerStr        string
	serviceStr         string
	resourceStr        string
	responseSchemaStr  string
	methodStr          string
	viewDTO            RelationDTO
	subqueryDTO        SubqueryDTO
	viewAST            sqlparser.Statement
	containsDBMSTable  bool
	isPhysicalTable    bool
	isMaterializedView bool
}

func (hi *standardHeirarchyIdentifiers) IsPhysicalTable() bool {
	return hi.isPhysicalTable
}

func (hi *standardHeirarchyIdentifiers) SetIsPhysicalTable(isPhysical bool) {
	hi.isPhysicalTable = isPhysical
}

func (hi *standardHeirarchyIdentifiers) IsMaterializedView() bool {
	return hi.isMaterializedView
}

func (hi *standardHeirarchyIdentifiers) SetIsMaterializedView(isMaterialized bool) {
	hi.isMaterializedView = isMaterialized
}

func (hi *standardHeirarchyIdentifiers) IsPgInternalObject() bool {
	isMatch := pgInternalObjectRegex.MatchString(hi.GetTableName())
	return isMatch
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

func (hi *standardHeirarchyIdentifiers) GetView() (RelationDTO, bool) {
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

func (hi *standardHeirarchyIdentifiers) WithView(viewDTO RelationDTO) HeirarchyIdentifiers {
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
