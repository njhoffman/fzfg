package fzfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DebugModes parses the --debug flag value into a set of mode names.
func DebugModes(flag string) map[string]bool {
	modes := map[string]bool{}
	if flag == "" {
		return modes
	}
	for _, m := range strings.Split(flag, ",") {
		modes[strings.TrimSpace(m)] = true
	}
	return modes
}

// PrintDebug outputs the requested debug sections.
func PrintDebug(result *InitResult, modes map[string]bool) {
	if len(modes) == 0 {
		return
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	indent := lipgloss.NewStyle().PaddingLeft(2)

	w := os.Stderr
	sep := dimStyle.Render(strings.Repeat("─", 60))

	if modes["summary"] {
		fmt.Fprintln(w)
		fmt.Fprintln(w, headerStyle.Render("Init Summary"))
		fmt.Fprintln(w, sep)
		for _, step := range result.Steps {
			status := okStyle.Render("ok")
			if step.Status == "error" {
				status = errStyle.Render("err")
			} else if step.Status == "warn" || step.Status == "skip" {
				status = warnStyle.Render(step.Status)
			}
			line := fmt.Sprintf("%-12s  [%s]  %s",
				step.Name, status, dimStyle.Render(step.Duration.String()))
			fmt.Fprintln(w, indent.Render(line))
			if step.Message != "" {
				fmt.Fprintln(w, indent.Render(errStyle.Render("  "+step.Message)))
			}
		}
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, "%s  %s\n", indent.Render("total"), dimStyle.Render(result.Total.String()))
	}

	if modes["timings"] {
		fmt.Fprintln(w)
		fmt.Fprintln(w, headerStyle.Render("Timings"))
		fmt.Fprintln(w, sep)
		for _, step := range result.Steps {
			pct := float64(step.Duration) / float64(result.Total) * 100
			bar := strings.Repeat("█", int(pct/5))
			line := fmt.Sprintf("%-12s  %8s  %5.1f%%  %s",
				step.Name, step.Duration.String(), pct, dimStyle.Render(bar))
			fmt.Fprintln(w, indent.Render(line))
		}
		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, "%s  %s\n", indent.Render("total"), result.Total.String())
	}

	if modes["diffs"] {
		fmt.Fprintln(w)
		fmt.Fprintln(w, headerStyle.Render("Config Diffs from Defaults"))
		fmt.Fprintln(w, sep)
		printOptionDiffs(w, result, indent, keyStyle, valStyle, dimStyle)
	}

	if modes["envs"] {
		fmt.Fprintln(w)
		fmt.Fprintln(w, headerStyle.Render("Environment Variables"))
		fmt.Fprintln(w, sep)

		printEnvVar(w, "FZF_DEFAULT_COMMAND", result.FinalCmd, indent, keyStyle, valStyle, dimStyle)
		printEnvVar(w, "FZF_DEFAULT_OPTS", result.FinalOpts, indent, keyStyle, valStyle, dimStyle)

		if result.RscOpts != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, indent.Render(dimStyle.Render("(from fzfrc)")))
			fmt.Fprintln(w, indent.Render(wrapLong(result.RscOpts, 76)))
		}
		if result.EnvOpts != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, indent.Render(dimStyle.Render("(from env FZF_DEFAULT_OPTS)")))
			fmt.Fprintln(w, indent.Render(wrapLong(result.EnvOpts, 76)))
		}
	}

	if modes["keybinds"] {
		fzfrcPath := ResolveFzfrcPath()
		binds := CollectAllBindings(fzfrcPath, result.ConfigFile)
		fmt.Fprint(w, FormatKeybindPreview(binds))
	}

	if modes["available-keys"] {
		fzfrcPath := ResolveFzfrcPath()
		binds := CollectAllBindings(fzfrcPath, result.ConfigFile)
		available := AvailableKeys(binds)
		fmt.Fprint(w, FormatAvailableKeys(available))
	}

	if modes["trace"] {
		fmt.Fprintln(w)
		fmt.Fprintln(w, headerStyle.Render("Config Trace"))
		fmt.Fprintln(w, sep)
		for _, step := range result.Steps {
			if len(step.Snapshot) == 0 {
				continue
			}
			fmt.Fprintln(w, indent.Render(keyStyle.Render(step.Name+":")))
			for k, v := range step.Snapshot {
				display := v
				if len(display) > 80 {
					display = display[:77] + "..."
				}
				fmt.Fprintf(w, "    %s = %s\n",
					dimStyle.Render(k), valStyle.Render(display))
			}
		}
	}

	fmt.Fprintln(w)
}

// printEnvVar outputs a single env var with formatting.
func printEnvVar(w *os.File, name, value string, indent, keyStyle, valStyle, dimStyle lipgloss.Style) {
	if value == "" {
		fmt.Fprintf(w, "%s  %s\n",
			indent.Render(keyStyle.Render(name)),
			dimStyle.Render("(not set)"))
	} else {
		fmt.Fprintln(w, indent.Render(keyStyle.Render(name)))
		fmt.Fprintln(w, indent.Render(wrapLong(value, 76)))
	}
}

// printOptionDiffs shows user options that differ from fzf defaults.
func printOptionDiffs(w *os.File, result *InitResult, indent, keyStyle, valStyle, dimStyle lipgloss.Style) {
	// Load definitions if available
	if result.ConfigFile == "" {
		fmt.Fprintln(w, indent.Render(dimStyle.Render("(no config loaded)")))
		return
	}

	defsDir := filepath.Join(filepath.Dir(result.ConfigFile), "definitions")
	optDefs, err := LoadOptionDefs(filepath.Join(defsDir, "options.yaml"))
	if err != nil {
		fmt.Fprintln(w, indent.Render(dimStyle.Render("(definitions not available)")))
		return
	}
	flat := optDefs.FlattenOptionDefs()

	diffCount := 0
	for groupName, opts := range result.Config.Options {
		groupDiffs := []string{}
		for optName, userVal := range opts {
			def, known := flat[optName]
			if !known {
				groupDiffs = append(groupDiffs, fmt.Sprintf(
					"%s = %v %s",
					keyStyle.Render(optName),
					valStyle.Render(fmt.Sprint(userVal)),
					dimStyle.Render("(unknown option)"),
				))
				continue
			}

			// Compare against default
			defStr := fmt.Sprint(def.Default)
			userStr := fmt.Sprint(userVal)
			if defStr != userStr {
				groupDiffs = append(groupDiffs, fmt.Sprintf(
					"%s = %s %s",
					keyStyle.Render(optName),
					valStyle.Render(userStr),
					dimStyle.Render(fmt.Sprintf("(default: %s)", defStr)),
				))
				diffCount++
			}
		}

		if len(groupDiffs) > 0 {
			fmt.Fprintln(w, indent.Render(keyStyle.Render("["+groupName+"]")))
			for _, d := range groupDiffs {
				fmt.Fprintln(w, "    "+d)
			}
		}
	}

	if diffCount == 0 {
		fmt.Fprintln(w, indent.Render(dimStyle.Render("(no differences from defaults)")))
	}
}

// wrapLong wraps a long string at word boundaries.
func wrapLong(s string, width int) string {
	if len(s) <= width {
		return s
	}
	words := strings.Fields(s)
	var lines []string
	line := ""
	for _, w := range words {
		if len(line)+len(w)+1 > width && line != "" {
			lines = append(lines, line)
			line = w
		} else {
			if line != "" {
				line += " "
			}
			line += w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n    ")
}
