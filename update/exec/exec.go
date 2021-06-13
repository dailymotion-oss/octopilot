package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ExecUpdater struct {
	Command  string
	FilePath string
	Args     []string
	Stdout   string
	Stderr   string
	Timeout  time.Duration
}

func NewUpdater(params map[string]string) (*ExecUpdater, error) {
	updater := &ExecUpdater{}

	updater.Command = params["cmd"]
	if len(updater.Command) == 0 {
		return nil, errors.New("missing cmd parameter")
	}

	if file, ok := params["file"]; ok && len(strings.TrimSpace(file)) > 0 {
		updater.FilePath = file
	}

	if args, ok := params["args"]; ok {
		updater.Args = strings.Split(args, " ")
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

func (r *ExecUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	if r.Timeout > 0 {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(ctx, r.Timeout)
		defer cancelFunc()
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	if r.FilePath != "" {
		filePaths, err := filepath.Glob(filepath.Join(repoPath, r.FilePath))
		if err != nil {
			return false, fmt.Errorf("failed to expand glob pattern %s: %w", r.FilePath, err)
		}
		if len(filePaths) > 0 {
			r.Args = append(r.Args, filePaths...)
		}
	}

	cmd := exec.CommandContext(ctx, r.Command, r.Args...)
	cmd.Dir = repoPath
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run cmd '%s' with args %v - got stdout [%s] and stderr [%s]: %w", r.Command, r.Args, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err)
	}

	if len(r.Stdout) > 0 {
		if err = r.writeCmdOutputToFile(stdout, repoPath, r.Stdout); err != nil {
			return false, fmt.Errorf("failed to write stdout of cmd '%s' to %s: %w", r.Command, r.Stdout, err)
		}
	}
	if len(r.Stderr) > 0 {
		if err = r.writeCmdOutputToFile(stderr, repoPath, r.Stderr); err != nil {
			return false, fmt.Errorf("failed to write stderr of cmd '%s' to %s: %w", r.Command, r.Stderr, err)
		}
	}

	return true, nil
}

func (r *ExecUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Run %s", r.Command)
	body = fmt.Sprintf("Running command `%s`", r.Command)
	if len(r.Args) > 0 {
		body = fmt.Sprintf("%s with args %v", body, r.Args)
	}
	return title, body
}

func (r *ExecUpdater) String() string {
	return fmt.Sprintf("Exec[cmd=%s,args=%v]", r.Command, r.Args)
}

func (r *ExecUpdater) writeCmdOutputToFile(output bytes.Buffer, repoPath, filePath string) error {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(repoPath, filePath)
	}

	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
	}

	err = ioutil.WriteFile(filePath, output.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output of cmd '%s' to %s: %w", r.Command, filePath, err)
	}

	return nil
}
