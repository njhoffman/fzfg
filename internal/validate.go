package fzfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// OptionDef describes a single fzf option's definition from the definitions file.
type OptionDef struct {
	Description string      `yaml:"description"`
	Aliases     []string    `yaml:"aliases"`
	Required    bool        `yaml:"required"`
	Type        string      `yaml:"type"`
	Default     interface{} `yaml:"default"`
	Negate      []string    `yaml:"negate"`
	Value       *ValueDef   `yaml:"value"`
	Effects     []string    `yaml:"effects"`
	Conditions  []Condition `yaml:"conditions"`
}

// ValueDef describes the allowed values for an enum or complex option.
type ValueDef struct {
	Type        string                   `yaml:"type"`
	Description string                   `yaml:"description"`
	Values      map[string]*EnumValueDef `yaml:"values"`
}

// EnumValueDef describes a single enum value.
type EnumValueDef struct {
	Description string   `yaml:"description"`
	Effects     []string `yaml:"effects"`
}

// Condition describes when an option's behavior changes.
type Condition struct {
	If   string `yaml:"if"`
	Then string `yaml:"then"`
}

// OptionDefs is the top-level structure of the options definitions file.
type OptionDefs struct {
	Options map[string]map[string]OptionDef `yaml:"options"`
}

// ValidationError represents a single validation issue.
type ValidationError struct {
	Option  string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("option %q: %s", e.Option, e.Message)
}

// ValidationResult holds all errors and warnings from validation.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

func (r *ValidationResult) addError(option, msg string) {
	r.Errors = append(r.Errors, ValidationError{Option: option, Message: msg})
}

func (r *ValidationResult) addWarning(option, msg string) {
	r.Warnings = append(r.Warnings, ValidationError{Option: option, Message: msg})
}

// HasErrors returns true if there are validation errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// LoadOptionDefs loads the option definitions from the definitions YAML file.
func LoadOptionDefs(defsPath string) (*OptionDefs, error) {
	data, err := os.ReadFile(defsPath)
	if err != nil {
		return nil, fmt.Errorf("reading definitions file: %w", err)
	}
	var defs OptionDefs
	if err := yaml.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("parsing definitions file: %w", err)
	}
	return &defs, nil
}

// FlattenOptionDefs returns a flat map of option-name -> OptionDef,
// collapsing the category grouping.
func (d *OptionDefs) FlattenOptionDefs() map[string]OptionDef {
	flat := make(map[string]OptionDef)
	for _, category := range d.Options {
		for name, def := range category {
			flat[name] = def
		}
	}
	return flat
}

// ValidateOptions checks user-provided options against the definitions.
// It validates: unknown options, type mismatches, invalid enum values.
func ValidateOptions(userOpts Options, defs map[string]OptionDef) *ValidationResult {
	result := &ValidationResult{}

	for name, value := range userOpts {
		def, known := defs[name]
		if !known {
			result.addError(name, "unknown option")
			continue
		}

		validateType(result, name, value, &def)
	}

	return result
}

// validateType checks if the user's value matches the expected type.
func validateType(result *ValidationResult, name string, value interface{}, def *OptionDef) {
	switch def.Type {
	case "boolean":
		if _, ok := value.(bool); !ok {
			result.addError(name, fmt.Sprintf("expected boolean, got %T", value))
		}
	case "integer":
		switch v := value.(type) {
		case int, float64:
			// ok
		case bool:
			// multi uses bool false as default, allow it
		case string:
			if _, err := strconv.Atoi(v); err != nil {
				result.addError(name, fmt.Sprintf("expected integer, got string %q", v))
			}
		default:
			result.addError(name, fmt.Sprintf("expected integer, got %T", value))
		}
	case "enum":
		validateEnum(result, name, value, def)
	case "string":
		// strings are flexible — most YAML values coerce to string
	case "list":
		// lists can be string or sequence
		if def.Value != nil && def.Value.Type == "enum" {
			validateListEnum(result, name, value, def)
		}
	}
}

// validateEnum checks if the value is one of the allowed enum values.
func validateEnum(result *ValidationResult, name string, value interface{}, def *OptionDef) {
	if def.Value == nil || def.Value.Values == nil {
		return
	}

	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case bool:
		// some enums like "wrap" accept bool false as "disabled"
		return
	default:
		result.addError(name, fmt.Sprintf("expected enum string, got %T", value))
		return
	}

	if _, ok := def.Value.Values[strVal]; !ok {
		allowed := make([]string, 0, len(def.Value.Values))
		for k := range def.Value.Values {
			allowed = append(allowed, k)
		}
		result.addError(name, fmt.Sprintf(
			"invalid value %q, allowed: [%s]",
			strVal, strings.Join(allowed, ", "),
		))
	}
}

// validateListEnum checks list items against allowed enum values.
func validateListEnum(result *ValidationResult, name string, value interface{}, def *OptionDef) {
	if def.Value == nil || def.Value.Values == nil {
		return
	}

	var items []string
	switch v := value.(type) {
	case string:
		items = strings.Split(v, ",")
	case []interface{}:
		for _, item := range v {
			items = append(items, fmt.Sprint(item))
		}
	default:
		return
	}

	for _, item := range items {
		item = strings.TrimSpace(item)
		if _, ok := def.Value.Values[item]; !ok {
			allowed := make([]string, 0, len(def.Value.Values))
			for k := range def.Value.Values {
				allowed = append(allowed, k)
			}
			result.addError(name, fmt.Sprintf(
				"invalid list value %q, allowed: [%s]",
				item, strings.Join(allowed, ", "),
			))
		}
	}
}

