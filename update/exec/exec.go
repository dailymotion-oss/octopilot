package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ExecUpdater struct {
	Command string
	Args    string
	Timeout time.Duration
}

func NewUpdater(params map[string]string) (*ExecUpdater, error) {
	updater := &ExecUpdater{}

	updater.Command = params["cmd"]
	if len(updater.Command) == 0 {
		return nil, errors.New("missing cmd parameter")
	}

	updater.Args = params["args"]

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
	var args []string
	if len(r.Args) > 0 {
		args = []string{r.Args}
	}

	if r.Timeout > 0 {
		var cancelFunc context.CancelFunc
		ctx, cancelFunc = context.WithTimeout(ctx, r.Timeout)
		defer cancelFunc()
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.CommandContext(ctx, r.Command, args...)
	cmd.Dir = repoPath
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run cmd '%s' with args '%s' - got stdout [%s] and stderr [%s]: %w", r.Command, r.Args, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err)
	}

	return true, nil
}

func (r *ExecUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Run %s", r.Command)
	body = fmt.Sprintf("Running command `%s`", r.Command)
	if len(r.Args) > 0 {
		body = fmt.Sprintf("%s with args '%s'", body, r.Args)
	}
	return title, body
}

func (r *ExecUpdater) String() string {
	return fmt.Sprintf("Exec[cmd=%s]", r.Command)
}
