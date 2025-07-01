package vers

import (
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

func TestLanguageVersions(t *testing.T) {
	versions := &LanguageVersions{
		SemVer:     "1.2.3",
		Python:     "1.2.3",
		JavaScript: "v1.2.3",
		DotNet:     "1.2.3",
		Go:         "v1.2.3",
	}

	require.Equal(t, "1.2.3", versions.SemVer)
	require.Equal(t, "1.2.3", versions.Python)
	require.Equal(t, "v1.2.3", versions.JavaScript)
	require.Equal(t, "1.2.3", versions.DotNet)
	require.Equal(t, "v1.2.3", versions.Go)
}

func TestOptions(t *testing.T) {
	repo, err := testRepoCreate()
	require.NoError(t, err)

	opts := Options{
		Repository:     repo,
		Commitish:      plumbing.Revision("HEAD"),
		OmitCommitHash: true,
		ReleasePrefix:  "2.0.0",
		IsPreRelease:   true,
		TagPattern:     "^v",
	}

	require.Equal(t, repo, opts.Repository)
	require.Equal(t, plumbing.Revision("HEAD"), opts.Commitish)
	require.True(t, opts.OmitCommitHash)
	require.Equal(t, "2.0.0", opts.ReleasePrefix)
	require.True(t, opts.IsPreRelease)
	require.Equal(t, "^v", opts.TagPattern)
	require.Nil(t, opts.TagFilter) // Should be nil until applied
}

func TestVersionComponents(t *testing.T) {
	version, err := semver.Parse("1.2.3-alpha.1")
	require.NoError(t, err)

	now := time.Now()
	components := &VersionComponents{
		Semver:    version,
		Dirty:     true,
		ShortHash: "abc123de",
		Timestamp: now,
		IsExact:   false,
	}

	require.Equal(t, version, components.Semver)
	require.True(t, components.Dirty)
	require.Equal(t, "abc123de", components.ShortHash)
	require.Equal(t, now, components.Timestamp)
	require.False(t, components.IsExact)
}

func TestOptionsWithTagFilter(t *testing.T) {
	repo, err := testRepoCreate()
	require.NoError(t, err)

	tagFilter := func(tag string) bool {
		return tag == "v1.0.0"
	}

	opts := Options{
		Repository: repo,
		Commitish:  plumbing.Revision("HEAD"),
		TagFilter:  tagFilter,
	}

	require.Equal(t, repo, opts.Repository)
	require.Equal(t, plumbing.Revision("HEAD"), opts.Commitish)
	require.NotNil(t, opts.TagFilter)
	require.True(t, opts.TagFilter("v1.0.0"))
	require.False(t, opts.TagFilter("v2.0.0"))
}
