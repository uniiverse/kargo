package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_stringReplacer_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "inPath not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): inPath is required",
			},
		},
		{
			name: "inPath is empty string",
			config: promotion.Config{
				"inPath": "",
			},
			expectedProblems: []string{
				"inPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "outPath not specified",
			config: promotion.Config{
				"inPath": "/input.yaml",
			},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "outPath is empty string",
			config: promotion.Config{
				"inPath":  "/input.yaml",
				"outPath": "",
			},
			expectedProblems: []string{
				"outPath: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"inPath":  "/input.yaml",
				"outPath": "/output.yaml",
			},
			expectedProblems: nil,
		},
	}

	r := newStringReplacer(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*stringReplacer)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_stringReplacer_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T, string)
		config     builtin.StringReplacerConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful replacement",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: replacements
  annotations:
    universe.engineer/string-replacer: "true"
data:
  CLUSTER_NAME: prod-us-east1
  ENVIRONMENT: production
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  config: |
    cluster = "REPLACE_ME[CLUSTER_NAME]"
    env = "REPLACE_ME[ENVIRONMENT]"
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				b, readErr := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, readErr)
				output := string(b)
				assert.Contains(t, output, `cluster = "prod-us-east1"`)
				assert.Contains(t, output, `env = "production"`)
				assert.NotContains(t, output, "REPLACE_ME")
			},
		},
		{
			name: "replacer ConfigMap preserved in output",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: replacements
  annotations:
    universe.engineer/string-replacer: "true"
data:
  CLUSTER_NAME: dev-cluster
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  host: REPLACE_ME[CLUSTER_NAME]
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				b, readErr := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, readErr)
				output := string(b)
				assert.Contains(t, output, "universe.engineer/string-replacer")
				assert.Contains(t, output, "name: replacements")
			},
		},
		{
			name: "multiple occurrences of same placeholder",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: replacements
  annotations:
    universe.engineer/string-replacer: "true"
data:
  CLUSTER_NAME: my-cluster
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: alloy-config
data:
  config.alloy: |
    external_labels = {
      "cluster" = "REPLACE_ME[CLUSTER_NAME]",
      "k8s_cluster_name" = "REPLACE_ME[CLUSTER_NAME]",
    }
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				b, readErr := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, readErr)
				output := string(b)
				assert.NotContains(t, output, "REPLACE_ME")
				assert.Equal(t, 2, strings.Count(output, `"my-cluster"`))
			},
		},
		{
			name: "no annotated ConfigMap",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "no ConfigMap with annotation")
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "multiple annotated ConfigMaps",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: replacer-1
  annotations:
    universe.engineer/string-replacer: "true"
data:
  FOO: bar
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: replacer-2
  annotations:
    universe.engineer/string-replacer: "true"
data:
  BAZ: qux
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "multiple ConfigMaps with annotation")
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name:       "input file not found",
			setupFiles: func(*testing.T, string) {},
			config: builtin.StringReplacerConfig{
				InPath:  "nonexistent.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error reading input file")
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "output to subdirectory",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: replacements
  annotations:
    universe.engineer/string-replacer: "true"
