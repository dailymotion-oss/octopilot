package yq

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	gologging "gopkg.in/op/go-logging.v1"
)

func init() {
	gologging.SetLevel(gologging.CRITICAL, "yq-lib")
}

type YQUpdater struct {
	FilePath     string
	Expression   string
	Output       string
	OutputToJSON bool
	Indent       int
	Trim         bool
	UnwrapScalar bool
}

func NewUpdater(params map[string]string) (*YQUpdater, error) {
	updater := &YQUpdater{}

	updater.FilePath = params["file"]
	if len(updater.FilePath) == 0 {
		return nil, errors.New("missing file parameter")
	}

	updater.Expression = params["expression"]
	if len(updater.Expression) == 0 {
		return nil, errors.New("missing expression parameter")
	}

	updater.Indent, _ = strconv.Atoi(params["indent"])
	if updater.Indent <= 0 {
		updater.Indent = 2
	}

	var err error
	if updater.UnwrapScalar, err = strconv.ParseBool(params["unwrapscalar"]); err != nil {
		// let's unwrap scalar by default, same as yq
		updater.UnwrapScalar = true
	}
	updater.Trim, _ = strconv.ParseBool(params["trim"])
	updater.OutputToJSON, _ = strconv.ParseBool(params["json"])
	updater.Output = params["output"]

	return updater, nil
}

func (u *YQUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	expressionNode, err := yqlib.NewExpressionParser().ParseExpression(u.Expression)
	if err != nil {
		return false, fmt.Errorf("failed to parse yq expression %s: %w", u.Expression, err)
	}

	var output io.Writer
	switch u.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "":
		// will be handled later, because we'll be writing in-place to the source file
	default:
		outputFilePath := u.Output
		if !filepath.IsAbs(outputFilePath) {
			outputFilePath = filepath.Join(repoPath, outputFilePath)
		}
		f, err := os.Create(outputFilePath)
		if err != nil {
			return false, fmt.Errorf("failed to create output file %s: %w", outputFilePath, err)
		}
		defer f.Close()
		output = f
	}

	filePaths, err := filepath.Glob(filepath.Join(repoPath, u.FilePath))
	if err != nil {
		return false, fmt.Errorf("failed to expand glob pattern %s: %w", u.FilePath, err)
	}

	var (
		streamEvaluator = yqlib.NewStreamEvaluator()
		updated         = false
	)
	for _, filePath := range filePaths {
		relFilePath, err := filepath.Rel(repoPath, filePath)
		if err != nil {
			relFilePath = filePath
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to access file %s: %w", relFilePath, err)
		}

		fileData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to read file %s: %w", relFilePath, err)
		}

		buffer := new(bytes.Buffer)
		printer := yqlib.NewPrinter(buffer, u.OutputToJSON, u.UnwrapScalar, false, u.Indent, true)
		err = streamEvaluator.Evaluate(relFilePath, bytes.NewReader(fileData), expressionNode, printer)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate expression `%s` for file %s: %w", u.Expression, filePath, err)
		}

		if u.Trim {
			buffer = bytes.NewBuffer(bytes.TrimSpace(buffer.Bytes()))
		}

		if reflect.DeepEqual(fileData, buffer.Bytes()) {
			continue
		}

		if output != nil {
			_, err = buffer.WriteTo(output)
		} else {
			// we need to write in-place in the same (source) file
			err = ioutil.WriteFile(filePath, buffer.Bytes(), fileInfo.Mode())
		}
		if err != nil {
			return false, fmt.Errorf("failed to write yq result to the output: %w", err)
		}

		updated = true
	}

	return updated, nil
}

func (u *YQUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Update %s", u.FilePath)
	body = fmt.Sprintf("Updating file(s) `%s` using yq expression `%s`", u.FilePath, u.Expression)
	return title, body
}

func (u *YQUpdater) String() string {
	return fmt.Sprintf("YQ[file=%s,expression=%s,output=%s,indent=%v]", u.FilePath, u.Expression, u.Output, u.Indent)
}
