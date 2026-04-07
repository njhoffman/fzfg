package fzfg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v2"
)

// Keybind represents a single key binding with its source and description.
type Keybind struct {
	Key         string
	Action      string
	Description string
	Source      string // "default", "fzfrc", "config", "module"
	Category    string // grouping for display
}

// AllBindableKeys is the comprehensive list of keys fzf can bind.
var AllBindableKeys = func() []string {
	keys := []string{
		"enter", "space", "backspace", "tab", "shift-tab", "esc",
		"delete", "insert", "up", "down", "left", "right",
		"home", "end", "page-up", "page-down",
		"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
		"double-click", "left-click", "right-click",
		"scroll-up", "scroll-down",
		"shift-up", "shift-down", "shift-left", "shift-right",
	}
	// ctrl- combos
	for c := 'a'; c <= 'z'; c++ {
		keys = append(keys, fmt.Sprintf("ctrl-%c", c))
	}
	keys = append(keys, "ctrl-space", "ctrl-/")
	// alt- combos
	for c := 'a'; c <= 'z'; c++ {
		keys = append(keys, fmt.Sprintf("alt-%c", c))
	}
	for c := 'A'; c <= 'Z'; c++ {
		keys = append(keys, fmt.Sprintf("alt-%c", c))
	}
	keys = append(keys,
		"alt-enter", "alt-space", "alt-backspace",
		"alt-up", "alt-down", "alt-left", "alt-right",
		"alt-/", "alt-?", "alt-_",
	)
	// common ctrl-alt combos
	for c := 'a'; c <= 'z'; c++ {
		keys = append(keys, fmt.Sprintf("ctrl-alt-%c", c))
	}
	return keys
}()

// ParseFzfrcBindings reads an fzfrc file and extracts bindings with
// their preceding comment as the description.
func ParseFzfrcBindings(path string) ([]Keybind, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var binds []Keybind
	var pendingComment string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Capture comment lines (only standalone comments, not inline)
		if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "# --") {
			pendingComment = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			continue
		}

		// Check for --bind lines
		if strings.HasPrefix(trimmed, "--bind") {
			// Extract the binding spec
			spec := trimmed
			spec = strings.TrimPrefix(spec, "--bind")
			spec = strings.TrimSpace(spec)
			spec = strings.Trim(spec, `"'`)

			// Parse key:action
			colonIdx := strings.Index(spec, ":")
			if colonIdx > 0 {
				key := spec[:colonIdx]
				action := spec[colonIdx+1:]

				// Handle continuation lines and composite actions
				// Strip the action to first composite for display
				displayAction := action
				if plusIdx := strings.Index(action, "+"); plusIdx > 0 && len(action) > 60 {
					displayAction = action[:plusIdx] + "+..."
				}

				binds = append(binds, Keybind{
					Key:         key,
					Action:      displayAction,
					Description: pendingComment,
					Source:      "fzfrc",
				})
			}
			pendingComment = ""
			continue
		}

		// Non-comment, non-bind line resets pending comment
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			pendingComment = ""
		}
	}

	return binds, nil
}

