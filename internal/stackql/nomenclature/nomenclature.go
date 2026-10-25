package nomenclature

import (
	"strings"

	"github.com/stackql/stackql-provider-registry/registry/pkg/nomenclature"
)

type ProviderDesignation nomenclature.ProviderDesignation

func ExtractProviderDesignation(providerStr string) (ProviderDesignation, error) {
	rv, err := nomenclature.ExtractProviderDesignation(providerStr)
	if !strings.ContainsAny(providerStr, ":@") {
		rv.Tag = ""
	}
	return ProviderDesignation(rv), err
}
