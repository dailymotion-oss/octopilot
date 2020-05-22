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

	"github.com/dailymotion/octopilot/update/value"

	"github.com/mikefarah/yq/v3/pkg/yqlib"
	gologging "gopkg.in/op/go-logging.v1"
	"gopkg.in/yaml.v3"
)

func init() {
	gologging.SetLevel(gologging.CRITICAL, "yq")
}

type YamlUpdater struct {
	FilePath   string
	Path       string
	AutoCreate bool
	Style      string
	Trim       bool
	Valuer     value.Valuer
}

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

	updater.AutoCreate, _ = strconv.ParseBool(params["create"])
	updater.Trim, _ = strconv.ParseBool(params["trim"])
	updater.Style = params["style"]

	updater.Valuer = valuer

	return updater, nil
}

func (u *YamlUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	var (
		yq          = yqlib.NewYqLib()
		valueParser = yqlib.NewValueParser()
	)

	value, err := u.Valuer.Value(ctx, repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to get value: %w", err)
	}

	updateCmd := yqlib.UpdateCommand{
		Command:   "update",
		Overwrite: true,
		Path:      u.Path,
		Value:     valueParser.Parse(value, "", u.Style),
	}

	filePaths, err := filepath.Glob(filepath.Join(repoPath, u.FilePath))
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

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to read file %s: %w", relFilePath, err)
		}

		var rootNode yaml.Node
		err = yaml.Unmarshal(data, &rootNode)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal YAML file %s: %w", filePath, err)
		}

		err = yq.Update(&rootNode, updateCmd, u.AutoCreate)
		if err != nil {
			return false, fmt.Errorf("failed to update YAML file %s: %w", filePath, err)
		}

		updatedData, err := yaml.Marshal(&rootNode)
		if err != nil {
			return false, fmt.Errorf("failed to marshal updated YAML content for %s: %w", filePath, err)
		}

		if u.Trim {
			updatedData = bytes.TrimSpace(updatedData)
		}

		if reflect.DeepEqual(data, updatedData) {
			continue
		}

		err = ioutil.WriteFile(filePath, updatedData, fileInfo.Mode())
		if err != nil {
			return false, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}

		updated = true
	}

	return updated, nil
}

func (u *YamlUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Update %s", u.FilePath)
	body = fmt.Sprintf("Updating path `%s` in file(s) `%s`", u.Path, u.FilePath)
	return title, body
}

func (u *YamlUpdater) String() string {
	return fmt.Sprintf("YAML[path=%s,file=%s,create=%v]", u.Path, u.FilePath, u.AutoCreate)
}
