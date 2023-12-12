package methodselect

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

type IMethodSelector interface {
	GetMethod(resource anysdk.Resource, methodName string) (anysdk.OperationStore, error)

	GetMethodForAction(
		resource anysdk.Resource,
		iqlAction string,
		parameters parserutil.ColumnKeyedDatastore) (anysdk.OperationStore, string, error)
}

func NewMethodSelector(provider string, version string) (IMethodSelector, error) {
	switch provider { //nolint:gocritic // acceptable
	default:
		return newGoogleMethodSelector(version)
	}
}

func newGoogleMethodSelector(version string) (IMethodSelector, error) {
	switch version { //nolint:gocritic // acceptable
	default:
		return &DefaultMethodSelector{}, nil
	}
}

type DefaultMethodSelector struct {
}

func (sel *DefaultMethodSelector) GetMethodForAction(
	resource anysdk.Resource,
	iqlAction string,
	parameters parserutil.ColumnKeyedDatastore) (anysdk.OperationStore, string, error) {
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
		return nil, "", fmt.Errorf(
			"iql action = '%s' curently not supported, there is no method mapping possible for any resource",
			iqlAction)
	}
	m, err := sel.getMethodByNameAndParameters(resource, methodName, parameters)
	return m, methodName, err
}

func (sel *DefaultMethodSelector) GetMethod(
	resource anysdk.Resource, methodName string) (anysdk.OperationStore, error) {
	return sel.getMethodByName(resource, methodName)
}

func (sel *DefaultMethodSelector) getMethodByName(
	resource anysdk.Resource, methodName string) (anysdk.OperationStore, error) {
	m, err := resource.FindMethod(methodName)
	if err != nil {
		return nil, fmt.Errorf("no method = '%s' for resource = '%s'", methodName, resource.GetName())
	}
	return m, nil
}

func (sel *DefaultMethodSelector) getMethodByNameAndParameters(
	resource anysdk.Resource, methodName string,
	parameters parserutil.ColumnKeyedDatastore) (anysdk.OperationStore, error) {
	stringifiedParams := parameters.GetStringified()
	m, remainingParams, ok := resource.GetFirstMethodMatchFromSQLVerb(methodName, stringifiedParams)
	if !ok {
		return nil, fmt.Errorf("no appropriate method = '%s' for resource = '%s'", methodName, resource.GetName())
	}
	// TODO: fix this bodge and
	//       refactor such that:
	//         - Server selection and variable assignment is AOT and binding
	//         - Server selection is passed in to `Paramaterize()`
	if resource != nil {
		svc, svcExists := resource.GetService()
		if svcExists && len(svc.GetServers()) > 0 {
			for _, srv := range svc.GetServers() {
				for k := range srv.Variables {
					delete(remainingParams, k)
				}
			}
		}
	}
	parameters.DeleteStringMap(remainingParams)
	return m, nil
}
