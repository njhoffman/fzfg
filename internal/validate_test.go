package fzfg

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to build a minimal definitions file for testing
func writeTestDefs(t *testing.T, dir string) string {
	t.Helper()
	defsDir := filepath.Join(dir, "definitions")
	os.MkdirAll(defsDir, 0755)

	content := `options:
  search:
    exact:
      description: Enable exact-match.
      aliases: [-e, --exact]
      required: false
      type: boolean
      default: false
    scheme:
      description: Choose scoring scheme.
      aliases: [--scheme]
      required: false
      type: enum
      default: default
      value:
        type: enum
        values:
          default:
            description: Generic scoring.
          path:
            description: Path scoring.
            effects:
              - tiebreak=pathname,length
          history:
            description: History scoring.
            effects:
              - tiebreak=index
    tiebreak:
      description: Sort criteria.
      aliases: [--tiebreak]
      required: false
      type: list
      default: length
      value:
        type: enum
        values:
          length:
            description: Prefer shorter.
          chunk:
            description: Prefer shorter chunk.
          pathname:
            description: Prefer filename match.
          begin:
            description: Prefer begin match.
          end:
            description: Prefer end match.
          index:
            description: Prefer earlier input.
      conditions:
        - if: scheme=path
          then: tiebreak=pathname,length
        - if: scheme=history
          then: tiebreak=index
    ignore-case:
      description: Case-insensitive match.
      aliases: [--ignore-case]
      required: false
      type: boolean
      default: false
      effects:
        - smart-case=false
    smart-case:
      description: Smart-case match.
      aliases: [--smart-case]
      required: false
      type: boolean
      default: true
      conditions:
        - if: ignore-case=true
          then: disabled
  layout:
    border:
      description: Border style.
      aliases: [--border]
      required: false
      type: enum
      default: rounded
      value:
        type: enum
        values:
          rounded:
            description: Rounded.
          sharp:
            description: Sharp.
          none:
            description: No border.
    border-label:
      description: Label on border.
      aliases: [--border-label]
      required: false
      type: string
      default: null
      conditions:
        - if: border=none
          then: ignored
  list:
    scroll-off:
      description: Lines to keep above/below cursor.
      aliases: [--scroll-off]
      required: false
      type: integer
      default: 3
    gap:
      description: Empty lines between items.
      aliases: [--gap]
      required: false
      type: integer
      default: 0
    multi:
      description: Enable multi-select.
      aliases: [-m, --multi]
      required: false
      type: integer
      default: false
`
	os.WriteFile(filepath.Join(defsDir, "options.yaml"), []byte(content), 0644)
	return defsDir
}

func TestValidateOptions_UnknownOption(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"nonexistent-option": true}
	result := ValidateOptions(userOpts, flat)

	if !result.HasErrors() {
		t.Error("expected error for unknown option")
	}
	if result.Errors[0].Option != "nonexistent-option" {
		t.Errorf("expected error on 'nonexistent-option', got %q", result.Errors[0].Option)
	}
}

func TestValidateOptions_ValidBoolean(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"exact": true}
	result := ValidateOptions(userOpts, flat)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestValidateOptions_InvalidBooleanType(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"exact": "yes"}
	result := ValidateOptions(userOpts, flat)

	if !result.HasErrors() {
		t.Error("expected type error for boolean given string")
	}
}

func TestValidateOptions_ValidEnum(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"border": "sharp"}
	result := ValidateOptions(userOpts, flat)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestValidateOptions_InvalidEnum(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"border": "dotted"}
	result := ValidateOptions(userOpts, flat)

	if !result.HasErrors() {
		t.Error("expected error for invalid enum value 'dotted'")
	}
}

func TestValidateOptions_ValidInteger(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"scroll-off": 5}
	result := ValidateOptions(userOpts, flat)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestValidateOptions_InvalidIntegerType(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"scroll-off": "abc"}
	result := ValidateOptions(userOpts, flat)

	if !result.HasErrors() {
		t.Error("expected error for string given to integer option")
	}
}

func TestValidateOptions_IntegerAcceptsBoolForMulti(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	// multi's default is bool false, should be accepted
	userOpts := Options{"multi": true}
	result := ValidateOptions(userOpts, flat)

	if result.HasErrors() {
		t.Errorf("unexpected errors for multi=true: %v", result.Errors)
	}
}

