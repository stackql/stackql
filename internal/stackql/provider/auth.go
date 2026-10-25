package provider

import (
	"errors"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"

	"github.com/stackql/stackql/internal/stackql/constants"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

func (gp *GenericProvider) CheckCredentialFile(authCtx *dto.AuthCtx) error {
	return omnisdk.NewCredential(gp.GetProviderString(), *dto.OmniAuth(authCtx)).Check()
}

func (gp *GenericProvider) ShowAuth(authCtx *dto.AuthCtx) (*omnisdk.AuthMetadata, error) {
	if authCtx == nil {
		return nil, errors.New(constants.NotAuthenticatedShowStr)
	}
	principal, err := omnisdk.NewCredential(gp.GetProviderString(), *dto.OmniAuth(authCtx)).Principal()
	if err != nil {
		return nil, err
	}
	return &omnisdk.AuthMetadata{
		Principal: principal, Type: strings.ToUpper(authCtx.Type),
		Source: authCtx.GetCredentialsSourceDescriptorString(),
	}, nil
}
