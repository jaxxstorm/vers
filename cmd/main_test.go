package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/jaxxstorm/vers"
	"github.com/stretchr/testify/require"
)

func TestIsVersionString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1.2.3", true},
		{"v1.2.3", true},
		{"1.2.3-alpha.1", true},
		{"v2.0.0-beta.2", true},
		{"HEAD", false},
		{"main", false},
		{"feature/branch", false},
		{"abc123def", false},
		{"", false},
		{"1.2", false}, // Not enough parts
		{"v", false},
		{"1", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := isVersionString(test.input)
			require.Equal(t, test.expected, result, "Input: %s", test.input)
		})
	}
}

func TestGetVersionOutput(t *testing.T) {
	versions := &vers.LanguageVersions{
		SemVer:     "1.2.3",
		Python:     "1.2.3",
		JavaScript: "v1.2.3",
		DotNet:     "1.2.3",
		Go:         "v1.2.3",
	}

	tests := []struct {
		language string
		expected string
	}{
		{"generic", "1.2.3"},
		{"semver", "1.2.3"},
		{"python", "1.2.3"},
		{"javascript", "v1.2.3"},
		{"js", "v1.2.3"},
		{"node", "v1.2.3"},
		{"dotnet", "1.2.3"},
		{".net", "1.2.3"},
		{"csharp", "1.2.3"},
		{"go", "v1.2.3"},
		{"golang", "v1.2.3"},
		{"unknown", "1.2.3"}, // Should default to SemVer
	}

	for _, test := range tests {
		t.Run(test.language, func(t *testing.T) {
			result := getVersionOutput(versions, test.language)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestCLIShowVersion(t *testing.T) {
	cli := &CLI{ShowVersion: true}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.showVersion()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	output, _ := ioutil.ReadAll(r)
	outputStr := string(output)

	require.Contains(t, outputStr, "vers version")
	require.Contains(t, outputStr, "dev") // Default version should be "dev"
}

func TestCLIShowVersionJSON(t *testing.T) {
	cli := &CLI{ShowVersion: true, JSON: true}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.showVersion()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	output, _ := ioutil.ReadAll(r)

	var versionInfo map[string]string
	err = json.Unmarshal(output, &versionInfo)
	require.NoError(t, err)

	require.Equal(t, "dev", versionInfo["version"])
	require.Equal(t, "vers", versionInfo["name"])
}

func TestCLIConvertVersion(t *testing.T) {
	cli := &CLI{Commitish: "1.2.3", Language: "python"}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.convertVersion()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	output, _ := ioutil.ReadAll(r)
	outputStr := strings.TrimSpace(string(output))

	require.Equal(t, "1.2.3", outputStr)
}

func TestCLIConvertVersionJSON(t *testing.T) {
	cli := &CLI{Commitish: "1.2.3", JSON: true}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cli.convertVersion()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	output, _ := ioutil.ReadAll(r)

	var versions vers.LanguageVersions
	err = json.Unmarshal(output, &versions)
	require.NoError(t, err)

	require.Equal(t, "1.2.3", versions.SemVer)
	require.Equal(t, "1.2.3", versions.Python)
	require.Equal(t, "v1.2.3", versions.JavaScript)
	require.Equal(t, "1.2.3", versions.DotNet)
	require.Equal(t, "v1.2.3", versions.Go)
}

func TestCLICalculateVersionNonGitRepo(t *testing.T) {
	// Create a temporary non-git directory
	tmpDir, err := ioutil.TempDir("", "non-git")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cli := &CLI{Repo: tmpDir, Language: "generic"}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cli.calculateVersion()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	output, _ := ioutil.ReadAll(r)
	outputStr := strings.TrimSpace(string(output))

	require.Equal(t, "0.0.0-dev", outputStr)
}

func TestCLICalculateVersionNonGitRepoJSON(t *testing.T) {
	// Create a temporary non-git directory
	tmpDir, err := ioutil.TempDir("", "non-git")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cli := &CLI{Repo: tmpDir, JSON: true}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cli.calculateVersion()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	output, _ := ioutil.ReadAll(r)

	var versions vers.LanguageVersions
	err = json.Unmarshal(output, &versions)
	require.NoError(t, err)

	// Should get fallback versions
	require.Equal(t, "0.0.0-dev", versions.SemVer)
	require.Equal(t, "0.0.0.dev0", versions.Python)
	require.Equal(t, "v0.0.0-dev", versions.JavaScript)
	require.Equal(t, "0.0.0-dev", versions.DotNet)
	require.Equal(t, "v0.0.0-dev", versions.Go)
}

func TestCLIRun(t *testing.T) {
	t.Run("Show version", func(t *testing.T) {
		cli := &CLI{ShowVersion: true}

		// Capture stdout to avoid polluting test output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cli.Run()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		output, _ := ioutil.ReadAll(r)
		require.Contains(t, string(output), "vers version")
	})

	t.Run("Convert version", func(t *testing.T) {
		cli := &CLI{Commitish: "1.2.3"}

		// Capture stdout to avoid polluting test output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cli.Run()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		output, _ := ioutil.ReadAll(r)
		require.Equal(t, "1.2.3\n", string(output))
	})

	t.Run("Calculate version in non-git directory", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "non-git")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		cli := &CLI{Repo: tmpDir}

		// Capture stdout to avoid polluting test output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err = cli.Run()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		output, _ := ioutil.ReadAll(r)
		require.Equal(t, "0.0.0-dev\n", string(output))
	})
}
