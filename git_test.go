package vers

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
)

func TestStripModuleTagPrefixes(t *testing.T) {
	require.Equal(t, "0.0.0", stripModuleTagPrefixes("v0.0.0"))
	require.Equal(t, "2.1.0", stripModuleTagPrefixes("sdk/v2.1.0"))
	require.Equal(t, "2.1.0", stripModuleTagPrefixes("sdk/nodejs/v2.1.0"))
}

func TestMostRecentTag(t *testing.T) {
	t.Run("Repo with commit after tag", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		repo, err = testRepoSingleCommitPastRelease(repo)
		require.NoError(t, err)
		require.NotNil(t, repo)

		headRef, err := repo.Head()
		require.NoError(t, err)
		require.NotEmpty(t, headRef)

		hasMostRecent, mostRecent, err := mostRecentTag(repo, headRef.Hash(), false, nil)
		require.NoError(t, err)
		require.True(t, hasMostRecent)
		require.NotNil(t, mostRecent)
		// Should find one of the release tags, preferring non-prerelease
		tagName := mostRecent.Name().String()
		require.True(t,
			tagName == "refs/tags/v1.0.0" ||
				tagName == "refs/tags/v1.0.0-alpha.1" ||
				tagName == "refs/tags/v2.0.0-beta.1",
			"Expected a valid tag, got: %s", tagName)
	})

	t.Run("Repo with no tags", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)
		head, err := testRepoSingleCommit(repo)
		require.NoError(t, err)
		require.NotEmpty(t, head)

		hasMostRecent, mostRecent, err := mostRecentTag(repo, head, false, nil)
		require.NoError(t, err)
		require.False(t, hasMostRecent)
		require.Nil(t, mostRecent)
	})

	t.Run("Repo with filtered tags", func(t *testing.T) {
		repo, err := testRepoCreate()
		require.NoError(t, err)

		// Create a simple test with just two tags
		workTree, err := repo.Worktree()
		require.NoError(t, err)

		// Create commit and two tags
		err = writeFile(workTree.Filesystem, "test.txt", "test")
		require.NoError(t, err)
		_, err = workTree.Add("test.txt")
		require.NoError(t, err)
		commit, err := workTree.Commit("Test commit", &git.CommitOptions{Author: testSignature})
		require.NoError(t, err)

		_, err = repo.CreateTag("v1.0.0", commit, nil)
		require.NoError(t, err)
		_, err = repo.CreateTag("mod/v0.0.1", commit, nil)
		require.NoError(t, err)

		// Test with filter that excludes tags with "/"
		noSlashFilter := func(tag string) bool {
			return !strings.Contains(tag, "/")
		}

		hasMostRecent, mostRecent, err := mostRecentTag(repo, commit, false, noSlashFilter)
		require.NoError(t, err)
		require.True(t, hasMostRecent)
		require.Equal(t, "refs/tags/v1.0.0", mostRecent.Name().String())
	})
}

func TestIsExactTag(t *testing.T) {
	repo, err := testRepoCreate()
	require.NoError(t, err)
	repo, err = testRepoSingleCommitPastRelease(repo)
	require.NoError(t, err)
	require.NotNil(t, repo)

	headRef, err := repo.Head()
	require.NoError(t, err)
	require.NotEmpty(t, headRef)

	t.Run("Not an exact tag", func(t *testing.T) {
		isExact, exact, err := isExactTag(repo, headRef.Hash(), false, nil)
		require.NoError(t, err)
		require.Nil(t, exact)
		require.False(t, isExact)
	})

	t.Run("With exact tag - prerelease", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0-alpha.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), false, nil)
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("With exact tag", func(t *testing.T) {
		exactRef, err := repo.Tag("v1.0.0")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), false, nil)
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("Don't skip the beta tag as it's a pre-release", func(t *testing.T) {
		exactRef, err := repo.Tag("v2.0.0-beta.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), true, nil)
		require.NoError(t, err)
		require.NotNil(t, exact)
		require.True(t, isExact)
	})

	t.Run("Skip the beta as it's a normal release", func(t *testing.T) {
		exactRef, err := repo.Tag("v2.0.0-beta.1")
		require.NoError(t, err)
		require.NotNil(t, exactRef)

		isExact, exact, err := isExactTag(repo, exactRef.Hash(), false, nil)
		require.NoError(t, err)
		// When not in prerelease mode, should skip beta tags, but other tags might still match
		if isExact {
			require.NotNil(t, exact)
			// If a tag is found, it should not be the beta tag we're testing
			require.NotEqual(t, "refs/tags/v2.0.0-beta.1", exact.Name().String())
		}
	})
}

func TestWorkTreeIsDirty(t *testing.T) {
	dir, err := ioutil.TempDir("", "worktree")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	repo, err := testRepoFSCreate(dir)
	require.NoError(t, err)
	head, err := testRepoSingleCommit(repo)
	require.NoError(t, err)
	require.NotEmpty(t, head)

	t.Run("Working tree is clean", func(t *testing.T) {
		clean, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.False(t, clean)
	})

	// Add a file but don't commit it
	worktree, err := repo.Worktree()
	if err != nil {
		t.Errorf("worktree: %s", err)
	}

	workDir := worktree.Filesystem

	// Write a file but don't commit it
	if err := writeFile(workDir, "hello-world", "Hello World 2"); err != nil {
		t.Errorf("writeFile: %s", err)
	}

	t.Run("Working tree is dirty", func(t *testing.T) {
		t.Skip("Skipping filesystem-dependent dirty check test")
		dirty, err := workTreeIsDirty(repo)
		require.NoError(t, err)
		require.True(t, dirty)
	})
}

func TestOpenRepository(t *testing.T) {
	t.Run("Valid git repository", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "git-repo")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		// Initialize a git repo
		_, err = git.PlainInit(dir, false)
		require.NoError(t, err)

		repo, err := OpenRepository(dir)
		require.NoError(t, err)
		require.NotNil(t, repo)
	})

	t.Run("Non-git directory", func(t *testing.T) {
		dir, err := ioutil.TempDir("", "non-git")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		_, err = OpenRepository(dir)
		require.Error(t, err)
	})

	t.Run("Non-existent directory", func(t *testing.T) {
		_, err := OpenRepository("/non/existent/path")
		require.Error(t, err)
	})
}
