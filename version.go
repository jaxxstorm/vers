// Package vers provides semantic versioning utilities for Git repositories.
//
// This file contains code adapted from pulumictl (https://github.com/pulumi/pulumictl)
// which is licensed under the Apache License 2.0. See NOTICE file for full attribution.
package vers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"
)

// Calculate determines version strings for multiple language ecosystems
// based on Git repository state and tags
func Calculate(opts Options) (*LanguageVersions, error) {
	if opts.Repository == nil {
		return nil, fmt.Errorf("repository is required")
	}

	if opts.Commitish == "" {
		opts.Commitish = "HEAD"
	}

	// Apply tag pattern filter if specified
	if opts.TagPattern != "" && opts.TagFilter == nil {
		re, err := regexp.Compile(opts.TagPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid tag pattern: %w", err)
		}
		opts.TagFilter = func(tag string) bool {
			return re.MatchString(tag)
		}
	}

	components, err := getVersionComponents(opts)
	if err != nil {
		return nil, fmt.Errorf("calculating version components: %w", err)
	}

	return buildLanguageVersions(components, opts)
}

// CalculateFromString parses an existing version string and converts it
// to different language-specific formats
func CalculateFromString(version string) (*LanguageVersions, error) {
	// Strip leading "v" if present
	normalised := strings.TrimPrefix(version, "v")

	parts := strings.SplitN(normalised, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("version must have exactly 3 parts: %q", version)
	}

	major, minor := parts[0], parts[1]
	patch := parts[2]

	pythonPatch, err := convertPatchToPython(patch)
	if err != nil {
		return nil, fmt.Errorf("converting patch for Python: %w", err)
	}

	genericVersion := fmt.Sprintf("%s.%s.%s", major, minor, patch)
	pythonVersion := fmt.Sprintf("%s.%s.%s", major, minor, pythonPatch)
	jsVersion := fmt.Sprintf("v%s", genericVersion)
	dotnetVersion := genericVersion
	goVersion := fmt.Sprintf("v%s", genericVersion)

	return &LanguageVersions{
		SemVer:     genericVersion,
		Python:     pythonVersion,
		JavaScript: jsVersion,
		DotNet:     dotnetVersion,
		Go:         goVersion,
	}, nil
}

func buildLanguageVersions(components *VersionComponents, opts Options) (*LanguageVersions, error) {
	// Build generic semantic version
	genericVersion := semver.Version{
		Major: components.Semver.Major,
		Minor: components.Semver.Minor,
		Patch: components.Semver.Patch,
		Build: components.Semver.Build,
	}

	// Handle pre-release versions
	if len(components.Semver.Pre) == 1 {
		genericVersion.Pre = []semver.PRVersion{
			components.Semver.Pre[0],
			{VersionStr: strconv.FormatInt(components.Timestamp.UTC().Unix(), 10)},
		}
	} else if len(components.Semver.Pre) > 1 {
		genericVersion.Pre = components.Semver.Pre
	}

	// Build version strings for each language
	baseVersion := fmt.Sprintf("%d.%d.%d",
		genericVersion.Major, genericVersion.Minor, genericVersion.Patch)

	preVersion, pythonPreVersion, err := buildPreVersionStrings(
		genericVersion, components, opts)
	if err != nil {
		return nil, err
	}

	// Add dirty suffix if needed
	if components.Dirty {
		separator := "."
		if preVersion == "" {
			separator = "+"
		}
		pythonPreVersion += "+dirty"
		preVersion += separator + "dirty"
	}

	// Build final versions
	version := baseVersion + preVersion
	pythonVersion := baseVersion + pythonPreVersion
	jsVersion := "v" + version
	dotnetVersion := version
	goVersion := "v" + version

	return &LanguageVersions{
		SemVer:     version,
		Python:     pythonVersion,
		JavaScript: jsVersion,
		DotNet:     dotnetVersion,
		Go:         goVersion,
	}, nil
}

