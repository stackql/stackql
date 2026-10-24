package intrinsic

import (
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/docparse/stackqldoc"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-provider-registry/signing/pqsig"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
	"gopkg.in/yaml.v2"
)

//nolint:nilnil // A nil verifier is the explicit, successful opt-out state.
func documentVerifier(runtime dto.RuntimeCtx) (pqsig.Verifier, error) {
	var cfg omnisdk.RegistryConfig
	if err := yaml.Unmarshal([]byte(runtime.RegistryRaw), &cfg); err != nil {
		return nil, fmt.Errorf("document verification config: %w", err)
	}
	var keys map[string]string
	if cfg.VerifyConfig != nil {
		if cfg.VerifyConfig.NopVerify {
			return nil, nil
		}
		keys = cfg.VerifyConfig.PublicKeys
	}
	return pqsig.NewVerifier(keys)
}

func documentOptions(ctx queryContext) ([]stackqldoc.Option, error) {
	verifier, err := documentVerifier(ctx.GetRuntimeContext())
	if err != nil {
		return nil, err
	}
	if verifier == nil {
		return nil, nil
	}
	return []stackqldoc.Option{stackqldoc.WithVerifier(verifier)}, nil
}
