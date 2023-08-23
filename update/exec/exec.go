// Package exec provides an updater that executes an external command to update the repository.
package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cosiner/argv"
)

// ExecUpdater is an updater that executes an external command to update the repository.
type ExecUpdater struct {
	Command string
	Path    string
	Args    []string
	Stdout  string
	Stderr  string
	Timeout time.Duration
}

// NewUpdater builds a new exec updater from the given parameters
func NewUpdater(params map[string]string) (*ExecUpdater, error) {
	updater := &ExecUpdater{}

	updater.Command = params["cmd"]
	if len(updater.Command) == 0 {
		return nil, errors.New("missing cmd parameter")
	}

	updater.Path = params["path"]

	if args, ok := params["args"]; ok {
		argv, err := argv.Argv(args, func(backquoted string) (string, error) {
			return backquoted, nil
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to convert args '%s' with argv: %w", args, err)
		}
		if len(argv) > 0 {
			updater.Args = argv[0]
		}
	}
	updater.Stdout = params["stdout"]
	updater.Stderr = params["stderr"]

	timeout := params["timeout"]
	if len(timeout) > 0 {
		var err error
		updater.Timeout, err = time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration for cmd timeout '%s': %w", timeout, err)
		}
	}

	return updater, nil
}

// Update updates the repository cloned at the given path, and returns true if changes have been made
func (u *ExecUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	if u.Timeout > 0 {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(ctx, u.Timeout)
		defer cancelFunc()
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.CommandContext(ctx, u.Command, u.Args...)
	cmd.Dir = repoPath + u.Path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run cmd '%s' with args %v in path: %s - got stdout [%s] and stderr [%s]: %w", u.Command, u.Args, u.Path, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err)
	}

	if len(u.Stdout) > 0 {
		if err = u.writeCmdOutputToFile(stdout, repoPath, u.Stdout); err != nil {
			return false, fmt.Errorf("failed to write stdout of cmd '%s' to %s: %w", u.Command, u.Stdout, err)
		}
	}
	if len(u.Stderr) > 0 {
		if err = u.writeCmdOutputToFile(stderr, repoPath, u.Stderr); err != nil {
			return false, fmt.Errorf("failed to write stderr of cmd '%s' to %s: %w", u.Command, u.Stderr, err)
		}
	}

	return true, nil
}

// Message returns the default title and body that should be used in the commits / pull requests
func (u *ExecUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Run %s", u.Command)
	body = fmt.Sprintf("Running command `%s`", u.Command)
	if len(u.Args) > 0 {
		body = fmt.Sprintf("%s with args %v", body, u.Args)
	}
	return title, body
}

// String returns a string representation of the updater
func (u *ExecUpdater) String() string {
	return fmt.Sprintf("Exec[cmd=%s,args=%v]", u.Command, u.Args)
}

func (u *ExecUpdater) writeCmdOutputToFile(output bytes.Buffer, repoPath, filePath string) error {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(repoPath, filePath)
	}

	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
	}

	err = os.WriteFile(filePath, output.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output of cmd '%s' to %s: %w", u.Command, filePath, err)
	}

	return nil
}
