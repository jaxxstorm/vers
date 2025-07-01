package vers

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

func TestCalculate(t *testing.T) {
	t.Run("Repo with no tags", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		_, err = testRepoSingleCommit(repo)
		require.NoError(t, err)

		opts := Options{
			Repository: repo,
			Commitish:  plumbing.Revision("HEAD"),
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Contains(t, version.SemVer, "0.0.1-alpha")
		require.Contains(t, version.DotNet, "0.0.1-alpha")
		require.Contains(t, version.JavaScript, "v0.0.1-alpha")
		require.Contains(t, version.Python, "0.0.1a")
		require.Contains(t, version.Go, "v0.0.1-alpha")
	})

	t.Run("Repo with exact tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := Options{
			Repository: repo,
			Commitish:  plumbing.Revision("HEAD"),
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0", version.SemVer)
		require.Equal(t, "1.0.0", version.DotNet)
		require.Equal(t, "v1.0.0", version.JavaScript)
		require.Equal(t, "1.0.0", version.Python)
		require.Equal(t, "v1.0.0", version.Go)
	})

	t.Run("Repo with commit after tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		repo, err = testRepoSingleCommitPastRelease(repo)
		require.NoError(t, err)

		opts := Options{
			Repository: repo,
			Commitish:  plumbing.Revision("HEAD"),
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Contains(t, version.SemVer, "1.1.0-alpha")
		require.Contains(t, version.DotNet, "1.1.0-alpha")
		require.Contains(t, version.JavaScript, "v1.1.0-alpha")
		require.Contains(t, version.Python, "1.1.0a")
		require.Contains(t, version.Go, "v1.1.0-alpha")
	})

	t.Run("Repo with pre-release tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0-alpha.1",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := Options{
			Repository:   repo,
			Commitish:    plumbing.Revision("HEAD"),
			IsPreRelease: true,
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Equal(t, "1.0.0-alpha.1", version.SemVer)
		require.Equal(t, "1.0.0-alpha.1", version.DotNet)
		require.Equal(t, "v1.0.0-alpha.1", version.JavaScript)
		require.Equal(t, "1.0.0a1", version.Python)
		require.Equal(t, "v1.0.0-alpha.1", version.Go)
	})

	t.Run("Repo with release prefix override", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		_, err = testRepoSingleCommit(repo)
		require.NoError(t, err)

		opts := Options{
			Repository:    repo,
			Commitish:     plumbing.Revision("HEAD"),
			ReleasePrefix: "2.0.0",
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Contains(t, version.SemVer, "2.0.0-alpha")
		require.Contains(t, version.DotNet, "2.0.0-alpha")
		require.Contains(t, version.JavaScript, "v2.0.0-alpha")
		require.Contains(t, version.Python, "2.0.0a")
		require.Contains(t, version.Go, "v2.0.0-alpha")
	})

	t.Run("Repo with tag pattern filter", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		tagSequence := []string{
			"v1.0.0",
			"sdk/v2.0.0",
		}

		repo, err = testRepoWithTags(repo, tagSequence)
		require.NoError(t, err)

		opts := Options{
			Repository: repo,
			Commitish:  plumbing.Revision("HEAD"),
			TagPattern: "^sdk/",
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Equal(t, "2.0.0", version.SemVer)
		require.Equal(t, "2.0.0", version.DotNet)
		require.Equal(t, "v2.0.0", version.JavaScript)
		require.Equal(t, "2.0.0", version.Python)
		require.Equal(t, "v2.0.0", version.Go)
	})

	t.Run("Repo with omit commit hash", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		repo, err = testRepoSingleCommitPastRelease(repo)
		require.NoError(t, err)

		opts := Options{
			Repository:     repo,
			Commitish:      plumbing.Revision("HEAD"),
			OmitCommitHash: true,
		}
		version, err := Calculate(opts)
		require.NoError(t, err)

		require.Contains(t, version.SemVer, "1.1.0-alpha")
		require.Contains(t, version.DotNet, "1.1.0-alpha")
		require.Contains(t, version.JavaScript, "v1.1.0-alpha")
		require.Contains(t, version.Python, "1.1.0a")
		require.Contains(t, version.Go, "v1.1.0-alpha")
	})

	t.Run("Invalid tag pattern", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		_, err = testRepoSingleCommit(repo)
		require.NoError(t, err)

		opts := Options{
			Repository: repo,
			Commitish:  plumbing.Revision("HEAD"),
			TagPattern: "[invalid regex",
		}
		_, err = Calculate(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid tag pattern")
	})

	t.Run("Nil repository", func(t *testing.T) {
		opts := Options{
			Repository: nil,
			Commitish:  plumbing.Revision("HEAD"),
		}
		_, err := Calculate(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repository is required")
	})
}

