package intrinsic

import "github.com/stackql-labs/omnisdk/pkg/omnisdk"

func registryResourceNames(ctx queryContext, bundle, service string, _ bool) (map[string]string, error) {
	dir, err := docRoot(ctx, bundle)
	if err != nil {
		return nil, err
	}
	opts, err := documentOptions(ctx)
	if err != nil {
		return nil, err
	}
	return omnisdk.DocResourceNames(dir, bundle, service, opts...)
}
