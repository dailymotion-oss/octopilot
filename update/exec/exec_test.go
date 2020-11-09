package exec

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		params           map[string]string
		expected         *ExecUpdater
		expectedErrorMsg string
	}{
		{
			name: "valid params with cmd",
			params: map[string]string{
				"cmd": "ls",
			},
			expected: &ExecUpdater{
				Command: "ls",
			},
		},
		{
			name: "valid params with cmd and multiple args",
			params: map[string]string{
				"cmd":  "ls",
				"args": "one two three",
			},
			expected: &ExecUpdater{
				Command: "ls",
				Args:    []string{"one", "two", "three"},
			},
		},
		{
			name: "valid params with cmd, args, stdout, stderr and timeout",
			params: map[string]string{
				"cmd":     "ls",
				"args":    "-lh",
				"stdout":  "/path/to/some/file",
				"stderr":  "/path/to/some/other/file",
				"timeout": "3s",
			},
			expected: &ExecUpdater{
				Command: "ls",
				Args:    []string{"-lh"},
				Stdout:  "/path/to/some/file",
				Stderr:  "/path/to/some/other/file",
				Timeout: 3 * time.Second,
			},
		},
		{
			name:             "nil params",
			expectedErrorMsg: "missing cmd parameter",
		},
		{
			name: "missing mandatory cmd param",
			params: map[string]string{
				"args": "-lh",
			},
			expectedErrorMsg: "missing cmd parameter",
		},
		{
			name: "invalid timeout",
			params: map[string]string{
				"cmd":     "ls",
				"timeout": "15 seconds",
			},
			expectedErrorMsg: "failed to parse duration for cmd timeout '15 seconds': time: unknown unit  seconds in duration 15 seconds",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := NewUpdater(test.params)
			if len(test.expectedErrorMsg) > 0 {
				require.EqualError(t, err, test.expectedErrorMsg)
				assert.Nil(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		files                 map[string]string
		updater               *ExecUpdater
		expected              bool
		expectedErrorMessages []string
		extraCheck            func() bool
	}{
		{
			name: "delete a file",
			files: map[string]string{
				"file-to-delete.txt": "whatever content",
			},
			updater: &ExecUpdater{
				Command: "rm",
				Args:    []string{"file-to-delete.txt"},
				Timeout: 1 * time.Second,
			},
			expected: true,
			extraCheck: func() bool {
				// ensure the file has been deleted
				_, err := os.Stat(filepath.Join("testdata", "file-to-delete.txt"))
				return err != nil
			},
		},
		{
			name: "fail deleting a non-existing file",
			updater: &ExecUpdater{
				Command: "rm",
				Args:    []string{"does-not-exists.txt"},
				Timeout: 1 * time.Second,
			},
			expected: false,
			expectedErrorMessages: []string{
				"failed to run cmd 'rm' with args [does-not-exists.txt] - got stdout [] and stderr [rm: does-not-exists.txt: No such file or directory]: exit status 1",
				"failed to run cmd 'rm' with args [does-not-exists.txt] - got stdout [] and stderr [rm: cannot remove 'does-not-exists.txt': No such file or directory]: exit status 1",
			},
		},
		{
			name: "run a cmd and write its stdout to a file",
			files: map[string]string{
				"file-to-print.txt": "some content",
			},
			updater: &ExecUpdater{
				Command: "cat",
				Args:    []string{"file-to-print.txt"},
				Stdout:  "file-to-print.stdout",
				Timeout: 1 * time.Second,
			},
			expected: true,
			extraCheck: func() bool {
				actualFileContent, _ := ioutil.ReadFile(filepath.Join("testdata", "file-to-print.stdout"))
				return bytes.Equal(actualFileContent, []byte("some content"))
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			{
				for filename, content := range test.files {
					err := os.MkdirAll(filepath.Dir(filepath.Join("testdata", filename)), 0755)
					require.NoErrorf(t, err, "can't create testdata directories for %s", filename)
					err = ioutil.WriteFile(filepath.Join("testdata", filename), []byte(content), 0644)
					require.NoErrorf(t, err, "can't write testdata file %s", filename)
				}
			}

			actual, err := test.updater.Update(context.Background(), "testdata")
			if len(test.expectedErrorMessages) > 0 {
				require.Error(t, err)
				assert.Contains(t, test.expectedErrorMessages, err.Error())
				assert.False(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
				if test.extraCheck != nil {
					assert.True(t, test.extraCheck())
				}
			}
		})
	}
}
