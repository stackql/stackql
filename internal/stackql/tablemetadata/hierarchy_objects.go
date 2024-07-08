package tablemetadata

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

var (
	_ HeirarchyObjects = &standardHeirarchyObjects{}
)

type HeirarchyObjects interface {
	GetHeirarchyIDs() internaldto.HeirarchyIdentifiers
	GetObjectSchema() (anysdk.Schema, error)
	GetProvider() provider.IProvider
	GetRequestSchema() (anysdk.Schema, error)
	GetResponseSchemaAndMediaType() (anysdk.Schema, string, error)
	GetSelectableObjectSchema() (anysdk.Schema, error)
	GetSelectSchemaAndObjectPath() (anysdk.Schema, string, error)
	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetTableName() string
	GetSubquery() (internaldto.SubqueryDTO, bool)
	GetView() (internaldto.RelationDTO, bool)
	LookupSelectItemsKey() string
	SetProvider(provider.IProvider)
	SetSQLDataSource(sql_datasource.SQLDataSource)
	// De facto inheritance
	GetServiceHdl() anysdk.Service
	GetResource() anysdk.Resource
	GetMethodSet() anysdk.MethodSet
	GetMethod() anysdk.OperationStore
	SetMethod(anysdk.OperationStore)
	SetMethodSet(anysdk.MethodSet)
	SetMethodStr(string)
	SetResource(anysdk.Resource)
	SetServiceHdl(anysdk.Service)
	IsPGInternalObject() bool
	SetIndirect(internaldto.RelationDTO)
	GetIndirect() (internaldto.RelationDTO, bool)
}

func NewHeirarchyObjects(hIDs internaldto.HeirarchyIdentifiers) HeirarchyObjects {
	return &standardHeirarchyObjects{
		heirarchyIDs: hIDs,
		hr:           internaldto.NewHeirarchy(hIDs),
	}
}

type standardHeirarchyObjects struct {
	hr            internaldto.Heirarchy
	heirarchyIDs  internaldto.HeirarchyIdentifiers
	prov          provider.IProvider
	sqlDataSource sql_datasource.SQLDataSource
	indirect      internaldto.RelationDTO
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

func (ho *standardHeirarchyObjects) GetServiceHdl() anysdk.Service {
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

func (ho *standardHeirarchyObjects) GetResource() anysdk.Resource {
	return ho.hr.GetResource()
}

func (ho *standardHeirarchyObjects) GetMethodSet() anysdk.MethodSet {
	return ho.hr.GetMethodSet()
}

func (ho *standardHeirarchyObjects) GetMethod() anysdk.OperationStore {
	return ho.hr.GetMethod()
}

func (ho *standardHeirarchyObjects) SetServiceHdl(sh anysdk.Service) {
	ho.hr.SetServiceHdl(sh)
}

func (ho *standardHeirarchyObjects) SetResource(r anysdk.Resource) {
	ho.hr.SetResource(r)
}

func (ho *standardHeirarchyObjects) SetMethodSet(mSet anysdk.MethodSet) {
	ho.hr.SetMethodSet(mSet)
}

func (ho *standardHeirarchyObjects) SetMethod(m anysdk.OperationStore) {
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

func LookupSelectItemsKey(method anysdk.OperationStore) string {
	return lookupSelectItemsKey(method)
}

func lookupSelectItemsKey(method anysdk.OperationStore) string {
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
		return anysdk.AnonymousColumnName
	}
	return defaultSelectItemsKey
}

func (ho *standardHeirarchyObjects) GetResponseSchemaAndMediaType() (anysdk.Schema, string, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetResponseBodySchemaAndMediaType()
}

func (ho *standardHeirarchyObjects) GetSelectSchemaAndObjectPath() (anysdk.Schema, string, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetSelectSchemaAndObjectPath()
}

func (ho *standardHeirarchyObjects) GetRequestSchema() (anysdk.Schema, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return ho.GetRequestSchema()
}

func (ho *standardHeirarchyObjects) GetTableName() string {
	return ho.heirarchyIDs.GetTableName()
}

func (ho *standardHeirarchyObjects) GetObjectSchema() (anysdk.Schema, error) {
	return ho.getObjectSchema()
}

func (ho *standardHeirarchyObjects) getObjectSchema() (anysdk.Schema, error) {
	rv, _, err := ho.GetMethod().GetResponseBodySchemaAndMediaType()
	return rv, err
}

func (ho *standardHeirarchyObjects) GetSelectableObjectSchema() (anysdk.Schema, error) {
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
