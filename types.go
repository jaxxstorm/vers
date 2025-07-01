// Package vers provides semantic versioning utilities for Git repositories.
//
// This file contains code adapted from pulumictl (https://github.com/pulumi/pulumictl)
// which is licensed under the Apache License 2.0. See NOTICE file for full attribution.
package vers

import (
	"time"

	"github.com/blang/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// LanguageVersions contains version strings for different language ecosystems
type LanguageVersions struct {
	SemVer     string `json:"semver"`
	Python     string `json:"python"`
	JavaScript string `json:"javascript"`
	DotNet     string `json:"dotnet"`
	Go         string `json:"go"`
}

// Options configures version calculation behavior
type Options struct {
	// Repository is the Git repository to analyze
	Repository *git.Repository

	// Commitish specifies which commit to analyze (default: "HEAD")
	Commitish plumbing.Revision

	// OmitCommitHash excludes commit hash from non-release versions
	OmitCommitHash bool

	// ReleasePrefix overrides the version prefix (e.g., "3.0.0")
	ReleasePrefix string

	// IsPreRelease indicates this is a pre-release build
	IsPreRelease bool

	// TagFilter allows filtering which tags to consider
	TagFilter func(string) bool

	// TagPattern is a regex pattern to filter tags (alternative to TagFilter)
	TagPattern string
}

// VersionComponents contains the raw components used for version calculation
type VersionComponents struct {
	Semver    semver.Version
	Dirty     bool
	ShortHash string
	Timestamp time.Time
	IsExact   bool
}
