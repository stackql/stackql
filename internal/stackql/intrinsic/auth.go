package intrinsic

import (
	"context"
	"strings"
	"time"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

// authFunc translates the SQL command; credential validation belongs to Omni.
func authFunc(ctx queryContext, node *sqlparser.Auth) func() internaldto.ExecutorOutput {
	return func() internaldto.ExecutorOutput {
		configured, err := ctx.GetAuthContext(node.Provider)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		candidate := configured.Clone()
		if node.KeyFilePath != "" {
			candidate.KeyFilePath = node.KeyFilePath
		}
		if node.KeyEnvVar != "" {
			candidate.KeyEnvVar = node.KeyEnvVar
		}
		candidate.Type = strings.ToLower(node.Type)
		if candidate.Type == "" {
			candidate.Type = dto.AuthNullStr
			if candidate.KeyFilePath != "" || candidate.KeyEnvVar != "" {
				candidate.Type = dto.AuthServiceAccountStr
			}
		}
		credential := omnisdk.NewCredential(node.Provider, *dto.OmniAuth(candidate))
		if err = credential.Check(); err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		candidate.Active = true
		if principal, principalErr := credential.Principal(); principalErr == nil {
			candidate.ID = principal
		}
		*configured = *candidate
		return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
	}
}

func authRevokeFunc(ctx queryContext, node *sqlparser.AuthRevoke) func() internaldto.ExecutorOutput {
	return func() internaldto.ExecutorOutput {
		configured, err := ctx.GetAuthContext(node.Provider)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		deadline, cancel := context.WithTimeout(executionContext(ctx), time.Minute)
		defer cancel()
		err = omnisdk.NewCredential(node.Provider, *dto.OmniAuth(configured)).Revoke(deadline)
		if err == nil {
			configured.Active, configured.ID = false, ""
		}
		return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
	}
}