// ParseConfigBindings extracts bindings from the fzfg.yaml bindings section.
func ParseConfigBindings(confPath string) ([]Keybind, error) {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Bindings map[string]map[string]interface{} `yaml:"bindings"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var binds []Keybind
	for category, group := range raw.Bindings {
		if category == "all" {
			continue // skip the merged alias
		}
		for key, action := range group {
			if key == "<<" {
				continue // skip YAML merge keys
			}
			binds = append(binds, Keybind{
				Key:      key,
				Action:   fmt.Sprint(action),
				Source:   "config",
				Category: category,
			})
		}
	}

	return binds, nil
}

// FzfDefaultBindings returns fzf's built-in default key bindings.
func FzfDefaultBindings() []Keybind {
	defaults := []struct {
		key, action, desc, cat string
	}{
		{"ctrl-c", "abort", "Exit without selection", "core"},
		{"ctrl-g", "abort", "Exit without selection", "core"},
		{"ctrl-q", "abort", "Exit without selection", "core"},
		{"esc", "abort", "Exit without selection", "core"},
		{"enter", "accept", "Confirm selection", "core"},
		{"double-click", "accept", "Confirm selection", "core"},
		{"ctrl-b", "backward-char", "Cursor left", "editing"},
		{"left", "backward-char", "Cursor left", "editing"},
		{"ctrl-h", "backward-delete-char", "Delete char before cursor", "editing"},
		{"backspace", "backward-delete-char", "Delete char before cursor", "editing"},
		{"alt-backspace", "backward-kill-word", "Delete word before cursor", "editing"},
		{"alt-b", "backward-word", "Previous word", "editing"},
		{"shift-left", "backward-word", "Previous word", "editing"},
		{"ctrl-a", "beginning-of-line", "Start of line", "editing"},
		{"home", "beginning-of-line", "Start of line", "editing"},
		{"ctrl-l", "clear-screen", "Redraw screen", "display"},
		{"delete", "delete-char", "Delete char at cursor", "editing"},
		{"ctrl-d", "delete-char/eof", "Delete or EOF", "editing"},
		{"ctrl-j", "down", "Move down", "navigation"},
		{"down", "down", "Move down", "navigation"},
		{"ctrl-n", "down-match", "Next match", "navigation"},
		{"alt-down", "down-match", "Next match", "navigation"},
		{"ctrl-e", "end-of-line", "End of line", "editing"},
		{"end", "end-of-line", "End of line", "editing"},
		{"ctrl-f", "forward-char", "Cursor right", "editing"},
		{"right", "forward-char", "Cursor right", "editing"},
		{"alt-f", "forward-word", "Next word", "editing"},
		{"shift-right", "forward-word", "Next word", "editing"},
		{"alt-d", "kill-word", "Delete word after cursor", "editing"},
		{"page-down", "page-down", "Page down", "navigation"},
		{"page-up", "page-up", "Page up", "navigation"},
		{"shift-down", "preview-down", "Scroll preview down", "preview"},
		{"shift-up", "preview-up", "Scroll preview up", "preview"},
		{"tab", "toggle+down", "Toggle select + down", "selection"},
		{"shift-tab", "toggle+up", "Toggle select + up", "selection"},
		{"ctrl-/", "toggle-wrap-word", "Toggle word wrap", "display"},
		{"alt-/", "toggle-wrap-word", "Toggle word wrap", "display"},
		{"ctrl-u", "unix-line-discard", "Delete to start of line", "editing"},
		{"ctrl-w", "unix-word-rubout", "Delete word (Unix)", "editing"},
		{"ctrl-k", "up", "Move up", "navigation"},
		{"up", "up", "Move up", "navigation"},
		{"ctrl-p", "up-match", "Previous match", "navigation"},
		{"alt-up", "up-match", "Previous match", "navigation"},
		{"ctrl-y", "yank", "Paste deleted text", "editing"},
	}

	var binds []Keybind
	for _, d := range defaults {
		binds = append(binds, Keybind{
			Key:         d.key,
			Action:      d.action,
			Description: d.desc,
			Source:      "default",
			Category:    d.cat,
		})
	}
	return binds
}

// CollectAllBindings gathers bindings from all sources in precedence order.
// Later sources override earlier ones for the same key.
func CollectAllBindings(fzfrcPath, configPath string) []Keybind {
	byKey := map[string]Keybind{}

	// Layer 1: fzf defaults (lowest precedence)
	for _, b := range FzfDefaultBindings() {
		byKey[b.Key] = b
	}

	// Layer 2: fzfrc
	if fzfrcPath != "" {
		if binds, err := ParseFzfrcBindings(fzfrcPath); err == nil {
			for _, b := range binds {
				byKey[b.Key] = b
			}
		}
	}

	// Layer 3: fzfg config
	if configPath != "" {
		if binds, err := ParseConfigBindings(configPath); err == nil {
			for _, b := range binds {
				byKey[b.Key] = b
			}
		}
	}

	// Flatten and sort
	var all []Keybind
	for _, b := range byKey {
		all = append(all, b)
	}
	sort.Slice(all, func(i, j int) bool {
		return keySortOrder(all[i].Key) < keySortOrder(all[j].Key)
	})
	return all
}

// AvailableKeys returns all bindable keys NOT currently bound.
func AvailableKeys(bound []Keybind) []string {
	usedKeys := map[string]bool{}
	for _, b := range bound {
		usedKeys[strings.ToLower(b.Key)] = true
	}

	var available []string
	for _, k := range AllBindableKeys {
		if !usedKeys[strings.ToLower(k)] {
			available = append(available, k)
		}
	}
	return available
}

// FormatKeybindPreview generates a colorful which-key style preview string.
func FormatKeybindPreview(binds []Keybind) string {
	keyCol := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true).Width(18)
	actionCol := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Width(30)
	descCol := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	srcDefault := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	srcFzfrc := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	srcConfig := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	srcModule := lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// Group by category
	categories := map[string][]Keybind{}
	catOrder := []string{}
	for _, b := range binds {
		cat := b.Category
		if cat == "" {
			cat = b.Source
		}
		if _, seen := categories[cat]; !seen {
			catOrder = append(catOrder, cat)
		}
		categories[cat] = append(categories[cat], b)
	}

	var buf strings.Builder
	buf.WriteString(headerStyle.Render("  Key Bindings") + "\n")
	buf.WriteString(dimStyle.Render("  "+strings.Repeat("─", 76)) + "\n")

	// Header row
	buf.WriteString(fmt.Sprintf("  %s %s %s %s\n",
		keyCol.Render("KEY"),
		actionCol.Render("ACTION"),
		lipgloss.NewStyle().Width(6).Render("SRC"),
		descCol.Render("DESCRIPTION"),
	))
	buf.WriteString(dimStyle.Render("  "+strings.Repeat("─", 76)) + "\n")

	for _, cat := range catOrder {
		bindsInCat := categories[cat]
		buf.WriteString(dimStyle.Render(fmt.Sprintf("\n  [%s]", cat)) + "\n")

		for _, b := range bindsInCat {
			var srcStyled string
			switch b.Source {
			case "default":
				srcStyled = srcDefault.Render("def")
			case "fzfrc":
				srcStyled = srcFzfrc.Render("rc ")
			case "config":
				srcStyled = srcConfig.Render("cfg")
			case "module":
				srcStyled = srcModule.Render("mod")
			default:
				srcStyled = dimStyle.Render(b.Source[:3])
			}

			desc := b.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			buf.WriteString(fmt.Sprintf("  %s %s %s  %s\n",
				keyCol.Render(b.Key),
				actionCol.Render(b.Action),
				srcStyled,
				descCol.Render(desc),
			))
		}
	}

	return buf.String()
}

// FormatAvailableKeys generates a formatted list of unbound keys.
func FormatAvailableKeys(available []string) string {
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var buf strings.Builder
	buf.WriteString(headerStyle.Render("  Available (Unbound) Keys") + "\n")
	buf.WriteString(dimStyle.Render("  "+strings.Repeat("─", 60)) + "\n")

	// Group by prefix
	groups := map[string][]string{}
	groupOrder := []string{"ctrl-", "alt-", "ctrl-alt-", "f", "other"}
	for _, k := range available {
		switch {
		case strings.HasPrefix(k, "ctrl-alt-"):
			groups["ctrl-alt-"] = append(groups["ctrl-alt-"], k)
		case strings.HasPrefix(k, "ctrl-"):
			groups["ctrl-"] = append(groups["ctrl-"], k)
		case strings.HasPrefix(k, "alt-"):
			groups["alt-"] = append(groups["alt-"], k)
		case strings.HasPrefix(k, "f") && len(k) <= 3:
			groups["f"] = append(groups["f"], k)
		default:
			groups["other"] = append(groups["other"], k)
		}
	}

	for _, prefix := range groupOrder {
		keys := groups[prefix]
		if len(keys) == 0 {
			continue
		}

		label := prefix
		if label == "f" {
			label = "function"
		} else if label == "other" {
			label = "other"
		}

		buf.WriteString(fmt.Sprintf("\n  %s %s\n",
			dimStyle.Render(fmt.Sprintf("[%s]", label)),
			dimStyle.Render(fmt.Sprintf("(%d)", len(keys))),
		))

		// Print in columns
		cols := 4
		for i, k := range keys {
			buf.WriteString(fmt.Sprintf("  %s", keyStyle.Render(fmt.Sprintf("%-18s", k))))
			if (i+1)%cols == 0 {
				buf.WriteString("\n")
			}
		}
		if len(keys)%cols != 0 {
			buf.WriteString("\n")
		}
	}

	buf.WriteString(fmt.Sprintf("\n  %s\n",
		dimStyle.Render(fmt.Sprintf("Total: %d available keys", len(available))),
	))

	return buf.String()
}

// keySortOrder returns a sortable string for consistent key ordering.
func keySortOrder(key string) string {
	prefixOrder := map[string]string{
		"ctrl-alt-": "0",
		"ctrl-":     "1",
		"alt-":      "2",
		"shift-":    "3",
	}
	for prefix, order := range prefixOrder {
		if strings.HasPrefix(key, prefix) {
			return order + key
		}
	}
	return "9" + key
}

// ResolveFzfrcPath finds the fzfrc file.
func ResolveFzfrcPath() string {
	if f := os.Getenv("FZF_DEFAULT_OPTS_FILE"); f != "" && isFile(f) {
		return f
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".fzf", ".fzfrc"),
		filepath.Join(home, ".fzfrc"),
	}
	for _, c := range candidates {
		if isFile(c) {
			return c
		}
	}
	return ""
}