data:
  NAME: test
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: REPLACE_ME[NAME]
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "sub/dir/output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				b, readErr := os.ReadFile(
					filepath.Join(dir, "sub", "dir", "output.yaml"),
				)
				require.NoError(t, readErr)
				assert.Contains(t, string(b), "name: test")
			},
		},
		{
			name: "unreplaced placeholders cause failure",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: replacements
  annotations:
    universe.engineer/string-replacer: "true"
data:
  CLUSTER_NAME: my-cluster
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  cluster: REPLACE_ME[CLUSTER_NAME]
  region: REPLACE_ME[REGION]
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "unreplaced placeholders remain")
				require.ErrorContains(t, err, "REPLACE_ME[REGION]")
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "ConfigMap without annotation is not used",
			setupFiles: func(t *testing.T, dir string) {
				content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: not-a-replacer
data:
  CLUSTER_NAME: should-not-be-used
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: replacements
  annotations:
    universe.engineer/string-replacer: "true"
data:
  CLUSTER_NAME: correct-value
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: target
data:
  host: REPLACE_ME[CLUSTER_NAME]
`
				require.NoError(t, os.WriteFile(
					filepath.Join(dir, "input.yaml"),
					[]byte(content), 0o600,
				))
			},
			config: builtin.StringReplacerConfig{
				InPath:  "input.yaml",
				OutPath: "output.yaml",
			},
			assertions: func(t *testing.T, dir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				b, readErr := os.ReadFile(filepath.Join(dir, "output.yaml"))
				require.NoError(t, readErr)
				output := string(b)
				assert.Contains(t, output, "host: correct-value")
				assert.NotContains(t, output, "REPLACE_ME")
			},
		},
	}

	runner := &stringReplacer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setupFiles(t, tempDir)
			stepCtx := &promotion.StepContext{WorkDir: tempDir}
			result, err := runner.run(stepCtx, tt.config)
			tt.assertions(t, tempDir, result, err)
		})
	}
}

func Test_splitYAMLDocuments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "single document",
			input:    "apiVersion: v1\nkind: ConfigMap\n",
			expected: 1,
		},
		{
			name:     "two documents",
			input:    "apiVersion: v1\nkind: ConfigMap\n---\napiVersion: v1\nkind: Secret\n",
			expected: 2,
		},
		{
			name:     "leading separator",
			input:    "---\napiVersion: v1\nkind: ConfigMap\n",
			expected: 1,
		},
		{
			name:     "empty input",
			input:    "",
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := splitYAMLDocuments([]byte(tt.input))
			assert.Len(t, docs, tt.expected)
		})
	}
}

func Test_extractReplacements(t *testing.T) {
	tests := []struct {
		name         string
		docs         [][]byte
		expectErr    string
		expectedKeys []string
	}{
		{
			name: "annotated ConfigMap",
			docs: [][]byte{
				[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n" +
					"  annotations:\n    universe.engineer/string-replacer: \"true\"\n" +
					"data:\n  FOO: bar\n"),
				[]byte("kind: ConfigMap\n"),
			},
			expectedKeys: []string{"FOO"},
		},
		{
			name: "no annotated ConfigMap",
			docs: [][]byte{
				[]byte("kind: ConfigMap\ndata:\n  FOO: bar\n"),
			},
			expectErr: "no ConfigMap with annotation",
		},
		{
			name: "multiple annotated ConfigMaps",
			docs: [][]byte{
				[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n" +
					"  annotations:\n    universe.engineer/string-replacer: \"true\"\n" +
					"data:\n  A: b\n"),
				[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n" +
					"  annotations:\n    universe.engineer/string-replacer: \"true\"\n" +
					"data:\n  C: d\n"),
			},
			expectErr: "multiple ConfigMaps with annotation",
		},
		{
			name: "ConfigMap without annotation is ignored",
			docs: [][]byte{
				[]byte("kind: ConfigMap\ndata:\n  X: y\n"),
				[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n" +
					"  annotations:\n    universe.engineer/string-replacer: \"true\"\n" +
					"data:\n  KEY: val\n"),
			},
			expectedKeys: []string{"KEY"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacements, err := extractReplacements(tt.docs)
			if tt.expectErr != "" {
				require.ErrorContains(t, err, tt.expectErr)
				return
			}
			require.NoError(t, err)
			for _, key := range tt.expectedKeys {
				assert.Contains(t, replacements, key)
			}
		})
	}
}

func Test_applyReplacements(t *testing.T) {
	docs := [][]byte{
		[]byte(`cluster = "REPLACE_ME[CLUSTER]"`),
		[]byte(`env = "REPLACE_ME[ENV]", cluster = "REPLACE_ME[CLUSTER]"`),
	}
	replacements := map[string]string{
		"CLUSTER": "prod",
		"ENV":     "production",
	}

	result := applyReplacements(docs, replacements)

	require.Len(t, result, 2)
	assert.Equal(t, `cluster = "prod"`, string(result[0]))
	assert.Equal(t, `env = "production", cluster = "prod"`, string(result[1]))
}

func Test_findUnreplacedPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		docs     [][]byte
		expected []string
	}{
		{
			name: "no placeholders",
			docs: [][]byte{
				[]byte(`cluster = "prod"`),
			},
			expected: nil,
		},
		{
			name: "one placeholder",
			docs: [][]byte{
				[]byte(`cluster = "REPLACE_ME[CLUSTER]"`),
			},
			expected: []string{"REPLACE_ME[CLUSTER]"},
		},
		{
			name: "deduplicated across documents",
			docs: [][]byte{
				[]byte(`a = "REPLACE_ME[X]"`),
				[]byte(`b = "REPLACE_ME[X]", c = "REPLACE_ME[Y]"`),
			},
			expected: []string{"REPLACE_ME[X]", "REPLACE_ME[Y]"},
		},
		{
			name:     "empty docs",
			docs:     [][]byte{},
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findUnreplacedPlaceholders(tt.docs)
			assert.Equal(t, tt.expected, result)
		})
	}
}
