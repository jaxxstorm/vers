package main

import (
	"fmt"
	"log"
	"regexp"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jaxxstorm/vers"
)

func main() {
	repo, err := vers.OpenRepository(".")
	if err != nil {
		log.Fatal(err)
	}

	// Advanced options with tag filtering and custom configuration
	opts := vers.Options{
		Repository:     repo,
		Commitish:      plumbing.Revision("HEAD"),
		OmitCommitHash: true,
		ReleasePrefix:  "2.0.0",
		IsPreRelease:   true,
		TagPattern:     "^v\\d+\\.\\d+\\.\\d+$", // Only version tags like v1.2.3
	}

	versions, err := vers.Calculate(opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Advanced calculation results:\n")
	fmt.Printf("SemVer: %s\n", versions.SemVer)
	fmt.Printf("Python: %s\n", versions.Python)
	fmt.Printf("JavaScript: %s\n", versions.JavaScript)
	fmt.Printf(".NET: %s\n", versions.DotNet)
	fmt.Printf("Go: %s\n", versions.Go)

	// Example with custom tag filter function
	optsWithFilter := vers.Options{
		Repository: repo,
		Commitish:  plumbing.Revision("HEAD"),
		TagFilter: func(tag string) bool {
			// Only consider tags that start with "release/"
			matched, _ := regexp.MatchString("^release/", tag)
			return matched
		},
	}

	filteredVersions, err := vers.Calculate(optsWithFilter)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nWith custom tag filter:\n")
	fmt.Printf("SemVer: %s\n", filteredVersions.SemVer)
}