func TestCalculateFromString(t *testing.T) {
	t.Run("Basic semantic version", func(t *testing.T) {
		version, err := CalculateFromString("1.2.3")
		require.NoError(t, err)
		require.Equal(t, "1.2.3", version.SemVer)
		require.Equal(t, "1.2.3", version.Python)
		require.Equal(t, "v1.2.3", version.JavaScript)
		require.Equal(t, "1.2.3", version.DotNet)
		require.Equal(t, "v1.2.3", version.Go)
	})

	t.Run("Version with v prefix", func(t *testing.T) {
		version, err := CalculateFromString("v1.2.3")
		require.NoError(t, err)
		require.Equal(t, "1.2.3", version.SemVer)
		require.Equal(t, "1.2.3", version.Python)
		require.Equal(t, "v1.2.3", version.JavaScript)
		require.Equal(t, "1.2.3", version.DotNet)
		require.Equal(t, "v1.2.3", version.Go)
	})

	t.Run("Pre-release version", func(t *testing.T) {
		version, err := CalculateFromString("1.2.3-alpha.1")
		require.NoError(t, err)
		require.Equal(t, "1.2.3-alpha.1", version.SemVer)
		require.Equal(t, "1.2.3a1", version.Python)
		require.Equal(t, "v1.2.3-alpha.1", version.JavaScript)
		require.Equal(t, "1.2.3-alpha.1", version.DotNet)
		require.Equal(t, "v1.2.3-alpha.1", version.Go)
	})

	t.Run("Beta pre-release version", func(t *testing.T) {
		version, err := CalculateFromString("1.2.3-beta.2")
		require.NoError(t, err)
		require.Equal(t, "1.2.3-beta.2", version.SemVer)
		require.Equal(t, "1.2.3b2", version.Python)
		require.Equal(t, "v1.2.3-beta.2", version.JavaScript)
		require.Equal(t, "1.2.3-beta.2", version.DotNet)
		require.Equal(t, "v1.2.3-beta.2", version.Go)
	})

	t.Run("RC pre-release version", func(t *testing.T) {
		version, err := CalculateFromString("1.2.3-rc.1")
		require.NoError(t, err)
		require.Equal(t, "1.2.3-rc.1", version.SemVer)
		require.Equal(t, "1.2.3rc1", version.Python)
		require.Equal(t, "v1.2.3-rc.1", version.JavaScript)
		require.Equal(t, "1.2.3-rc.1", version.DotNet)
		require.Equal(t, "v1.2.3-rc.1", version.Go)
	})

	t.Run("Invalid version format", func(t *testing.T) {
		_, err := CalculateFromString("1.2")
		require.Error(t, err)
		require.Contains(t, err.Error(), "version must have exactly 3 parts")
	})

	t.Run("Empty version", func(t *testing.T) {
		_, err := CalculateFromString("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "version must have exactly 3 parts")
	})
}

func TestGenerateFallbackVersion(t *testing.T) {
	version := GenerateFallbackVersion()
	require.NotNil(t, version)
	require.Equal(t, "0.0.0-dev", version.SemVer)
	require.Equal(t, "0.0.0.dev0", version.Python)
	require.Equal(t, "v0.0.0-dev", version.JavaScript)
	require.Equal(t, "0.0.0-dev", version.DotNet)
	require.Equal(t, "v0.0.0-dev", version.Go)
}
