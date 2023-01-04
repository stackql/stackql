package azureauth

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

var (
	_ AzureTokenSource = &standardAzureTokenSource{}
)

type AzureTokenSource interface {
	GetToken(context.Context) (azcore.AccessToken, error)
}

type standardAzureTokenSource struct {
}

func NewDefaultCredentialAzureTokenSource() (AzureTokenSource, error) {
	return &standardAzureTokenSource{}, nil
}

func (ats *standardAzureTokenSource) GetToken(ctx context.Context) (azcore.AccessToken, error) {
	var token azcore.AccessToken
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return token, fmt.Errorf("azure credential acquire error = '%s'", err.Error())
	}
	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{
			"https://management.core.windows.net//.default",
		},
	}
	token, err = cred.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		return token, fmt.Errorf("azure token get error = '%s'", err.Error())
	}
	return token, nil
}
