package methodselect

import (
	"fmt"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type IMethodSelector interface {
	GetMethod(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error)

	GetMethodForAction(resource *openapistackql.Resource, iqlAction string) (*openapistackql.OperationStore, string, error)
}

func NewMethodSelector(provider string, version string) (IMethodSelector, error) {
	switch provider {
	case "google", "okta":
		return newGoogleMethodSelector(version)
	}
	return nil, fmt.Errorf("method selector for provider = '%s', api version = '%s' currently not supported", provider, version)
}

func newGoogleMethodSelector(version string) (IMethodSelector, error) {
	switch version {
	case "v1":
		return &DefaultGoogleMethodSelector{}, nil
	}
	return nil, fmt.Errorf("method selector for google, api version = '%s' currently not supported", version)
}

type DefaultGoogleMethodSelector struct {
}

func (sel *DefaultGoogleMethodSelector) GetMethodForAction(resource *openapistackql.Resource, iqlAction string) (*openapistackql.OperationStore, string, error) {
	var methodName string
	switch strings.ToLower(iqlAction) {
	case "select":
		methodName = "list"
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
	m, err := sel.getMethodByName(resource, methodName)
	return m, methodName, err
}

func (sel *DefaultGoogleMethodSelector) GetMethod(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error) {
	return sel.getMethodByName(resource, methodName)
}

func (sel *DefaultGoogleMethodSelector) getMethodByName(resource *openapistackql.Resource, methodName string) (*openapistackql.OperationStore, error) {
	m, err := resource.FindMethod(methodName)
	if err != nil {
		return nil, fmt.Errorf("no method = '%s' for resource = '%s'", methodName, resource.Name)
	}
	return m, nil
}
