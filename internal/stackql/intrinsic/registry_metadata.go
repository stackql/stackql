package intrinsic

import "github.com/stackql/stackql/internal/stackql/provider"

// Execution-backend selection does not alter the registry's presentation
// metadata. Keep resource keys for routing, and declared names for SHOW output.
func registryResourceNames(ctx queryContext, bundle, service string, extended bool) (map[string]string, error) {
	if !previewCfg.getOmniAll() {
		return map[string]string{}, nil
	}
	host, ok := ctx.(interface {
		GetProvider(string) (provider.IProvider, error)
	})
	if !ok {
		return map[string]string{}, nil
	}
	prov, err := host.GetProvider(bundle)
	if err != nil {
		return nil, err
	}
	resources, err := prov.GetResourcesRedacted(service, ctx.GetRuntimeContext(), extended)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(resources))
	for key, resource := range resources {
		names[key] = resource.GetName()
	}
	return names, nil
}
