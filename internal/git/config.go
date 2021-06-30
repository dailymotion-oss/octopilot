package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configformat "github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/mitchellh/go-homedir"
)

// ConfigValue returns the "best" value for the given configuration key.
// It respects the "git config" algorithm to search for a value:
// - first, the local git repository based on the working directory
// - then the global git config, either in $HOME/.gitconfig or $XDG_CONFIG_HOME/git/config or $HOME/.config/git/config
// - and finally the system git config, from /etc/gitconfig
// If not found, it will return an empty string.
// The key syntax is the same as used in the "git config" cmdline: a dot-separated path of elements.
// For example: `user.name` or `remote.origin.url`.
func ConfigValue(key string) string {
	// local $PWD/.git/config
	// or any .git directory in the upper hierarchy
	gitDir, err := findGitDirectory()
	if err == nil {
		if value := getConfigValue(filepath.Join(gitDir, "config"), key); len(value) > 0 {
			return value
		}
	}

	// global $HOME/.gitconfig
	homeDir, err := homedir.Dir()
	if err == nil {
		if value := getConfigValue(filepath.Join(homeDir, ".gitconfig"), key); len(value) > 0 {
			return value
		}
	}

	// global $XDG_CONFIG_HOME/git/config
	if value := getConfigValue(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "git", "config"), key); len(value) > 0 {
		return value
	}

	// global $HOME/.config/git/config
	if len(homeDir) > 0 {
		if value := getConfigValue(filepath.Join(homeDir, ".config", "git", "config"), key); len(value) > 0 {
			return value
		}
	}

	// system /etc/gitconfig
	if value := getConfigValue("/etc/gitconfig", key); len(value) > 0 {
		return value
	}

	return ""
}

// CurrentRepositoryURL returns the URL of the current git repository, based on the "origin" remote URL.
// If no repository could be found (from the current working directory), or if there is no "origin" remote
// configured, it will return an empty string.
func CurrentRepositoryURL() string {
	gitDir, err := findGitDirectory()
	if err != nil {
		return ""
	}
	repoURL := getConfigValue(filepath.Join(gitDir, "config"), "remote.origin.url")
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimPrefix(repoURL, "git@github.com:")
	if len(repoURL) > 0 && !strings.HasPrefix(repoURL, "http") {
		repoURL = "https://github.com/" + repoURL
	}
	return repoURL
}

// getConfigValue returns the value associated with the given key in the given git config file path.
// The key syntax is the same as used in the "git config" cmdline: a dot-separated path of elements.
// For example: `user.name` or `remote.origin.url`.
// In case of error or if no value could be found, it will just return an empty string.
func getConfigValue(configFilePath string, key string) string {
	file, err := os.Open(configFilePath)
	if err != nil {
		return ""
	}
	defer file.Close() // nolint: errcheck

	cfg := configformat.New()
	if err = configformat.NewDecoder(file).Decode(cfg); err != nil {
		return ""
	}

	keyElems := strings.Split(key, ".")
	if len(keyElems) < 2 {
		return ""
	}

	section := cfg.Section(keyElems[0])
	if section == nil {
		return ""
	}

	switch len(keyElems) {
	case 2:
		return section.Option(keyElems[1])
	case 3:
		subSection := section.Subsection(keyElems[1])
		if subSection == nil {
			return ""
		}
		return subSection.Option(keyElems[2])
	default:
		return ""
	}
}

// findGitDirectory returns the path of the local ".git" directory, based on the working directory.
// It starts at the working directory, and walks up the filesystem hierarchy until it finds a valid ".git" directory.
// If it can't retrieve the working directory, and can't find a ".git" directory it will return an error.
func findGitDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	dir := wd
	for {
		fi, _ := os.Stat(filepath.Join(dir, ".git", "config"))
		if fi != nil && !fi.IsDir() {
			return filepath.Join(dir, ".git"), nil
		}

		if len(dir) == 0 || (len(dir) == 1 && os.IsPathSeparator(dir[0])) {
			return "", fmt.Errorf("failed to find a .git directory starting from %s", wd)
		}

		dir = strings.TrimSuffix(dir, string(os.PathSeparator))
		dir = filepath.Dir(dir)
	}
}
