package fzfg

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const combinedConfigName = "config.yaml"

// preprocessIncludes reads the main config file, expands any !include directives
// by nesting module content under the parent key, writes the combined result to
// config.yaml in the same directory, and returns the path to the combined file.
func preprocessIncludes(configPath string) (string, error) {
	baseDir := filepath.Dir(configPath)
	outputPath := filepath.Join(baseDir, combinedConfigName)

	file, err := os.Open(configPath)
	if err != nil {
		return "", fmt.Errorf("unable to read config file: %w", err)
	}
	defer file.Close()

	var result strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		key, pattern, ok := parseIncludeDirective(line)
		if !ok {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		expanded, err := expandInclude(baseDir, key, pattern)
		if err != nil {
			return "", fmt.Errorf("expanding !include for key %q: %w", key, err)
		}
		result.WriteString(expanded)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading config file: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(result.String()), 0644); err != nil {
		return "", fmt.Errorf("writing combined config: %w", err)
	}

	return outputPath, nil
}

// parseIncludeDirective checks if a line matches the pattern:
//
//	key: !include <glob_pattern>
//
// Returns the key name, glob pattern, and whether it matched.
func parseIncludeDirective(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "!include") {
		return "", "", false
	}

	colonIdx := strings.Index(trimmed, ":")
	if colonIdx < 1 {
		return "", "", false
	}

	key := strings.TrimSpace(trimmed[:colonIdx])
	rest := strings.TrimSpace(trimmed[colonIdx+1:])

	if !strings.HasPrefix(rest, "!include") {
		return "", "", false
	}

	pattern := strings.TrimSpace(strings.TrimPrefix(rest, "!include"))
	if pattern == "" {
		return "", "", false
	}

	return key, pattern, true
}

// expandInclude resolves a glob pattern relative to baseDir, reads each matching
// file, and returns the combined content nested under the given key with proper
// YAML indentation (2 spaces).
func expandInclude(baseDir, key, pattern string) (string, error) {
	fullPattern := filepath.Join(baseDir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	if len(matches) == 0 {
		return fmt.Sprintf("# %s: !include %s (no matching files)\n", key, pattern), nil
	}

	var result strings.Builder

	for _, match := range matches {
		content, err := os.ReadFile(match)
		if err != nil {
			return "", fmt.Errorf("reading module file %q: %w", match, err)
		}

		// Write the key header for this module
		result.WriteString(fmt.Sprintf("%s: # !include %s\n", key, filepath.Base(match)))

		// Indent each line of module content by 2 spaces under the key
		lines := strings.Split(string(content), "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) == "" {
				result.WriteString("\n")
			} else {
				result.WriteString("  ")
				result.WriteString(l)
				result.WriteString("\n")
			}
		}
	}

	return result.String(), nil
}
