package builtin

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindStringReplacer = "string-replacer"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindStringReplacer,
			Value: newStringReplacer,
		},
	)
}

const stringReplacerAnnotation = "universe.engineer/string-replacer"

// stringReplacer is an implementation of the promotion.StepRunner interface
// that performs string replacements across YAML manifests. It finds a
// ConfigMap annotated with universe.engineer/string-replacer: "true" and
// uses its data entries as REPLACE_ME[KEY] → value substitution pairs.
type stringReplacer struct {
	schemaLoader gojsonschema.JSONLoader
}

// newStringReplacer returns an implementation of the promotion.StepRunner
// interface that performs string replacements across YAML manifests.
func newStringReplacer(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &stringReplacer{
		schemaLoader: getConfigSchemaLoader(stepKindStringReplacer),
	}
}

// Run implements the promotion.StepRunner interface.
func (s *stringReplacer) Run(
	_ context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := s.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return s.run(stepCtx, cfg)
}

// convert validates stringReplacer configuration against a JSON schema and
// converts it into a builtin.StringReplacerConfig struct.
func (s *stringReplacer) convert(
	cfg promotion.Config,
) (builtin.StringReplacerConfig, error) {
	return validateAndConvert[builtin.StringReplacerConfig](
		s.schemaLoader, cfg, stepKindStringReplacer,
	)
}

func (s *stringReplacer) run(
	stepCtx *promotion.StepContext,
	cfg builtin.StringReplacerConfig,
) (promotion.StepResult, error) {
	inPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.InPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error resolving input path %q: %w", cfg.InPath, err)
	}

	raw, err := os.ReadFile(inPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf(
				"error reading input file %q: %w",
				cfg.InPath, fs.SanitizePathError(err, stepCtx.WorkDir),
			)
	}

	docs := splitYAMLDocuments(raw)

	replacements, err := extractReplacements(docs)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	var result [][]byte
	if replacements != nil {
		result = applyReplacements(docs, replacements)
	} else {
		result = docs
	}

	if remaining := findUnreplacedPlaceholders(result); len(remaining) > 0 {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf(
				"unreplaced placeholders remain in output: %s",
				strings.Join(remaining, ", "),
			)
	}

	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error resolving output path %q: %w", cfg.OutPath, err)
	}

	if err = os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf(
				"error creating output directory for %q: %w",
				cfg.OutPath, fs.SanitizePathError(err, stepCtx.WorkDir),
			)
	}

	output := bytes.Join(result, []byte("\n---\n"))
	if err = os.WriteFile(outPath, output, 0o600); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf(
				"error writing output file %q: %w",
				cfg.OutPath, fs.SanitizePathError(err, stepCtx.WorkDir),
			)
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// configMapResource is a minimal representation of a Kubernetes ConfigMap
// used to detect the string-replacer annotation and extract data entries.
type configMapResource struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Data map[string]string `json:"data"`
}

// splitYAMLDocuments splits raw YAML bytes on document separators ("---").
func splitYAMLDocuments(data []byte) [][]byte {
	var docs [][]byte
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var current bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if current.Len() > 0 {
				trimmed := bytes.TrimSpace(current.Bytes())
				cp := make([]byte, len(trimmed))
				copy(cp, trimmed)
				docs = append(docs, cp)
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		trimmed := bytes.TrimSpace(current.Bytes())
		cp := make([]byte, len(trimmed))
		copy(cp, trimmed)
		docs = append(docs, cp)
	}
	return docs
}

// extractReplacements finds a ConfigMap annotated with
// universe.engineer/string-replacer: "true" and returns its data entries
// as the replacement map.
func extractReplacements(docs [][]byte) (map[string]string, error) {
	var (
		replacements map[string]string
		found        bool
	)
	for _, doc := range docs {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		var cm configMapResource
		if err := yaml.Unmarshal(doc, &cm); err != nil ||
			cm.Kind != "ConfigMap" {
			continue
		}
		if cm.Metadata.Annotations[stringReplacerAnnotation] != "true" {
			continue
		}
		if found {
			return nil, fmt.Errorf(
				"multiple ConfigMaps with annotation %q found",
				stringReplacerAnnotation,
			)
		}
		replacements = cm.Data
		found = true
	}
	if !found {
		return nil, nil
	}
	return replacements, nil
}

// applyReplacements performs text substitution on each document, replacing
// every occurrence of REPLACE_ME[KEY] with the corresponding value.
func applyReplacements(
	docs [][]byte,
	replacements map[string]string,
) [][]byte {
	result := make([][]byte, len(docs))
	for i, doc := range docs {
		s := string(doc)
		for key, value := range replacements {
			s = strings.ReplaceAll(s, "REPLACE_ME["+key+"]", value)
		}
		result[i] = []byte(s)
	}
	return result
}

const placeholderPrefix = "REPLACE_ME["

// findUnreplacedPlaceholders scans the documents for any remaining
// REPLACE_ME[...] placeholders and returns a deduplicated list.
func findUnreplacedPlaceholders(docs [][]byte) []string {
	seen := make(map[string]struct{})
	var remaining []string
	for _, doc := range docs {
		s := string(doc)
		for {
			idx := strings.Index(s, placeholderPrefix)
			if idx < 0 {
				break
			}
			s = s[idx:]
			end := strings.Index(s, "]")
			if end < 0 {
				break
			}
			placeholder := s[:end+1]
			if _, ok := seen[placeholder]; !ok {
				seen[placeholder] = struct{}{}
				remaining = append(remaining, placeholder)
			}
			s = s[end+1:]
		}
	}
	return remaining
}
