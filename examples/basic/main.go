package main

import (
	"fmt"

	"github.com/jaxxstorm/vers"
)

func main() {
	// Basic usage: get version for current repository
	repo, err := vers.OpenRepository(".")
	if err != nil {
		// Handle non-git directories gracefully
		versions := vers.GenerateFallbackVersion()
		fmt.Printf("Fallback SemVer: %s\n", versions.SemVer)
		fmt.Printf("Fallback Python: %s\n", versions.Python)
		fmt.Printf("Fallback Go: %s\n", versions.Go)
		return
	}

	// Calculate versions for HEAD
	opts := vers.Options{
		Repository: repo,
		Commitish:  "HEAD",
	}

	versions, err := vers.Calculate(opts)
	if err != nil {
		// Handle repositories with no history
		versions = vers.GenerateFallbackVersion()
	}

	fmt.Printf("SemVer: %s\n", versions.SemVer)
	fmt.Printf("Python: %s\n", versions.Python)
	fmt.Printf("JavaScript: %s\n", versions.JavaScript)
	fmt.Printf(".NET: %s\n", versions.DotNet)
	fmt.Printf("Go: %s\n", versions.Go)
}
