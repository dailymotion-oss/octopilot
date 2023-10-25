package glob

import (
	"path/filepath"
	"testing"
)

func generateTestCases(testdataDir string) map[string]map[string]bool {
	return map[string]map[string]bool{
		"**/data.txt": {
			filepath.Join(testdataDir, "data.txt"):                    true,
			filepath.Join(testdataDir, "subdirectory1/data.txt"):      true,
			filepath.Join(testdataDir, "subdirectory2/data.txt"):      true,
			filepath.Join(testdataDir, "subdirectory2/test/data.txt"): true,
		},
		"**/*/data.txt": {
			filepath.Join(testdataDir, "subdirectory1/data.txt"):      true,
			filepath.Join(testdataDir, "subdirectory2/data.txt"):      true,
			filepath.Join(testdataDir, "subdirectory2/test/data.txt"): true,
		},
		"*/**/data.txt": {
			filepath.Join(testdataDir, "subdirectory1/data.txt"):      true,
			filepath.Join(testdataDir, "subdirectory2/data.txt"):      true,
			filepath.Join(testdataDir, "subdirectory2/test/data.txt"): true,
		},
		"*/data.txt": {
			filepath.Join(testdataDir, "subdirectory2/data.txt"): true,
			filepath.Join(testdataDir, "subdirectory1/data.txt"): true,
		},
		"subdirectory1/data.txt": {
			filepath.Join(testdataDir, "subdirectory1/data.txt"): true,
		},
		"*.txt": {
			filepath.Join(testdataDir, "data.txt"): true,
		},
		"non_existent_directory/data.txt": {},
		"non_existent_file.txt":           {},
	}
}

func TestExpandGlobPatternWithPatterns(t *testing.T) {
	testdataDir := "testdata"
	testCases := generateTestCases(testdataDir)

	for pattern, expectedPaths := range testCases {
		filePaths, err := ExpandGlobPattern(testdataDir, pattern)
		if err != nil {
			t.Errorf("Error expanding glob pattern %s: %v", pattern, err)
			continue
		}

		actualPathsMap := make(map[string]bool)
		for _, path := range filePaths {
			if !expectedPaths[path] {
				t.Errorf("Expected path %s not to exist for pattern %s, but it does", path, pattern)
			}
			actualPathsMap[path] = true
		}

		for expectedPath := range expectedPaths {
			if !actualPathsMap[expectedPath] {
				t.Errorf("Expected path %s to exist for pattern %s, but it does not", expectedPath, pattern)
			}
		}
	}
}
