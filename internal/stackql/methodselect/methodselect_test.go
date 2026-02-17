package methodselect_test

import (
	"testing"

	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql/internal/stackql/methodselect"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stretchr/testify/assert"
)

func TestGetMethodForAction(t *testing.T) {
	sel := &methodselect.DefaultMethodSelector{}
	vr := "v0.1.0"
	svc, err := formulation.LoadProviderAndServiceFromPaths(
		"./testdata/registry/src/aws/"+vr+"/provider.yaml",
		"./testdata/registry/src/aws/"+vr+"/services/s3.yaml",
	)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	resource, err := svc.GetResource("bucket_acls")
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	tests := []struct {
		name       string
		action     string
		parameters parserutil.ColumnKeyedDatastore
		wantErr    bool
		wantMethod string
	}{
		{"SELECT action", "SELECT", parserutil.NewParameterMap(), false, "select"},
		{"UNSUPPORTED action", "UNSUPPORTED", parserutil.NewParameterMap(), true, ""},
		{"Mixed case action", "SeLeCt", parserutil.NewParameterMap(), false, "select"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, methodName, err := sel.GetMethodForAction(resource, tt.action, tt.parameters)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMethod, methodName)
			assert.NotNil(t, method)
		})
	}
}

func TestGetMethod(t *testing.T) {
	sel := &methodselect.DefaultMethodSelector{}
	vr := "v0.1.0"
	svc, err := formulation.LoadProviderAndServiceFromPaths(
		"./testdata/registry/src/aws/"+vr+"/provider.yaml",
		"./testdata/registry/src/aws/"+vr+"/services/s3.yaml",
	)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	resource, err := svc.GetResource("bucket_acls")
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	tests := []struct {
		name       string
		methodName string
		wantErr    bool
	}{
		{"Existing method", "get_bucket_acl", false},
		{"Non-existent method", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := sel.GetMethod(resource, tt.methodName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, method)
		})
	}
}

func TestGetMethodByNameAndParameters(t *testing.T) {
	sel := &methodselect.DefaultMethodSelector{}
	vr := "v0.1.0"
	svc, err := formulation.LoadProviderAndServiceFromPaths(
		"./testdata/registry/src/aws/"+vr+"/provider.yaml",
		"./testdata/registry/src/aws/"+vr+"/services/s3.yaml",
	)
	assert.NoError(t, err)
	assert.NotNil(t, svc)

	resource, err := svc.GetResource("bucket_acls")
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	tests := []struct {
		name       string
		methodName string
		parameters parserutil.ColumnKeyedDatastore
		wantErr    bool
	}{
		{
			"Existing method with parameters",
			"select",
			parserutil.NewParameterMap(),
			false,
		},
		{
			"Non-existent method with parameters",
			"nonexistent",
			parserutil.NewParameterMap(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := sel.GetMethodByNameAndParameters(resource, tt.methodName, tt.parameters)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, method)
		})
	}
}

func TestNewMethodSelector(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		version  string
		wantErr  bool
	}{
		{"Default provider", "aws", "v0.1.0", false},
		{"Unsupported provider", "unsupported", "v0.1.0", false}, // Defaults to Google
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := methodselect.NewMethodSelector(tt.provider, tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, selector)
		})
	}
}