func TestValidateOptions_ValidListEnum(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"tiebreak": "length,begin"}
	result := ValidateOptions(userOpts, flat)

	if result.HasErrors() {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestValidateOptions_InvalidListEnum(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{"tiebreak": "length,bogus"}
	result := ValidateOptions(userOpts, flat)

	if !result.HasErrors() {
		t.Error("expected error for invalid list enum value 'bogus'")
	}
}

func TestValidateEffects_SchemePathConflict(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	// scheme=path implies tiebreak=pathname,length, but user set tiebreak=length
	userOpts := Options{
		"scheme":   "path",
		"tiebreak": "length",
	}
	result := ValidateEffects(userOpts, flat)

	if len(result.Warnings) == 0 {
		t.Error("expected warning for tiebreak conflicting with scheme=path effect")
	}
}

func TestValidateEffects_SchemePathNoConflict(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	// scheme=path with matching tiebreak — no warning
	userOpts := Options{
		"scheme":   "path",
		"tiebreak": "pathname,length",
	}
	result := ValidateEffects(userOpts, flat)

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestValidateEffects_IgnoreCaseDisablesSmartCase(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	// ignore-case=true has effect smart-case=false, but user set smart-case=true
	userOpts := Options{
		"ignore-case": true,
		"smart-case":  true,
	}
	result := ValidateEffects(userOpts, flat)

	if len(result.Warnings) == 0 {
		t.Error("expected warning for smart-case conflicting with ignore-case effect")
	}
}

func TestValidateConditions_SmartCaseDisabledByIgnoreCase(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{
		"ignore-case": true,
		"smart-case":  true,
	}
	result := ValidateConditions(userOpts, flat)

	if len(result.Warnings) == 0 {
		t.Error("expected warning that smart-case is disabled when ignore-case=true")
	}
}

func TestValidateConditions_BorderLabelIgnoredWhenNoBorder(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{
		"border":       "none",
		"border-label": "My Label",
	}
	result := ValidateConditions(userOpts, flat)

	if len(result.Warnings) == 0 {
		t.Error("expected warning that border-label is ignored when border=none")
	}
}

func TestValidateConditions_TiebreakOverriddenByScheme(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)
	defs, _ := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	flat := defs.FlattenOptionDefs()

	userOpts := Options{
		"scheme":   "history",
		"tiebreak": "length",
	}
	result := ValidateConditions(userOpts, flat)

	if len(result.Warnings) == 0 {
		t.Error("expected warning for tiebreak being overridden by scheme=history")
	}
}

func TestValidateConfig_FullPipeline(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)

	userOpts := Options{
		"exact":      true,
		"border":     "sharp",
		"scroll-off": 5,
		"scheme":     "default",
	}

	result, err := ValidateConfig(userOpts, defsDir)
	if err != nil {
		t.Fatalf("ValidateConfig error: %v", err)
	}
	if result.HasErrors() {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestValidateConfig_MixedErrorsAndWarnings(t *testing.T) {
	dir := t.TempDir()
	defsDir := writeTestDefs(t, dir)

	userOpts := Options{
		"exact":        "not-a-bool", // type error
		"border":       "none",       // valid
		"border-label": "My Label",   // warning: ignored when border=none
		"fake-option":  true,         // unknown option error
	}

	result, err := ValidateConfig(userOpts, defsDir)
	if err != nil {
		t.Fatalf("ValidateConfig error: %v", err)
	}

	if len(result.Errors) < 2 {
		t.Errorf("expected at least 2 errors (type + unknown), got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) < 1 {
		t.Errorf("expected at least 1 warning (border-label ignored), got %d", len(result.Warnings))
	}
}

func TestLoadOptionDefs_ReadsActualFile(t *testing.T) {
	// Test against the real definitions file
	defsPath := filepath.Join("../configs/definitions/options.yaml")
	if _, err := os.Stat(defsPath); os.IsNotExist(err) {
		t.Skip("definitions file not found, skipping integration test")
	}

	defs, err := LoadOptionDefs(defsPath)
	if err != nil {
		t.Fatalf("LoadOptionDefs error: %v", err)
	}

	flat := defs.FlattenOptionDefs()

	// Spot-check a few known options exist
	for _, name := range []string{"exact", "scheme", "border", "multi", "ansi", "reverse"} {
		if _, ok := flat[name]; !ok {
			t.Errorf("expected option %q in definitions", name)
		}
	}

	// Check scheme has enum values
	schemeDef := flat["scheme"]
	if schemeDef.Value == nil || len(schemeDef.Value.Values) == 0 {
		t.Error("scheme should have enum values defined")
	}
	if _, ok := schemeDef.Value.Values["path"]; !ok {
		t.Error("scheme should have 'path' enum value")
	}
}
