package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jaxxstorm/vers"
)

// Version will be set by build process
var Version = "dev"

type CLI struct {
	Commitish      string `arg:"" optional:"" help:"Git commitish to analyze or version string to convert (default: HEAD)"`
	Language       string `short:"l" default:"generic" enum:"generic,semver,python,javascript,js,node,dotnet,csharp,go,golang" help:"Output format"`
	Repo           string `short:"r" help:"Repository path (default: current directory)"`
	VersionPrefix  string `help:"Version prefix override (e.g., '3.0.0')"`
	OmitCommitHash bool   `short:"o" help:"Omit commit hash from version"`
	IsPreRelease   bool   `help:"Mark as pre-release version"`
	TagPattern     string `help:"Regex pattern to filter tags (e.g., '^sdk/')"`
	JSON           bool   `short:"j" help:"Output as JSON"`
	ShowVersion    bool   `help:"Show version information" name:"version"`
}

func main() {
	var cli CLI

	kong.Parse(&cli,
		kong.Name("vers"),
		kong.Description("Calculate semantic versions from Git repository state or convert version strings"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": Version,
		},
	)

	err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func (c *CLI) Run() error {

	if c.ShowVersion {
		return c.showVersion()
	}

	// Check if the input looks like a version string to convert
	if c.Commitish != "" && isVersionString(c.Commitish) {
		return c.convertVersion()
	}

	// Otherwise, calculate from git repository
	return c.calculateVersion()
}

func (c *CLI) showVersion() error {
	// Calculate version from git repository instead of showing binary version
	return c.calculateVersion()
}

func (c *CLI) convertVersion() error {
	versions, err := vers.CalculateFromString(c.Commitish)
	if err != nil {
		return fmt.Errorf("converting version: %w", err)
	}

	if c.JSON {
		return json.NewEncoder(os.Stdout).Encode(versions)
	}

	output := getVersionOutput(versions, c.Language)
	fmt.Println(output)

	return nil
}

func (c *CLI) calculateVersion() error {
	commitish := "HEAD"
	if c.Commitish != "" {
		commitish = c.Commitish
	}

	repoPath := c.Repo
	if repoPath == "" {
		var err error
		repoPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Try to open repository, but handle gracefully if it's not a git repo
	repo, err := vers.OpenRepository(repoPath)
	if err != nil {
		// If we can't open the repository, generate a fallback version
		versions := vers.GenerateFallbackVersion()

		if c.JSON {
			return json.NewEncoder(os.Stdout).Encode(versions)
		}

		output := getVersionOutput(versions, c.Language)
		fmt.Println(output)
		return nil
	}

	// Validate commitish against actual git repository
	if c.Commitish != "" && !isValidCommitishInRepo(repo, c.Commitish) {
		fmt.Fprintf(os.Stderr, "WARN: '%s' does not exist in this git repository (not a valid branch, tag, or commit)\n", c.Commitish)
	}

	opts := vers.Options{
		Repository:     repo,
		Commitish:      plumbing.Revision(commitish),
		OmitCommitHash: c.OmitCommitHash,
		ReleasePrefix:  c.VersionPrefix,
		IsPreRelease:   c.IsPreRelease,
		TagPattern:     c.TagPattern,
	}

	versions, err := vers.Calculate(opts)
	if err != nil {
		// If calculation fails (e.g., no git history), use fallback
		versions = vers.GenerateFallbackVersion()
	}

	if c.JSON {
		return json.NewEncoder(os.Stdout).Encode(versions)
	}

	output := getVersionOutput(versions, c.Language)
	fmt.Println(output)

	return nil
}

// isVersionString checks if the input looks like a version string rather than a git reference
func isVersionString(input string) bool {
	// First, check for obvious git references that should NOT be treated as versions
	if isGitReference(input) {
		return false
	}

	// Check if it looks like a semantic version (with or without 'v' prefix)
	trimmed := strings.TrimPrefix(input, "v")
	if trimmed == input && strings.HasPrefix(input, "v") {
		// If trimming 'v' didn't change anything but input starts with 'v',
		// it might be something like "version" which isn't a version string
		return false
	}

	// Must contain at least one dot for x.y or x.y.z format
	if !strings.Contains(trimmed, ".") {
		return false
	}

	// Split by dots and validate each part
	parts := strings.Split(trimmed, ".")
	if len(parts) < 2 {
		return false
	}

	// First part must be numeric
	if len(parts[0]) == 0 || !isNumeric(parts[0]) {
		return false
	}

	// Second part must be numeric
	if len(parts[1]) == 0 || !isNumeric(parts[1]) {
		return false
	}

	// If there's a third part, it should be numeric or contain pre-release info
	if len(parts) >= 3 && len(parts[2]) > 0 {
		// Allow formats like "1.2.3", "1.2.3-alpha", "1.2.3+build"
		patchPart := parts[2]
		// Split on '-' or '+' to separate patch from pre-release/build metadata
		if idx := strings.IndexAny(patchPart, "-+"); idx >= 0 {
			patchPart = patchPart[:idx]
		}
		if !isNumeric(patchPart) {
			return false
		}
	}

	return true
}

// isGitReference checks if the input looks like a git reference (branch, tag, commit hash, etc.)
func isGitReference(input string) bool {
	// Common git references that should not be treated as versions
	commonRefs := []string{
		"HEAD", "head", "main", "master", "develop", "dev", "trunk",
		"origin", "upstream",
	}

	for _, ref := range commonRefs {
		if strings.EqualFold(input, ref) {
			return true
		}
	}

	// Check for branch-like patterns (e.g., "feature/something", "release/v1.0")
	if strings.Contains(input, "/") {
		return true
	}

	// Check for commit hash patterns (7-40 hex characters)
	if len(input) >= 7 && len(input) <= 40 && isHexString(input) {
		return true
	}

	// Check for ref patterns like "refs/heads/main" or "refs/tags/v1.0.0"
	if strings.HasPrefix(input, "refs/") {
		return true
	}

	// Check for relative commit references like "HEAD~1", "HEAD^", "HEAD~10"
	if strings.Contains(input, "~") || strings.Contains(input, "^") {
		return true
	}

	return false
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range strings.ToLower(s) {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// isValidCommitishInRepo checks if the commitish exists in the actual git repository
func isValidCommitishInRepo(repo *git.Repository, commitish string) bool {
	// Check if it's a branch
	if isBranchInRepo(repo, commitish) {
		return true
	}

	// Check if it's a tag
	if isTagInRepo(repo, commitish) {
		return true
	}

	// Check if it's a commit hash
	if isCommitInRepo(repo, commitish) {
		return true
	}

	// Check if it's a common git reference like HEAD
	commonRefs := []string{"HEAD", "head"}
	for _, ref := range commonRefs {
		if strings.EqualFold(commitish, ref) {
			return true
		}
	}

	return false
}

// isBranchInRepo checks if the commitish is a valid branch in the repository
func isBranchInRepo(repo *git.Repository, branch string) bool {
	branches, err := repo.Branches()
	if err != nil {
		return false
	}

	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		if branchName == branch {
			return fmt.Errorf("found") // Use error to break out of ForEach
		}
		return nil
	})

	return err != nil // If we got an error, it means we found the branch
}

// isTagInRepo checks if the commitish is a valid tag in the repository
func isTagInRepo(repo *git.Repository, tag string) bool {
	tags, err := repo.Tags()
	if err != nil {
		return false
	}

	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()
		if tagName == tag {
			return fmt.Errorf("found") // Use error to break out of ForEach
		}
		return nil
	})

	return err != nil // If we got an error, it means we found the tag
}

