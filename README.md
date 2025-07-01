# Vers

A Go library and CLI tool for calculating semantic versions from Git repository state.

## Features

- **Multiple Language Support**: Generate versions for Go, Python, JavaScript, .NET, and generic SemVer
- **Git Integration**: Automatically calculates versions based on Git tags and repository state
- **Pre-release Support**: Handles alpha, beta, rc, and dev pre-release versions
- **Dirty Detection**: Detects uncommitted changes and marks versions accordingly
- **Tag Filtering**: Support for filtering tags with regex patterns
- **JSON Output**: CLI supports JSON output for automation
- **Clean API**: Simple Go library interface
- **Graceful Fallbacks**: Works in non-Git directories with sensible default versions
- **Unified CLI**: Single command interface for both Git analysis and version conversion

## Installation

### CLI Tool
```bash
go install github.com/jaxxstorm/vers/cmd@latest
```

### Library
```bash
go get github.com/jaxxstorm/vers
```

## CLI Usage

### Basic Usage
```bash
# Calculate version for current HEAD
vers

# Calculate version for specific commit/tag
vers v1.2.3

# Convert existing version string
vers 1.2.3-alpha.1

# Get Python-compatible version
vers --language python

# Get version as JSON
vers --json

# Filter tags with pattern
vers --tag-pattern "^v"

# Mark as pre-release
vers --is-prerelease

# Show version information
vers --version
```

### Non-Git Directories
When run in a directory that's not a Git repository or has no Git history, `vers` will automatically generate a sensible fallback version:
- SemVer: `0.0.0-dev`
- Python: `0.0.0.dev0`
- JavaScript: `v0.0.0-dev`
- .NET: `0.0.0-dev`
- Go: `v0.0.0-dev`

### Available Language Formats
- `generic` / `semver` - Standard semantic versioning
- `python` - PEP440 compatible versioning
- `javascript` / `js` / `node` - Node.js/npm compatible
- `dotnet` / `csharp` - .NET compatible
- `go` / `golang` - Go module compatible

## Library Usage

### Basic Example
```go
package main

import (
    "fmt"
    "log"

    "github.com/jaxxstorm/vers"
)

func main() {
    // Open repository
    repo, err := vers.OpenRepository(".")
    if err != nil {
        // Handle non-git directories gracefully
        versions := vers.GenerateFallbackVersion()
        fmt.Printf("Fallback SemVer: %s\n", versions.SemVer)
        return
    }

    // Calculate versions
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
    fmt.Printf("Go: %s\n", versions.Go)
}
```

### Advanced Example
```go
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

    // Advanced options
    opts := vers.Options{
        Repository:     repo,
        Commitish:      plumbing.Revision("v1.2.3"),
        OmitCommitHash: true,
        ReleasePrefix:  "2.0.0",
        IsPreRelease:   true,
        TagPattern:     "^v\\d+\\.\\d+\\.\\d+$", // Only version tags
    }
    
    versions, err := vers.Calculate(opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Versions: %+v\n", versions)
}
```

### Convert Existing Versions
```go
package main

import (
    "fmt"
    "log"

    "github.com/jaxxstorm/vers"
)

func main() {
    // Convert an existing version string
    versions, err := vers.CalculateFromString("1.2.3-alpha.1")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Python: %s\n", versions.Python)
    fmt.Printf("JavaScript: %s\n", versions.JavaScript)
    fmt.Printf(".NET: %s\n", versions.DotNet)
}
```

## API Reference

### Types

#### `LanguageVersions`
Contains version strings formatted for different language ecosystems:
- `SemVer` - Standard semantic version
- `Python` - PEP440 compatible version
- `JavaScript` - Node.js/npm compatible version  
- `DotNet` - .NET compatible version
- `Go` - Go module compatible version

#### `Options`
Configuration for version calculation:
- `Repository` - Git repository to analyze (required)
- `Commitish` - Git commitish to analyze (default: "HEAD")
- `OmitCommitHash` - Exclude commit hash from versions
- `ReleasePrefix` - Override version prefix (e.g., "3.0.0")
- `IsPreRelease` - Mark as pre-release version
- `TagFilter` - Function to filter which tags to consider
- `TagPattern` - Regex pattern to filter tags

### Functions

#### `OpenRepository(path string) (*git.Repository, error)`
Opens a Git repository at the specified path.

#### `Calculate(opts Options) (*LanguageVersions, error)`
Calculates version strings based on Git repository state and tags.

#### `CalculateFromString(version string) (*LanguageVersions, error)`
Converts an existing version string to different language formats.

#### `GenerateFallbackVersion() *LanguageVersions`
Creates a default development version (0.0.0-dev variants) for use when Git is unavailable or repositories have no history.

## Version Calculation Logic

Vers uses the following logic to determine versions:

1. **Exact Tag Match**: If the current commit has a tag, use that version
2. **Recent Tag**: Find the most recent reachable tag and increment appropriately
3. **Default**: Use "0.0.0" if no tags are found

### Version Increments
- For versions `< 1.0.0`: Increment patch version
- For versions `>= 1.0.0`: Increment minor version
- Non-exact matches get `-alpha` pre-release suffix

### Dirty Detection
When uncommitted changes are detected:
- Adds `-dirty` suffix to development versions
- Uses native Git commands for performance when possible

## Language-Specific Formatting

### Python (PEP440)
- Converts `-alpha` to `a`
- Converts `-beta` to `b` 
- Converts `-rc` to `rc`
- Example: `1.2.3-alpha.1` → `1.2.3a1`

### JavaScript/Node.js
- Adds `v` prefix
- Example: `1.2.3` → `v1.2.3`

### .NET
- Standard semantic versioning format
- Example: `1.2.3-alpha.1`

### Go
- Adds `v` prefix for module compatibility
- Example: `1.2.3` → `v1.2.3`

## Testing

The project includes comprehensive unit tests covering all functionality:

```bash
# Run all tests
go test -v ./...

# Run specific test packages
go test -v .              # Library tests
go test -v ./cmd          # CLI tests

# Run tests with coverage
go test -cover ./...
```

### Test Coverage

The test suite covers:
- Git repository analysis and tag handling
- Version calculation logic for all language formats
- CLI argument parsing and output formatting
- Fallback behavior for non-Git directories
- Error handling and edge cases
- Version string conversion between formats

## Attribution

This project contains code adapted from [pulumictl](https://github.com/pulumi/pulumictl), which is licensed under the Apache License 2.0. The adapted code includes Git repository analysis, version calculation logic, and core data structures. See the [NOTICE](NOTICE) file for detailed attribution information.

## License

Apache 2.0 License - see LICENSE file for details.

