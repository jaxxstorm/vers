package main

import (
	"fmt"
	"log"

	"github.com/jaxxstorm/vers"
)

func main() {
	// Convert existing version strings to different formats
	examples := []string{
		"1.2.3",
		"v2.0.0",
		"1.5.0-alpha.1",
		"3.0.0-beta.2",
		"4.1.0-rc.1",
	}

	for _, version := range examples {
		fmt.Printf("Converting version: %s\n", version)

		versions, err := vers.CalculateFromString(version)
		if err != nil {
			log.Printf("Error converting %s: %v", version, err)
			continue
		}

		fmt.Printf("  SemVer:     %s\n", versions.SemVer)
		fmt.Printf("  Python:     %s\n", versions.Python)
		fmt.Printf("  JavaScript: %s\n", versions.JavaScript)
		fmt.Printf("  .NET:       %s\n", versions.DotNet)
		fmt.Printf("  Go:         %s\n", versions.Go)
		fmt.Println()
	}

	// Example of invalid version handling
	invalid := "1.2"
	fmt.Printf("Trying invalid version: %s\n", invalid)
	_, err := vers.CalculateFromString(invalid)
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}
}
