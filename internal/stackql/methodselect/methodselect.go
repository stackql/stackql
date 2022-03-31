package methodselect

import (
	"fmt"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type IMethodSelector interface {
	GetMethod(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error)

	GetMethodForAction(resource *openapistackql.Resource, iqlAction string, parameters map[string]interface{}) (*openapistackql.OperationStore, string, error)
}

func NewMethodSelector(provider string, version string) (IMethodSelector, error) {
	switch provider {
	default:
		return newGoogleMethodSelector(version)
	}
}

func newGoogleMethodSelector(version string) (IMethodSelector, error) {
	switch version {
	default:
		return &DefaultMethodSelector{}, nil
	}
}

type DefaultMethodSelector struct {
}

func (sel *DefaultMethodSelector) GetMethodForAction(resource *openapistackql.Resource, iqlAction string, parameters map[string]interface{}) (*openapistackql.OperationStore, string, error) {
	var methodName string
	switch strings.ToLower(iqlAction) {
	case "select":
		methodName = "select"
	case "delete":
		methodName = "delete"
	case "insert":
		methodName = "insert"
		m, err := resource.FindMethod(methodName)
		if err == nil {
			return m, methodName, nil
		}
		methodName = "create"
		m, err = resource.FindMethod(methodName)
		if err == nil {
			return m, methodName, nil
		}
		return nil, "", fmt.Errorf("iql action = '%s' curently not supported, there is no method mapping possible for any resource", iqlAction)
	default:
		return nil, "", fmt.Errorf("iql action = '%s' curently not supported, there is no method mapping possible for any resource", iqlAction)
	}
	m, err := sel.getMethodByNameAndParameters(resource, methodName, parameters)
	return m, methodName, err
}

func (sel *DefaultMethodSelector) GetMethod(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error) {
	return sel.getMethodByName(resource, methodName)
}

func (sel *DefaultMethodSelector) getMethodByName(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error) {
	m, err := resource.FindMethod(methodName)
	if err != nil {
		return nil, fmt.Errorf("no method = '%s' for resource = '%s'", methodName, resource.Name)
	}
	return m, nil
}

func (sel *DefaultMethodSelector) getMethodByNameAndParameters(resource *openapistackql.Resource, methodName string, parameters map[string]interface{}) (*openapistackql.OperationStore, error) {
	m, ok := resource.GetFirstMethodMatchFromSQLVerb(methodName, parameters)
	if !ok {
		return nil, fmt.Errorf("no appropriate method = '%s' for resource = '%s'", methodName, resource.Name)
	}
	return m, nil
}
