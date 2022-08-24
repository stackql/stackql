package methodselect

import (
	"fmt"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type IMethodSelector interface {
	GetMethod(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error)

	GetMethodForAction(resource *openapistackql.Resource, iqlAction string, parameters map[string]interface{}) (*openapistackql.OperationStore, string, map[string]interface{}, error)
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

func (sel *DefaultMethodSelector) GetMethodForAction(resource *openapistackql.Resource, iqlAction string, parameters map[string]interface{}) (*openapistackql.OperationStore, string, map[string]interface{}, error) {
	var methodName string
	switch strings.ToLower(iqlAction) {
	case "select":
		methodName = "select"
	case "delete":
		methodName = "delete"
	case "insert":
		methodName = "insert"
	case "update":
		methodName = "update"
	default:
		return nil, "", parameters, fmt.Errorf("iql action = '%s' curently not supported, there is no method mapping possible for any resource", iqlAction)
	}
	m, remainingParams, err := sel.getMethodByNameAndParameters(resource, methodName, parameters)
	return m, methodName, remainingParams, err
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

func (sel *DefaultMethodSelector) getMethodByNameAndParameters(resource *openapistackql.Resource, methodName string, parameters map[string]interface{}) (*openapistackql.OperationStore, map[string]interface{}, error) {
	m, remainingParams, ok := resource.GetFirstMethodMatchFromSQLVerb(methodName, parameters)
	if !ok {
		return nil, parameters, fmt.Errorf("no appropriate method = '%s' for resource = '%s'", methodName, resource.Name)
	}
	return m, remainingParams, nil
}
