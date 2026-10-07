package dependencies //nolint:testpackage // controlled build-info fixtures

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplacementAttribution(t *testing.T) {
	original := &debug.Module{
		Path: "github.com/upstream/parser", Version: "v1.0.0", Sum: "h1:original",
		Replace: &debug.Module{
			Path: "github.com/fork/parser", Version: "v0.0.0-20260101000000-abcdef123456", Sum: "h1:replacement",
		},
	}
	metadata := []Metadata{
		{Name: original.Path, Version: original.Version, License: "MIT"},
		{
			Name: original.Replace.Path, Version: original.Replace.Version, License: "Apache-2.0 OR MIT",
			Description: "Vitess-derived parser", SourceURL: "https://github.com/fork/parser",
			LicenseURL: "https://github.com/fork/parser/blob/abcdef123456/LICENSE",
		},
		{Name: "github.com/test/only", Version: "v1.0.0", License: "BSD-3-Clause"},
	}
	info := &debug.BuildInfo{Deps: []*debug.Module{original}}
	extended, err := rows(info, metadata, true, nil)
	require.NoError(t, err)
	require.Len(t, extended, 1)
	row := extended["000000000"]
	require.Equal(t, original.Replace.Path, row["name"])
	require.Equal(t, original.Replace.Version, row["version"])
	require.Equal(t, "h1:replacement", row["checksum"])
	require.Equal(t, "Apache-2.0 OR MIT", row["license"])
	require.Equal(t, original.Path, row["original_name"])
	require.Equal(t, original.Version, row["original_version"])
	require.Equal(t, metadata[1].LicenseURL, row["license_url"])
	require.Equal(t, "pkg:golang/"+original.Replace.Path+"@"+original.Replace.Version, row["purl"])
	ordinary, err := rows(info, metadata, false, nil)
	require.NoError(t, err)
	require.Len(t, ordinary["000000000"], len(Columns(false)))
	for _, column := range Columns(false) {
		require.Equal(t, ordinary["000000000"][column], row[column])
	}
}

func TestMissingMetadataAndLocalReplacement(t *testing.T) {
	info := &debug.BuildInfo{Deps: []*debug.Module{
		{Path: "github.com/ordinary/module", Version: "v2.0.0"},
		{Path: "github.com/original/module", Version: "v1.0.0", Replace: &debug.Module{Path: "../local"}},
	}}
	result, err := rows(info, []Metadata{
		{Name: "github.com/ordinary/module", Version: "v1.0.0", License: "MIT"},
	}, true, nil)
	require.NoError(t, err)
	require.Len(t, result, 2)
	local := result["000000000"]
	require.Equal(t, "../local", local["name"])
	require.Nil(t, local["version"])
	for _, column := range []string{"description", "source_url", "license_url", "purl", "checksum"} {
		require.Nil(t, local[column], column)
	}
	ordinary := result["000000001"]
	require.Equal(t, "UNKNOWN", ordinary["license"])
	require.Nil(t, ordinary["original_name"])
	require.Nil(t, ordinary["original_version"])
}

func TestLikeFixtures(t *testing.T) {
	original := &debug.Module{
		Path: "github.com/upstream/original", Version: "v1",
		Replace: &debug.Module{Path: "github.com/fork/parser", Version: "v2"},
	}
	info := &debug.BuildInfo{Deps: []*debug.Module{original}}
	metadata := []Metadata{{
		Name: original.Replace.Path, Version: "v2", Description: "Vitess's SQL parser [fork]",
	}}
	for _, pattern := range []string{
		"%FORK%", "github.com/fork/par_er", "%UPSTREAM%", "%vitess%", "%'s%", "%[fork]%", "%",
	} {
		t.Run(pattern, func(t *testing.T) {
			for _, extended := range []bool{false, true} {
				result, err := rows(info, metadata, extended, &pattern)
				require.NoError(t, err)
				require.Len(t, result, 1)
			}
		})
	}
	for _, pattern := range []string{"", "Vitess", "%missing%", "github.com/fork/par__er", "%\\%"} {
		result, err := rows(info, metadata, true, &pattern)
		require.NoError(t, err)
		require.Empty(t, result)
	}
}

func TestInventoryOrderingAndDeduplication(t *testing.T) {
	info := &debug.BuildInfo{Deps: []*debug.Module{
		{Path: "github.com/b/module", Version: "v1"},
		{Path: "github.com/a/module/sub", Version: "v1"},
		{Path: "github.com/a/module", Version: "v2"},
		{Path: "github.com/a/module", Version: "v1"},
		{Path: "github.com/a/module", Version: "v1"},
	}}
	result, err := rows(info, nil, false, nil)
	require.NoError(t, err)
	require.Len(t, result, 4)
	require.Equal(t, "github.com/a/module", result["000000000"]["name"])
	require.Equal(t, "v1", result["000000000"]["version"])
	require.Equal(t, "v2", result["000000001"]["version"])
	require.Equal(t, "github.com/a/module/sub", result["000000002"]["name"])
	require.Equal(t, "github.com/b/module", result["000000003"]["name"])
}

func TestBuildInfoReconciliation(t *testing.T) {
	_, err := rows(nil, nil, false, nil)
	require.ErrorContains(t, err, "build information is unavailable")
	modules := []*debug.Module{{Path: "github.com/a/module", Version: "v1", Sum: "h1:first"}}
	require.True(t, sameModules(modules, modules))
	require.False(t, sameModules(modules, nil))
	require.False(t, sameModules(modules, []*debug.Module{{Path: modules[0].Path, Version: "v2"}}))
	require.False(t, sameModules(modules, []*debug.Module{{
		Path: modules[0].Path, Version: "v1", Sum: "h1:different",
	}}))
	require.Nil(t, packageURL(&debug.Module{Path: `C:\local\module`}))
	require.Equal(t, "pkg:golang/github.com/acme/module@v2.0.0%2Bincompatible",
		packageURL(&debug.Module{Path: "github.com/acme/module", Version: "v2.0.0+incompatible"}))
}
