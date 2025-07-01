// Package vers provides semantic versioning utilities for Git repositories.
//
// This file contains code adapted from pulumictl (https://github.com/pulumi/pulumictl)
// which is licensed under the Apache License 2.0. See NOTICE file for full attribution.
package vers

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/blang/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// OpenRepository opens a Git repository at the specified path
func OpenRepository(path string) (*git.Repository, error) {
	return git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
}

func getVersionComponents(opts Options) (*VersionComponents, error) {
	revision, err := opts.Repository.ResolveRevision(opts.Commitish)
	if err != nil {
		return nil, fmt.Errorf("resolving commitish: %w", err)
	}

	commit, err := opts.Repository.CommitObject(*revision)
	if err != nil {
		return nil, fmt.Errorf("getting commit object: %w", err)
	}

	baseVersion, isExact, err := determineBaseVersion(
		opts.Repository, revision, opts.IsPreRelease, opts.TagFilter)
	if err != nil {
		return nil, fmt.Errorf("determining base version: %w", err)
	}

	version, err := semver.Parse(baseVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing base version %q: %w", baseVersion, err)
	}

	// Increment version for non-exact matches
	if !isExact {
		if version.Major == 0 {
			version.Patch++
		} else {
			version.Minor++
			version.Patch = 0
		}
		version.Pre = []semver.PRVersion{{VersionStr: "alpha"}}
	}

	// Apply release prefix override
	if opts.ReleasePrefix != "" {
		newVersion, err := semver.Parse(opts.ReleasePrefix)
		if err != nil {
			return nil, fmt.Errorf("parsing release prefix %q: %w", opts.ReleasePrefix, err)
		}
		version.Major = newVersion.Major
		version.Minor = newVersion.Minor
		version.Patch = newVersion.Patch
	}

	isDirty, err := workTreeIsDirty(opts.Repository)
	if err != nil {
		return nil, fmt.Errorf("checking if worktree is dirty: %w", err)
	}

	return &VersionComponents{
		Semver:    version,
		Dirty:     isDirty,
		ShortHash: revision.String()[:8],
		Timestamp: commit.Committer.When,
		IsExact:   isExact,
	}, nil
}

func determineBaseVersion(repo *git.Repository, revision *plumbing.Hash,
	isPrerelease bool, tagFilter func(string) bool) (string, bool, error) {

	commit, err := repo.CommitObject(*revision)
	if err != nil {
		return "", false, fmt.Errorf("getting commit object: %w", err)
	}

	// Check for exact tag match
	isExact, exactMatch, err := isExactTag(repo, commit.Hash, isPrerelease, tagFilter)
	if err != nil {
		return "", false, fmt.Errorf("checking exact tag: %w", err)
	}
	if isExact {
		return stripModuleTagPrefixes(exactMatch.Name().Short()), true, nil
	}

	// Find most recent tag
	hasRecent, recentMatch, err := mostRecentTag(repo, commit.Hash, isPrerelease, tagFilter)
	if err != nil {
		return "", false, fmt.Errorf("finding recent tag: %w", err)
	}
	if hasRecent {
		return stripModuleTagPrefixes(recentMatch.Name().Short()), false, nil
	}

	return "0.0.0", false, nil
}

func stripModuleTagPrefixes(tag string) string {
	_, versionComponent := path.Split(tag)
	return strings.TrimPrefix(versionComponent, "v")
}

func isExactTag(repo *git.Repository, hash plumbing.Hash,
	isPrerelease bool, tagFilter func(string) bool) (bool, *plumbing.Reference, error) {

	tags, err := repo.Tags()
	if err != nil {
		return false, nil, fmt.Errorf("listing tags: %w", err)
	}

	var exactTag *plumbing.Reference
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference {
			return nil
		}

		refName := ref.Name().String()

		// Skip beta/rc tags if not prerelease
		if !isPrerelease && (strings.Contains(refName, "beta") || strings.Contains(refName, "rc")) {
			return nil
		}

		// Apply tag filter
		if tagFilter != nil && !tagFilter(strings.TrimPrefix(refName, "refs/tags/")) {
			return nil
		}

		obj, err := repo.TagObject(ref.Hash())
		switch err {
		case nil:
			// Annotated tag
			if obj.Target == hash {
				exactTag = ref
				return storer.ErrStop
			}
		case plumbing.ErrObjectNotFound:
			// Lightweight tag
			if ref.Hash() == hash {
				exactTag = ref
				return storer.ErrStop
			}
		default:
			return err
		}

		return nil
	})

	return exactTag != nil, exactTag, err
}

func mostRecentTag(repo *git.Repository, ref plumbing.Hash,
	isPrerelease bool, tagFilter func(string) bool) (bool, *plumbing.Reference, error) {

	commit, err := repo.CommitObject(ref)
	if err != nil {
		return false, nil, fmt.Errorf("getting commit object: %w", err)
	}

	var mostRecentTag *plumbing.Reference
	walker := object.NewCommitPreorderIter(commit, nil, nil)

	err = walker.ForEach(func(commit *object.Commit) error {
		isExact, exact, err := isExactTag(repo, commit.Hash, isPrerelease, tagFilter)
		if err != nil {
			return err
		}

		if isExact {
			mostRecentTag = exact
			return storer.ErrStop
		}

		return nil
	})

	return mostRecentTag != nil, mostRecentTag, err
}

func workTreeIsDirty(repo *git.Repository) (bool, error) {
	workTree, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("getting worktree: %w", err)
	}

	// Fast path for filesystem storage
	if _, ok := repo.Storer.(*filesystem.Storage); ok {
		return checkDirtyWithGitCommand(workTree.Filesystem.Root())
	}

	// Fallback to go-git status check
	status, err := workTree.Status()
	if err != nil {
		return false, fmt.Errorf("getting git status: %w", err)
	}

	return !status.IsClean(), nil
}

func checkDirtyWithGitCommand(repoPath string) (bool, error) {
	// Refresh index first
	cmd := exec.Command("git", "update-index", "-q", "--refresh")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// If update-index fails, assume dirty
		return true, nil
	}

	// Check for changes
	cmd = exec.Command("git", "diff-files", "--name-status", "--ignore-space-at-eol")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return true, nil
		}
		return false, err
	}

	return len(output) > 0, nil
}
