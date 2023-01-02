

## Azure Auth

### Setting up Azure auth for stackql

#### Setup Azure Browser auth for individual user account / development use of stackql

Using a terminal, enter `az login` and then follow the login flow in the browser.

Having done this, pass the `--auth` parameter into `stackql` with Azure configured to use default auth, like this: `--auth='{ "azure": { "type": "azure_default" }, ... }'`.  Then, Azure auth should proceed transparently.

#### Setup Azure Service Principal auth for stackql

##### Using a client secret

Add the `AZURE_TENANT_ID`, `AZURE_CLIENT_ID` and `AZURE_CLIENT_SECRET` environment variables, as per [the documentation for the golang SDK](https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication-service-principal?tabs=azure-cli#-option-1-authenticate-with-a-secret).

Having done this, pass the `--auth` parameter into `stackql` with Azure configured to use default auth, like this: `--auth='{ "azure": { "type": "azure_default" }, ... }'`.  Then, Azure auth should proceed transparently.

#### Background and existing implementations

- Terraform: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs
- SDK: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#section-readme
- Azure Service Principal setup: https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication-service-principal?tabs=azure-cli
