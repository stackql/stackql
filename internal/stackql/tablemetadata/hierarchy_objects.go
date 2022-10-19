package tablemetadata

import (
	"fmt"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type HeirarchyObjects struct {
	dto.Heirarchy
	HeirarchyIds dto.HeirarchyIdentifiers
	Provider     provider.IProvider
}

func (ho *HeirarchyObjects) LookupSelectItemsKey() string {
	method := ho.Method
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

func (ho *HeirarchyObjects) GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error) {
	m := ho.Method
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetResponseBodySchemaAndMediaType()
}

func (ho *HeirarchyObjects) GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error) {
	m := ho.Method
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetSelectSchemaAndObjectPath()
}

func (ho *HeirarchyObjects) GetRequestSchema() (*openapistackql.Schema, error) {
	m := ho.Method
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return ho.GetRequestSchema()
}

func (ho *HeirarchyObjects) GetTableName() string {
	return ho.HeirarchyIds.GetTableName()
}

func (ho *HeirarchyObjects) GetObjectSchema() (*openapistackql.Schema, error) {
	return ho.getObjectSchema()
}

func (ho *HeirarchyObjects) getObjectSchema() (*openapistackql.Schema, error) {
	rv, _, err := ho.Method.GetResponseBodySchemaAndMediaType()
	return rv, err
}

func (ho *HeirarchyObjects) GetSelectableObjectSchema() (*openapistackql.Schema, error) {
	unsuitableSchemaMsg := "GetSelectableObjectSchema(): schema unsuitable for select query"
	itemObjS, _, err := ho.Method.GetSelectSchemaAndObjectPath()
	// rscStr, _ := tbl.GetResourceStr()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), unsuitableSchemaMsg)
	}
	if itemObjS == nil || err != nil {
		return nil, fmt.Errorf("could not locate dml object for response type '%v'", ho.Method.Response.ObjectKey)
	}
	return itemObjS, nil
}
