package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackageGraphExcludesNonBinaryModules(t *testing.T) {
	modules, err := decodeModules([]byte(`
		{"Standard":true}
		{"Module":{"Path":"github.com/stackql/stackql"}}
		{"Module":{"Path":"github.com/original/parser","Version":"v1","Replace":{
			"Path":"github.com/fork/parser","Version":"v2","Sum":"h1:fork"}}}
		{"Module":{"Path":"github.com/original/parser","Version":"v1","Replace":{
			"Path":"github.com/fork/parser","Version":"v2","Sum":"h1:fork"}}}
	`))
	require.NoError(t, err)
	require.Len(t, modules, 1)
	component := buildModule(modules["github.com/fork/parser@v2"])
	require.Equal(t, "github.com/original/parser", component.Path)
	require.Equal(t, "h1:fork", component.Replace.Sum)
	_, err = decodeModules([]byte(`{"Module":`))
	require.Error(t, err)
}

func TestPinnedLicenseLinks(t *testing.T) {
	for _, test := range []struct {
		path, version, expected string
	}{
		{"github.com/acme/module", "v1.0.0", "v1.0.0/LICENSE"},
		{"github.com/acme/module/v2", "v2.0.0", "v2.0.0/LICENSE"},
		{"github.com/acme/module/sdk/v2", "v2.0.0", "sdk/v2.0.0/sdk/LICENSE"},
		{"github.com/acme/module", "v0.0.0-20260101000000-abcdef123456", "abcdef123456/LICENSE"},
		{"github.com/acme/module", "v2.0.0+incompatible", "v2.0.0/LICENSE"},
	} {
		t.Run(test.path+"@"+test.version, func(t *testing.T) {
			require.Equal(t, "https://github.com/acme/module/blob/"+test.expected,
				licenseURL(&module{Path: test.path, Version: test.version}, "https://github.com/acme/module", "LICENSE"))
		})
	}
}
