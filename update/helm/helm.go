// Package helm provides an updater that updates Helm charts dependencies.
package helm

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dailymotion-oss/octopilot/update/value"
	"gopkg.in/yaml.v3"
)

// HelmUpdater is an updater that updates Helm charts dependencies.
type HelmUpdater struct {
	Dependency string
	Indent     int
	Valuer     value.Valuer
}

// NewUpdater builds a new Helm updater from the given parameters and valuer
func NewUpdater(params map[string]string, valuer value.Valuer) (*HelmUpdater, error) {
	updater := &HelmUpdater{}

	updater.Dependency = params["dependency"]
	if len(updater.Dependency) == 0 {
		return nil, errors.New("missing dependency parameter")
	}

	updater.Indent, _ = strconv.Atoi(params["indent"])
	if updater.Indent <= 0 {
		updater.Indent = 2
	}

	updater.Valuer = valuer

	return updater, nil
}

// Update updates the repository cloned at the given path, and returns true if changes have been made
func (u *HelmUpdater) Update(ctx context.Context, repoPath string) (bool, error) {
	charts, err := extractHelmChartsDirectories(repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to find Helm Charts located in %s: %w", repoPath, err)
	}

	value, err := u.Valuer.Value(ctx, repoPath)
	if err != nil {
		return false, fmt.Errorf("failed to get value: %w", err)
	}

	var updated bool
	for _, chartDir := range charts {
		dependenciesFilesPaths := []string{
			filepath.Join(chartDir, "requirements.yaml"), // helm2
			filepath.Join(chartDir, "Chart.yaml"),        // helm3
		}
		for _, dependenciesFilePath := range dependenciesFilesPaths {
			if _, err = os.Stat(dependenciesFilePath); err == nil {
				chartUpdated, err := u.updateChartDependenciesFile(dependenciesFilePath, value)
				if err != nil {
					return false, fmt.Errorf("failed to update Helm Chart located in %s: %w", chartDir, err)
				}
				if chartUpdated {
					updated = true
				}
			}
		}
	}

	return updated, nil
}

func (u *HelmUpdater) updateChartDependenciesFile(filePath string, version string) (bool, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var rootNode yaml.Node
	err = yaml.Unmarshal(data, &rootNode)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal YAML file %s: %w", filePath, err)
	}

	updated := u.updateChartDependenciesNode(&rootNode, version)
	if !updated {
		return false, nil
	}

	f, err := os.Create(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	enc.SetIndent(u.Indent)
	err = enc.Encode(&rootNode)
	if err != nil {
		return false, fmt.Errorf("failed to encode updated YAML content for %s: %w", filePath, err)
	}
	err = enc.Close()
	if err != nil {
		return false, fmt.Errorf("failed to close the YAML encoder for %s: %w", filePath, err)
	}

	return updated, nil
}

func (u *HelmUpdater) updateChartDependenciesNode(node *yaml.Node, version string) bool {
	if node == nil {
		return false
	}

	var updated bool
	for i, child := range node.Content {
		if child == nil {
			continue
		}

		if child.Value == "dependencies" {
			if i+1 < len(node.Content) {
				dependenciesValueNode := node.Content[i+1]
				for _, dependency := range dependenciesValueNode.Content {
					if u.updateChartDependencyNode(dependency, version) {
						updated = true
					}
				}
			}
		}

		if u.updateChartDependenciesNode(child, version) {
			updated = true
		}
	}

	return updated
}

func (u *HelmUpdater) updateChartDependencyNode(node *yaml.Node, version string) bool {
	if node == nil {
		return false
	}
	if node.Kind != yaml.MappingNode {
		return false
	}

	var dependencyNameMatch bool
	for i := 0; i < len(node.Content); i += 2 {
		var (
			keyNode   = node.Content[i]
			valueNode = node.Content[i+1]
		)
		if keyNode.Value == "name" && valueNode.Value == u.Dependency {
			dependencyNameMatch = true
			break
		}
	}
	if !dependencyNameMatch {
		return false
	}

	for i := 0; i < len(node.Content); i += 2 {
		var (
			keyNode   = node.Content[i]
			valueNode = node.Content[i+1]
		)
		if keyNode.Value == "version" && valueNode.Value != version {
			valueNode.SetString(version)
			return true
		}
	}

	return false
}

// Message returns the default title and body that should be used in the commits / pull requests
func (u *HelmUpdater) Message() (title, body string) {
	title = fmt.Sprintf("Update %s", u.Dependency)
	body = fmt.Sprintf("Updating dependency `%s`", u.Dependency)
	return title, body
}

// String returns a string representation of the updater
func (u *HelmUpdater) String() string {
	return fmt.Sprintf("Helm[dependency=%s]", u.Dependency)
}

func extractHelmChartsDirectories(baseDir string) ([]string, error) {
	var chartDirectories []string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			switch filepath.Base(path) {
			case ".git":
				return filepath.SkipDir
			default:
				return nil
			}
		}

		switch filepath.Base(path) {
		case "Chart.yaml":
			chartDirectories = append(chartDirectories, filepath.Dir(path))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chartDirectories, nil
}