func buildPreVersionStrings(genericVersion semver.Version, components *VersionComponents, opts Options) (string, string, error) {
	if len(genericVersion.Pre) == 0 {
		return "", "", nil
	}

	var preSuffix string
	if !components.IsExact {
		preSuffix = fmt.Sprintf(".%d", components.Timestamp.UTC().Unix())
	} else if len(genericVersion.Pre) > 1 {
		preSuffix = fmt.Sprintf(".%d", genericVersion.Pre[1].VersionNum)
	}

	pythonPreSuffix := "0"
	if preSuffix != "" {
		pythonPreSuffix = preSuffix[1:] // Remove leading "."
	}

	shortHash := ""
	if !opts.OmitCommitHash && !opts.IsPreRelease {
		shortHash = fmt.Sprintf("+%s", components.ShortHash)
	}

	preType := genericVersion.Pre[0].VersionStr

	var preVersion, pythonPreVersion string
	switch preType {
	case "dev":
		pythonPreVersion = fmt.Sprintf("dev%s", pythonPreSuffix)
		preVersion = fmt.Sprintf("-dev%s%s", preSuffix, shortHash)
	case "alpha":
		pythonPreVersion = fmt.Sprintf("a%s", pythonPreSuffix)
		preVersion = fmt.Sprintf("-alpha%s%s", preSuffix, shortHash)
	case "beta":
		pythonPreVersion = fmt.Sprintf("b%s", pythonPreSuffix)
		preVersion = fmt.Sprintf("-beta%s%s", preSuffix, shortHash)
	case "rc":
		pythonPreVersion = fmt.Sprintf("rc%s", pythonPreSuffix)
		preVersion = fmt.Sprintf("-rc%s%s", preSuffix, shortHash)
	default:
		return "", "", fmt.Errorf("invalid prerelease type: %q", preType)
	}

	return preVersion, pythonPreVersion, nil
}

// convertPatchToPython converts a patch version to Python-compatible format
func convertPatchToPython(patch string) (string, error) {
	re := regexp.MustCompile(`^(\d+)(.*)$`)
	matches := re.FindStringSubmatch(patch)
	if len(matches) != 3 {
		return patch, nil
	}

	version := matches[1]
	pre := matches[2]

	pythonPre, err := getPythonPreVersion(pre)
	if err != nil {
		return "", err
	}

	return version + pythonPre, nil
}

func getPythonPreVersion(preVersion string) (string, error) {
	if preVersion == "" {
		return "", nil
	}

	prefix, remaining := getPythonPrePrefix(preVersion)
	isDirty := strings.Contains(preVersion, "dirty")

	if isDirty {
		remaining = strings.Replace(remaining, "dirty", "", 1)
	}

	// Remove hash to avoid confusion with build numbers
	hashRe := regexp.MustCompile(`\+[0-9a-f]{8}\b`)
	hashMatches := hashRe.FindAllString(remaining, 5)
	if len(hashMatches) > 0 {
		shortHash := hashMatches[len(hashMatches)-1]
		remaining = strings.Replace(remaining, shortHash, "", 1)
	}

	// Extract version number
	numRe := regexp.MustCompile(`\W(\d+)(\W|$)`)
	nums := numRe.FindStringSubmatch(remaining)

	pythonPreSuffix := "0" // Default for PEP440 compliance
	if len(nums) == 3 {
		pythonPreSuffix = nums[1]
	}

	pythonPreVersion := ""
	if prefix != "" {
		pythonPreVersion = fmt.Sprintf("%s%s", prefix, pythonPreSuffix)
	}

	if isDirty {
		pythonPreVersion += "+dirty"
	}

	return pythonPreVersion, nil
}

func getPythonPrePrefix(preVersion string) (string, string) {
	switch {
	case strings.HasPrefix(preVersion, "-dev"):
		return "dev", preVersion[4:]
	case strings.HasPrefix(preVersion, "-alpha"):
		return "a", preVersion[6:]
	case strings.HasPrefix(preVersion, "-beta"):
		return "b", preVersion[5:]
	case strings.HasPrefix(preVersion, "-rc"):
		return "rc", preVersion[3:]
	default:
		return "", preVersion
	}
}

// GenerateFallbackVersion creates a default development version when git is unavailable
func GenerateFallbackVersion() *LanguageVersions {
	return &LanguageVersions{
		SemVer:     "0.0.0-dev",
		Python:     "0.0.0.dev0",
		JavaScript: "v0.0.0-dev",
		DotNet:     "0.0.0-dev",
		Go:         "v0.0.0-dev",
	}
}
