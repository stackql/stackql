package tablemetadata

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

var (
	_ HeirarchyObjects = &standardHeirarchyObjects{}
)

type HeirarchyObjects interface {
	GetHeirarchyIds() internaldto.HeirarchyIdentifiers
	GetObjectSchema() (*openapistackql.Schema, error)
	GetProvider() provider.IProvider
	GetRequestSchema() (*openapistackql.Schema, error)
	GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error)
	GetSelectableObjectSchema() (*openapistackql.Schema, error)
	GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error)
	GetTableName() string
	GetView() (internaldto.ViewDTO, bool)
	LookupSelectItemsKey() string
	SetProvider(provider.IProvider)
	// De facto inheritance
	GetServiceHdl() *openapistackql.Service
	GetResource() *openapistackql.Resource
	GetMethodSet() openapistackql.MethodSet
	GetMethod() *openapistackql.OperationStore
	SetMethod(*openapistackql.OperationStore)
	SetMethodSet(openapistackql.MethodSet)
	SetMethodStr(string)
	SetResource(*openapistackql.Resource)
	SetServiceHdl(*openapistackql.Service)
}

func NewHeirarchyObjects(hIDs internaldto.HeirarchyIdentifiers) HeirarchyObjects {
	return &standardHeirarchyObjects{
		heirarchyIds: hIDs,
		hr:           internaldto.NewHeirarchy(hIDs),
	}
}

type standardHeirarchyObjects struct {
	hr           internaldto.Heirarchy
	heirarchyIds internaldto.HeirarchyIdentifiers
	prov         provider.IProvider
}

func (ho *standardHeirarchyObjects) GetServiceHdl() *openapistackql.Service {
	return ho.hr.GetServiceHdl()
}

func (ho *standardHeirarchyObjects) GetView() (internaldto.ViewDTO, bool) {
	return ho.heirarchyIds.GetView()
}

func (ho *standardHeirarchyObjects) GetResource() *openapistackql.Resource {
	return ho.hr.GetResource()
}

func (ho *standardHeirarchyObjects) GetMethodSet() openapistackql.MethodSet {
	return ho.hr.GetMethodSet()
}

func (ho *standardHeirarchyObjects) GetMethod() *openapistackql.OperationStore {
	return ho.hr.GetMethod()
}

func (ho *standardHeirarchyObjects) SetServiceHdl(sh *openapistackql.Service) {
	ho.hr.SetServiceHdl(sh)
}

func (ho *standardHeirarchyObjects) SetResource(r *openapistackql.Resource) {
	ho.hr.SetResource(r)
}

func (ho *standardHeirarchyObjects) SetMethodSet(mSet openapistackql.MethodSet) {
	ho.hr.SetMethodSet(mSet)
}

func (ho *standardHeirarchyObjects) SetMethod(m *openapistackql.OperationStore) {
	ho.hr.SetMethod(m)
}

func (ho *standardHeirarchyObjects) SetProvider(prov provider.IProvider) {
	ho.prov = prov
}

func (ho *standardHeirarchyObjects) GetProvider() provider.IProvider {
	return ho.prov
}

func (ho *standardHeirarchyObjects) GetHeirarchyIds() internaldto.HeirarchyIdentifiers {
	return ho.heirarchyIds
}

func (ho *standardHeirarchyObjects) SetMethodStr(mStr string) {
	ho.hr.SetMethodStr(mStr)
}

func (ho *standardHeirarchyObjects) LookupSelectItemsKey() string {
	method := ho.GetMethod()
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
	switch responseSchema.Type {
	case "string", "integer":
		return openapistackql.AnonymousColumnName
	}
	return defaultSelectItemsKey
}

func (ho *standardHeirarchyObjects) GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetResponseBodySchemaAndMediaType()
}

func (ho *standardHeirarchyObjects) GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetSelectSchemaAndObjectPath()
}

func (ho *standardHeirarchyObjects) GetRequestSchema() (*openapistackql.Schema, error) {
	m := ho.GetMethod()
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return ho.GetRequestSchema()
}

func (ho *standardHeirarchyObjects) GetTableName() string {
	return ho.heirarchyIds.GetTableName()
}

func (ho *standardHeirarchyObjects) GetObjectSchema() (*openapistackql.Schema, error) {
	return ho.getObjectSchema()
}

func (ho *standardHeirarchyObjects) getObjectSchema() (*openapistackql.Schema, error) {
	rv, _, err := ho.GetMethod().GetResponseBodySchemaAndMediaType()
	return rv, err
}

func (ho *standardHeirarchyObjects) GetSelectableObjectSchema() (*openapistackql.Schema, error) {
	unsuitableSchemaMsg := "GetSelectableObjectSchema(): schema unsuitable for select query"
	itemObjS, _, err := ho.GetMethod().GetSelectSchemaAndObjectPath()
	// rscStr, _ := tbl.GetResourceStr()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), unsuitableSchemaMsg)
	}
	if itemObjS == nil || err != nil {
		return nil, fmt.Errorf("could not locate dml object for response type '%v'", ho.GetMethod().Response.ObjectKey)
	}
	return itemObjS, nil
}
