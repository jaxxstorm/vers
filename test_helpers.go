package vers

import (
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
)

var testSignature = &object.Signature{
	Name:  "test",
	Email: "test@example.com",
	When:  time.Now(),
}

// testRepoCreate creates a new in-memory git repository for testing
func testRepoCreate() (*git.Repository, error) {
	storage := memory.NewStorage()
	fs := memfs.New()
	return git.Init(storage, fs)
}

// testRepoFSCreate creates a new filesystem-based git repository for testing
func testRepoFSCreate(path string) (*git.Repository, error) {
	fs := osfs.New(path)
	storage := filesystem.NewStorage(fs, nil)
	return git.Init(storage, fs)
}

// testRepoSingleCommit adds a single commit to the repository and returns the commit hash
func testRepoSingleCommit(repo *git.Repository) (plumbing.Hash, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	err = writeFile(workTree.Filesystem, "test.txt", "Hello world")
	if err != nil {
		return plumbing.ZeroHash, err
	}

	_, err = workTree.Add("test.txt")
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return workTree.Commit("Initial commit", &git.CommitOptions{Author: testSignature})
}

// testRepoSingleCommitPastRelease creates a repo with a tag and then adds another commit
func testRepoSingleCommitPastRelease(repo *git.Repository) (*git.Repository, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	// Create initial commit and tag it
	err = writeFile(workTree.Filesystem, "initial.txt", "Initial content")
	if err != nil {
		return nil, err
	}

	_, err = workTree.Add("initial.txt")
	if err != nil {
		return nil, err
	}

	tagCommit, err := workTree.Commit("Release commit", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, err
	}

	// Create tags in order (most recent first for the test logic)
	_, err = repo.CreateTag("v1.0.0", tagCommit, nil)
	if err != nil {
		return nil, err
	}

	_, err = repo.CreateTag("v1.0.0-alpha.1", tagCommit, nil)
	if err != nil {
		return nil, err
	}

	_, err = repo.CreateTag("v2.0.0-beta.1", tagCommit, nil)
	if err != nil {
		return nil, err
	}

	// Add another commit after the tag
	err = writeFile(workTree.Filesystem, "post-release.txt", "Post release content")
	if err != nil {
		return nil, err
	}

	_, err = workTree.Add("post-release.txt")
	if err != nil {
		return nil, err
	}

	_, err = workTree.Commit("Post-release commit", &git.CommitOptions{Author: testSignature})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// testRepoWithTags creates a repository with a sequence of tags
func testRepoWithTags(repo *git.Repository, tags []string) (*git.Repository, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	for i, tag := range tags {
		filename := "file_" + tag + ".txt"
		content := "Content for " + tag

		err = writeFile(workTree.Filesystem, filename, content)
		if err != nil {
			return nil, err
		}

		_, err = workTree.Add(filename)
		if err != nil {
			return nil, err
		}

		commitMsg := "Commit for " + tag
		commitHash, err := workTree.Commit(commitMsg, &git.CommitOptions{Author: testSignature})
		if err != nil {
			return nil, err
		}

		// For the last tag, create all previous tags pointing to this commit
		if i == len(tags)-1 {
			for _, tagName := range tags {
				_, err = repo.CreateTag(tagName, commitHash, nil)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return repo, nil
}

// addFile is a helper function to add a file to the worktree
func addFile(t interface{}, worktree *git.Worktree, filename, content string) {
	err := writeFile(worktree.Filesystem, filename, content)
	if err != nil {
		if tester, ok := t.(interface{ Errorf(string, ...interface{}) }); ok {
			tester.Errorf("writeFile: %s", err)
		}
	}

	_, err = worktree.Add(filename)
	if err != nil {
		if tester, ok := t.(interface{ Errorf(string, ...interface{}) }); ok {
			tester.Errorf("Add: %s", err)
		}
	}
}

// writeFile writes content to a file in the given filesystem
func writeFile(fs billy.Filesystem, filename, content string) error {
	file, err := fs.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(content))
	return err
}
