

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

## k8s auth

k8s supports an adaptable auth flow [client-go credential plugins](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins), which can be and is leveraged by k8s vendors.

- Google have chosen to funnel their k8s auth offering through a `gcloud` plugin, which is opaque. Here is [a community golang implementation](https://pkg.go.dev/github.com/traviswt/gke-auth-plugin).
4


## Foreign auth patterns

### AWS Cross-Account Access

We implemented standard AWS cross-account access using `sts:AssumeRole`. The client account/user/runner is granted permission to assume a read-only audit role in the target account, while the target role trusts the client principal via a trust policy, optionally guarded by `ExternalId`. The target role was granted `SecurityAudit`, S3 read permissions, and Cloud Control (`cloudformation:ListResources`, `cloudformation:GetResource`, etc.) permissions so StackQL can perform live control-plane audits using temporary STS credentials rather than long-lived target-account keys. Docs: https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html

The `--auth` JSON string can be configured to assume a foreign role. This relies on assume-role access being configured in the target tenancy/account.

```json
{"aws":{"type":"aws_assume_role","keyIDenvvar":"AWS_ACCESS_KEY_ID","credentialsenvvar":"AWS_SECRET_ACCESS_KEY","aws_role_arn":"arn:aws:iam::123456789012:role/MyRole"}}
````

### Azure Cross-Tenant Access

We implemented Azure cross-tenant access using a multitenant App Registration in the client tenant. The target tenant admin granted consent to the application, which created an Enterprise Application/service principal inside the target tenant. The target tenant then assigned the built-in `Reader` role to that enterprise application at the subscription or management-group scope. StackQL authenticates using the client application's `client_id` and `client_secret`, but requests tokens from the target tenant authority to audit target Azure resources. Docs: https://learn.microsoft.com/en-us/entra/identity-platform/howto-convert-app-to-be-multi-tenant

No change to canonical Azure configuration is needed; use the client app ID/secret, the target tenant ID, and the target subscription ID.

### GCP Cross-Organization Access

We implemented GCP cross-organization access by creating a service account in the client project and granting that foreign service account IAM roles directly in the target organization/project. The target org/project granted roles such as `Viewer`, `Security Reviewer`, and `Folder Viewer` to the client-owned service account principal, allowing StackQL to audit the target environment while authenticating only with the client-side service account key. Docs: https://cloud.google.com/iam/docs/granting-changing-revoking-access

There is no change to existing Google auth for this.

### Overall

Example, presuming the sourced script contains the cited env vars:

```bash
source cicd/vol/vendor-secrets/foreign_to_stackql_user.sh

stackql --auth '{"aws":{"type":"aws_assume_role","keyIDenvvar":"AWS_ACCESS_KEY_ID","credentialsenvvar":"AWS_SECRET_ACCESS_KEY","aws_role_arn":"'"${STACKQL_AUDIT_ROLE_ARN}"'"}}' shell
```

## OIDC from github

GHA → cloud federation via each provider's official action; stackql then uses its standard auth types — **no federation-specific `--auth` string needed**.

Workflow needs `permissions: id-token: write`, then per cloud:

**AWS** — [GitHub docs](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services). IAM OIDC provider + role; `aws-actions/configure-aws-credentials@v4` with `role-to-assume`. Session token (`AWS_SESSION_TOKEN`) is picked up from env automatically — no extra field needed.
```
--auth '{"aws":{"type":"aws_signing_v4","keyIDenvvar":"AWS_ACCESS_KEY_ID","credentialsenvvar":"AWS_SECRET_ACCESS_KEY"}}'
```

**GCP** — [GitHub docs](https://github.com/google-github-actions/auth#setup). Workload Identity **Pool** + Provider, SA bound via **both** `roles/iam.workloadIdentityUser` **and** `roles/iam.serviceAccountTokenCreator` (the access-token mint path needs Token Creator on top of WIU); `google-github-actions/auth@v2` with `workload_identity_provider` + `service_account` + `token_format: 'access_token'`. Capture the action's `access_token` output into an env var (e.g. `GCP_ACCESS_TOKEN`) for the stackql step. The action's default `external_account` credentials file isn't consumable by stackql's `service_account` type.
```
--auth '{"google":{"type":"bearer","credentialsenvvar":"GCP_ACCESS_TOKEN"}}'
```

**Azure** — [action docs](https://github.com/Azure/login#login-with-openid-connect-oidc-recommended). Entra app + federated credential (one per org — Azure won't accept `repository_owner` matching); `azure/login@v2` with client/tenant/subscription IDs.
```
--auth '{"azure":{"type":"azure_default"}}'
```

### Gotchas
- GCP: use **Workload** Identity Federation (project-scoped), not Workforce. Impersonation via `token_format: access_token` needs **both** `roles/iam.workloadIdentityUser` and `roles/iam.serviceAccountTokenCreator` on the SA. GH secret = full provider resource name (`projects/.../providers/...`).
- Azure: federated credentials for GHA disallow `repository_owner` matching → create one credential per org, or regex on `claims['sub']`.