// isCommitInRepo checks if the commitish is a valid commit hash in the repository
func isCommitInRepo(repo *git.Repository, commit string) bool {
	// Must look like a hex string and be at least 4 characters (git allows short hashes)
	if len(commit) < 4 || !isHexString(commit) {
		return false
	}

	// Try to resolve it as a commit hash
	hash := plumbing.NewHash(commit)
	_, err := repo.CommitObject(hash)
	if err == nil {
		return true
	}

	// If full hash doesn't work, try to find a commit that starts with this prefix
	// This handles short commit hashes
	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return false
	}
	defer iter.Close()

	err = iter.ForEach(func(c *object.Commit) error {
		if strings.HasPrefix(c.Hash.String(), commit) {
			return fmt.Errorf("found") // Use error to break out of ForEach
		}
		return nil
	})

	return err != nil // If we got an error, it means we found the commit
}

func getVersionOutput(versions *vers.LanguageVersions, language string) string {
	switch strings.ToLower(language) {
	case "generic", "semver":
		return versions.SemVer
	case "python":
		return versions.Python
	case "javascript", "js", "node":
		return versions.JavaScript
	case "dotnet", ".net", "csharp":
		return versions.DotNet
	case "go", "golang":
		return versions.Go
	default:
		return versions.SemVer
	}
}
