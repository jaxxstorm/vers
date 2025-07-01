package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/go-git/go-git/v5/plumbing"
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
	// Handle version flag
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
	versionInfo := map[string]string{
		"version": Version,
		"name":    "vers",
	}

	if c.JSON {
		return json.NewEncoder(os.Stdout).Encode(versionInfo)
	}

	fmt.Printf("vers version %s\n", Version)
	return nil
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
	// Simple heuristic: if it contains dots and starts with a number or 'v', treat as version
	if strings.Contains(input, ".") {
		trimmed := strings.TrimPrefix(input, "v")
		if len(trimmed) > 0 && (trimmed[0] >= '0' && trimmed[0] <= '9') {
			// Check if it has at least 2 dots (x.y.z format)
			parts := strings.Split(trimmed, ".")
			return len(parts) >= 3
		}
	}
	return false
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