// ValidateEffects checks for cross-option side-effect conflicts after all
// user values are resolved. For example, if the user sets scheme=path,
// the definition says tiebreak should be pathname,length — warn if the
// user also explicitly set a different tiebreak.
func ValidateEffects(userOpts Options, defs map[string]OptionDef) *ValidationResult {
	result := &ValidationResult{}

	for name, value := range userOpts {
		def, ok := defs[name]
		if !ok {
			continue
		}

		// Check top-level effects
		checkEffects(result, name, def.Effects, userOpts)

		// Check enum-value-specific effects
		if def.Value != nil && def.Value.Values != nil {
			strVal := fmt.Sprint(value)
			if enumDef, ok := def.Value.Values[strVal]; ok && enumDef != nil {
				checkEffects(result, fmt.Sprintf("%s=%s", name, strVal), enumDef.Effects, userOpts)
			}
		}
	}

	return result
}

// checkEffects parses effect strings like "tiebreak=pathname,length" and warns
// if the user has set a conflicting value for the affected option.
func checkEffects(result *ValidationResult, source string, effects []string, userOpts Options) {
	for _, effect := range effects {
		parts := strings.SplitN(effect, "=", 2)
		if len(parts) != 2 {
			continue
		}
		targetOpt := strings.TrimSpace(parts[0])
		expectedVal := strings.TrimSpace(parts[1])

		userVal, userSet := userOpts[targetOpt]
		if !userSet {
			continue
		}

		userStr := fmt.Sprint(userVal)
		if userStr != expectedVal {
			result.addWarning(targetOpt, fmt.Sprintf(
				"value %q conflicts with effect from %s (expected %q)",
				userStr, source, expectedVal,
			))
		}
	}
}

// ValidateConditions checks conditional relationships between options.
// For example, if ignore-case=true, smart-case should be disabled.
func ValidateConditions(userOpts Options, defs map[string]OptionDef) *ValidationResult {
	result := &ValidationResult{}

	for name, def := range defs {
		for _, cond := range def.Conditions {
			checkCondition(result, name, cond, userOpts)
		}
	}

	return result
}

// checkCondition evaluates a single condition rule.
func checkCondition(result *ValidationResult, optName string, cond Condition, userOpts Options) {
	// Parse the "if" clause: "optname=value" format
	ifParts := strings.SplitN(cond.If, "=", 2)
	if len(ifParts) != 2 {
		return
	}

	condOpt := strings.TrimSpace(ifParts[0])
	condVal := strings.TrimSpace(ifParts[1])

	userVal, userSet := userOpts[condOpt]
	if !userSet {
		return
	}

	if fmt.Sprint(userVal) != condVal {
		return
	}

	// Condition is met — check what the "then" clause says
	thenLower := strings.ToLower(strings.TrimSpace(cond.Then))

	switch {
	case thenLower == "disabled":
		// The option should not be actively set when condition is met
		if _, optSet := userOpts[optName]; optSet {
			result.addWarning(optName, fmt.Sprintf(
				"option is disabled when %s=%s, but is explicitly set",
				condOpt, condVal,
			))
		}
	case thenLower == "ignored":
		if _, optSet := userOpts[optName]; optSet {
			result.addWarning(optName, fmt.Sprintf(
				"option is ignored when %s=%s",
				condOpt, condVal,
			))
		}
	case strings.Contains(thenLower, "="):
		// "then" specifies an expected value, e.g. "tiebreak=pathname,length"
		thenParts := strings.SplitN(cond.Then, "=", 2)
		if len(thenParts) == 2 {
			targetOpt := strings.TrimSpace(thenParts[0])
			expectedVal := strings.TrimSpace(thenParts[1])

			if userTargetVal, set := userOpts[targetOpt]; set {
				if fmt.Sprint(userTargetVal) != expectedVal {
					result.addWarning(targetOpt, fmt.Sprintf(
						"expected %q when %s=%s (from %s condition), but got %q",
						expectedVal, condOpt, condVal, optName, fmt.Sprint(userTargetVal),
					))
				}
			}
		}
	}
}

// ValidateConfig runs all validation checks on user options and returns
// a combined result. This is the main entry point for validation.
func ValidateConfig(userOpts Options, defsDir string) (*ValidationResult, error) {
	optionsPath := filepath.Join(defsDir, "options.yaml")
	optDefs, err := LoadOptionDefs(optionsPath)
	if err != nil {
		return nil, err
	}

	flat := optDefs.FlattenOptionDefs()
	combined := &ValidationResult{}

	// Phase 1: Validate individual options (type, enum values)
	typeResult := ValidateOptions(userOpts, flat)
	combined.Errors = append(combined.Errors, typeResult.Errors...)
	combined.Warnings = append(combined.Warnings, typeResult.Warnings...)

	// Phase 2: Check cross-option effects
	effectResult := ValidateEffects(userOpts, flat)
	combined.Errors = append(combined.Errors, effectResult.Errors...)
	combined.Warnings = append(combined.Warnings, effectResult.Warnings...)

	// Phase 3: Check conditional relationships
	condResult := ValidateConditions(userOpts, flat)
	combined.Errors = append(combined.Errors, condResult.Errors...)
	combined.Warnings = append(combined.Warnings, condResult.Warnings...)

	return combined, nil
}
