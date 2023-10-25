// Package regex provides an updater that uses a regex to update files.
package regex

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/dailymotion-oss/octopilot/internal/glob"
	"github.com/dailymotion-oss/octopilot/update/value"
)

// RegexUpdater is an updater that uses a regex to update files.
type RegexUpdater struct {
	FilePath string
	Pattern  string
	Regexp   *regexp.Regexp
	Valuer   value.Valuer
}

// NewUpdater builds a new regex updater from the given parameters and valuer
func NewUpdater(params map[string]string, valuer value.Valuer) (*RegexUpdater, error) {
	updater := &RegexUpdater{}

	updater.FilePath = params["file"]
	if len(updater.FilePath) == 0 {
		return nil, errors.New("missing file parameter")
	}

	updater.Pattern = params["pattern"]
	if len(updater.Pattern) == 0 {
		return nil, errors.New("missing pattern parameter")
	}

	var err error
	updater.Regexp, err = regexp.Compile(updater.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %s: %w", updater.Pattern, err)
	}
	if subexp := updater.Regexp.NumSubexp(); subexp != 1 {
		return nil, fmt.Errorf("invalid pattern %s: it must have a single parenthesized subexpression, but it has %d", updater.Pattern, subexp)
	}

	updater.Valuer = valuer

	return updater, nil
}

// Update updates the repository cloned at the given path, and returns true if changes have been made
func (u RegexUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	value, err := u.Valuer.Value(ctx, repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to get value: %w", err)
	}

	filePaths, err := glob.ExpandGlobPattern(repoPath, u.FilePath)
	if err != nil {
		return false, fmt.Errorf("failed to expand glob pattern %s: %w", u.FilePath, err)
	}

	var updated bool
	for _, filePath := range filePaths {
		relFilePath, err := filepath.Rel(repoPath, filePath)
		if err != nil {
			relFilePath = filePath
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to access file %s: %w", relFilePath, err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to read file %s: %w", relFilePath, err)
		}

		if !u.Regexp.Match(content) {
			continue
		}

		var (
			updatedContent  bytes.Buffer
			currentPosition int
		)
		allIndexes := u.Regexp.FindAllSubmatchIndex(content, -1)
		for _, indexes := range allIndexes {
			if len(indexes) == 4 {
				valueStartPosition := indexes[2]
				valueEndPosition := indexes[3]
				if _, err = updatedContent.Write(content[currentPosition:valueStartPosition]); err != nil {
					return false, fmt.Errorf("failed to copy existing content to the buffer: %w", err)
				}
				if _, err = updatedContent.WriteString(value); err != nil {
					return false, fmt.Errorf("failed to write new value to the buffer: %w", err)
				}
				currentPosition = valueEndPosition
			}
		}
		if _, err = updatedContent.Write(content[currentPosition:]); err != nil {
			return false, fmt.Errorf("failed to copy existing content to the buffer: %w", err)
		}

		if err = os.WriteFile(filePath, updatedContent.Bytes(), fileInfo.Mode()); err != nil {
			return false, fmt.Errorf("failed to write updated content to file %s: %w", relFilePath, err)
		}

		updated = true
	}

	return updated, nil
}

// Message returns the default title and body that should be used in the commits / pull requests
func (u RegexUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Update %s", u.FilePath)
	body = fmt.Sprintf("Updating file(s) `%s` using pattern `%s`", u.FilePath, u.Pattern)
	return title, body
}

// String returns a string representation of the updater
func (u RegexUpdater) String() string {
	return fmt.Sprintf("Regex[pattern=%s,file=%s]", u.Pattern, u.FilePath)
}
