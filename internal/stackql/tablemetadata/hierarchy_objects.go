package tablemetadata

import (
	"fmt"

	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

var (
	_ HeirarchyObjects = &standardHeirarchyObjects{}
)

type HeirarchyObjects interface {
	GetHeirarchyIDs() internaldto.HeirarchyIdentifiers
	GetObjectSchema() (formulation.Schema, error)
	GetProvider() provider.IProvider
	GetRequestSchema() (formulation.Schema, error)
	GetResponseSchemaAndMediaType() (formulation.Schema, string, error)
	GetSelectableObjectSchema() (formulation.Schema, error)
	GetSelectSchemaAndObjectPath() (formulation.Schema, string, error)
	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetTableName() string
	GetSubquery() (internaldto.SubqueryDTO, bool)
	GetView() (internaldto.RelationDTO, bool)
	LookupSelectItemsKey() string
	SetProvider(provider.IProvider)
	SetSQLDataSource(sql_datasource.SQLDataSource)
	// De facto inheritance
	GetServiceHdl() formulation.Service
	GetResource() formulation.Resource
	GetMethodSet() formulation.MethodSet
	GetMethod() formulation.StandardOperationStore
	SetMethod(formulation.StandardOperationStore)
	SetMethodSet(formulation.MethodSet)
	SetMethodStr(string)
	SetResource(formulation.Resource)
	SetServiceHdl(formulation.Service)
	IsPGInternalObject() bool
	SetIndirect(internaldto.RelationDTO)
	GetIndirect() (internaldto.RelationDTO, bool)
	IsAwait() bool
}

func NewHeirarchyObjects(hIDs internaldto.HeirarchyIdentifiers, isAwait bool) HeirarchyObjects {
	return &standardHeirarchyObjects{
		heirarchyIDs: hIDs,
		hr:           internaldto.NewHeirarchy(hIDs),
		isAwait:      isAwait,
	}
}

type standardHeirarchyObjects struct {
	hr            internaldto.Heirarchy
	heirarchyIDs  internaldto.HeirarchyIdentifiers
	prov          provider.IProvider
	sqlDataSource sql_datasource.SQLDataSource
	indirect      internaldto.RelationDTO
	isAwait       bool
}

func (ho *standardHeirarchyObjects) IsAwait() bool {
	return ho.isAwait
}

func (ho *standardHeirarchyObjects) GetIndirect() (internaldto.RelationDTO, bool) {
	return ho.indirect, ho.indirect != nil
}

func (ho *standardHeirarchyObjects) SetIndirect(indirect internaldto.RelationDTO) {
	ho.indirect = indirect
}

func (ho *standardHeirarchyObjects) IsPGInternalObject() bool {
	return ho.heirarchyIDs.IsPgInternalObject()
}

func (ho *standardHeirarchyObjects) GetServiceHdl() formulation.Service {
	return ho.hr.GetServiceHdl()
}

func (ho *standardHeirarchyObjects) GetSQLDataSource() (sql_datasource.SQLDataSource, bool) {
	return ho.sqlDataSource, ho.sqlDataSource != nil
}

func (ho *standardHeirarchyObjects) SetSQLDataSource(sqlDataSource sql_datasource.SQLDataSource) {
	ho.sqlDataSource = sqlDataSource
}

func (ho *standardHeirarchyObjects) GetView() (internaldto.RelationDTO, bool) {
	return ho.heirarchyIDs.GetView()
}

func (ho *standardHeirarchyObjects) GetSubquery() (internaldto.SubqueryDTO, bool) {
	return ho.heirarchyIDs.GetSubquery()
}

func (ho *standardHeirarchyObjects) GetResource() formulation.Resource {
	return ho.hr.GetResource()
}

func (ho *standardHeirarchyObjects) GetMethodSet() formulation.MethodSet {
	return ho.hr.GetMethodSet()
}

func (ho *standardHeirarchyObjects) GetMethod() formulation.StandardOperationStore {
	return ho.hr.GetMethod()
}

func (ho *standardHeirarchyObjects) SetServiceHdl(sh formulation.Service) {
	ho.hr.SetServiceHdl(sh)
}

func (ho *standardHeirarchyObjects) SetResource(r formulation.Resource) {
	ho.hr.SetResource(r)
}

func (ho *standardHeirarchyObjects) SetMethodSet(mSet formulation.MethodSet) {
	ho.hr.SetMethodSet(mSet)
}

func (ho *standardHeirarchyObjects) SetMethod(m formulation.StandardOperationStore) {
	ho.hr.SetMethod(m)
}

func (ho *standardHeirarchyObjects) SetProvider(prov provider.IProvider) {
	ho.prov = prov
}

func (ho *standardHeirarchyObjects) GetProvider() provider.IProvider {
	return ho.prov
}

func (ho *standardHeirarchyObjects) GetHeirarchyIDs() internaldto.HeirarchyIdentifiers {
	return ho.heirarchyIDs
}

func (ho *standardHeirarchyObjects) SetMethodStr(mStr string) {
	ho.hr.SetMethodStr(mStr)
}

func (ho *standardHeirarchyObjects) LookupSelectItemsKey() string {
	method := ho.GetMethod()
	return lookupSelectItemsKey(method)
}

func LookupSelectItemsKey(method formulation.OperationStore) string {
	return lookupSelectItemsKey(method)
}

func lookupSelectItemsKey(method formulation.OperationStore) string {
	if method == nil {
		return defaultSelectItemsKey
	}
	if sk := method.GetSelectItemsKey(); sk != "" {
		return sk
	}
	responseSchema, _, err := method.GetResponseBodySchemaAndMediaType()
	if responseSchema == nil || err != nil {
		return ""
	}
	switch responseSchema.GetType() {
	case "string", "integer":
		return formulation.AnonymousColumnName
	}
	return defaultSelectItemsKey
}

func (ho *standardHeirarchyObjects) GetResponseSchemaAndMediaType() (formulation.Schema, string, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetResponseBodySchemaAndMediaType()
}

func (ho *standardHeirarchyObjects) GetSelectSchemaAndObjectPath() (formulation.Schema, string, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetSelectSchemaAndObjectPath()
}

func (ho *standardHeirarchyObjects) GetRequestSchema() (formulation.Schema, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return ho.GetRequestSchema()
}

func (ho *standardHeirarchyObjects) GetTableName() string {
	return ho.heirarchyIDs.GetTableName()
}

func (ho *standardHeirarchyObjects) GetObjectSchema() (formulation.Schema, error) {
	return ho.getObjectSchema()
}

func (ho *standardHeirarchyObjects) getObjectSchema() (formulation.Schema, error) {
	rv, _, err := ho.GetMethod().GetResponseBodySchemaAndMediaType()
	return rv, err
}

func (ho *standardHeirarchyObjects) GetSelectableObjectSchema() (formulation.Schema, error) {
	unsuitableSchemaMsg := "GetSelectableObjectSchema(): schema unsuitable for select query"
	itemObjS, _, err := ho.GetMethod().GetSelectSchemaAndObjectPath()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, unsuitableSchemaMsg)
	}
	if itemObjS == nil {
		m, ok := ho.GetMethod().GetResponse()
		ts := "<unknown>"
		if ok {
			ts = fmt.Sprintf("'%T'", m.GetObjectKey())
		}
		return nil, fmt.Errorf("could not locate dml object for response type %s", ts)
	}
	return itemObjS, nil
}
