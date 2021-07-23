// Package yaml provides an updater that uses the yq lib to update YAML files.
package yaml

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/dailymotion-oss/octopilot/internal/yaml"
	"github.com/dailymotion-oss/octopilot/update/value"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	gologging "gopkg.in/op/go-logging.v1"
)

func init() {
	gologging.SetLevel(gologging.CRITICAL, "yq-lib")
}

// YamlUpdater is an updater that uses the yq lib to update YAML files.
type YamlUpdater struct {
	FilePath   string
	Path       string
	AutoCreate bool
	Style      string
	Trim       bool
	Indent     int
	Valuer     value.Valuer
}

// NewUpdater builds a new YAML updater from the given parameters and valuer
func NewUpdater(params map[string]string, valuer value.Valuer) (*YamlUpdater, error) {
	updater := &YamlUpdater{}

	updater.FilePath = params["file"]
	if len(updater.FilePath) == 0 {
		return nil, errors.New("missing file parameter")
	}

	updater.Path = params["path"]
	if len(updater.Path) == 0 {
		return nil, errors.New("missing path parameter")
	}

	updater.Indent, _ = strconv.Atoi(params["indent"])
	if updater.Indent <= 0 {
		updater.Indent = 2
	}

	updater.AutoCreate, _ = strconv.ParseBool(params["create"])
	updater.Trim, _ = strconv.ParseBool(params["trim"])
	updater.Style = params["style"]

	updater.Valuer = valuer

	return updater, nil
}

// Update updates the repository cloned at the given path, and returns true if changes have been made
func (u *YamlUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	value, err := u.Valuer.Value(ctx, repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to get value: %w", err)
	}

	expression, expressionNode, err := u.yqExpression(value)
	if err != nil {
		return false, fmt.Errorf("failed to parse yq expression %s: %w", expression, err)
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

		reader, leadingContent, err := yaml.ExtractLeadingContentForYQ(bytes.NewReader(fileData))
		if err != nil {
			return false, fmt.Errorf("failed to extract leading content from file %s: %w", relFilePath, err)
		}

		buffer := new(bytes.Buffer)
		printer := yqlib.NewPrinter(buffer, false, false, false, u.Indent, true)
		_, err = streamEvaluator.Evaluate(relFilePath, reader, expressionNode, printer, leadingContent)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate expression `%s` for file %s: %w", expression, filePath, err)
		}

		if u.Trim {
			buffer = bytes.NewBuffer(bytes.TrimSpace(buffer.Bytes()))
		}

		if reflect.DeepEqual(fileData, buffer.Bytes()) {
			continue
		}

		err = ioutil.WriteFile(filePath, buffer.Bytes(), fileInfo.Mode())
		if err != nil {
			return false, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}

		updated = true
	}

	return updated, nil
}

// Message returns the default title and body that should be used in the commits / pull requests
func (u *YamlUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Update %s", u.FilePath)
	body = fmt.Sprintf("Updating path `%s` in file(s) `%s`", u.Path, u.FilePath)
	return title, body
}

// String returns a string representation of the updater
func (u *YamlUpdater) String() string {
	return fmt.Sprintf("YAML[path=%s,file=%s,style=%s,create=%v,trim=%v,indent=%v]", u.Path, u.FilePath, u.Style, u.AutoCreate, u.Trim, u.Indent)
}

func (u *YamlUpdater) yqExpression(value string) (string, *yqlib.ExpressionNode, error) {
	var (
		parser        = yqlib.NewExpressionParser()
		rawExpression string
	)

	if _, err := parser.ParseExpression(u.Path); err == nil {
		// we have a valid yq v4 expression
		rawExpression = u.Path
	} else {
		//most likely an old v3 path format, let's convert it to a valid v4 path
		rawExpression = convertYqExpressionToV4(u.Path)
	}

	// add the assignment operator to set the new value
	expression := fmt.Sprintf(`(%s) as $x | $x = %q`, rawExpression, value)

	if u.AutoCreate {
		// ensure the new path is created first (the `... as $x | $x = ...` doesn't create it)
		expression = fmt.Sprintf(`%s = %q | %s`, rawExpression, value, expression)
	}

	if u.Style != "" {
		// set the style if needed
		expression = fmt.Sprintf(`%s | $x style=%q`, expression, u.Style)
	}

	expressionNode, err := parser.ParseExpression(expression)
	return expression, expressionNode, err
}

// convertYqExpressionToV4 converts from the old yq v3 format to the new yq v4 format
func convertYqExpressionToV4(v3Format string) string {
	if !strings.ContainsAny(v3Format, "()=") {
		// this is a simple path expression to traverse a hierarchy
		expression := v3Format
		if !strings.HasPrefix(expression, ".") {
			// let's ensure it starts with a dot
			expression = "." + expression
		}
		return expression
	}

	if strings.Contains(v3Format, "(") && strings.Contains(v3Format, ")") && strings.Contains(v3Format, "==") {
		// this is a path selection in an array, such as 'array.(name==foo).field'
		// let's rewrite it as '.array[] | select(.name == "foo") | .field'
		return fmt.Sprintf(`.%s[] | select(.%s == %q) | %s`,
			strings.TrimSuffix(strings.SplitN(v3Format, "(", 2)[0], "."),
			strings.SplitN(strings.SplitN(v3Format, "(", 2)[1], "==", 2)[0],
			strings.SplitN(strings.SplitN(v3Format, "==", 2)[1], ")", 2)[0],
			strings.SplitN(v3Format, ")", 2)[1],
		)
	}

	return ""
}
