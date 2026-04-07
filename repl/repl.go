package repl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Config holds REPL configuration.
type Config struct {
	LogLevel    string `yaml:"log-level"`
	JSONIndent  int    `yaml:"json-indent"`
	MaxItems    int    `yaml:"max-items"`
	ColorOutput bool   `yaml:"color-output"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		LogLevel:    "info",
		JSONIndent:  2,
		MaxItems:    100,
		ColorOutput: true,
	}
}

// REPL is the interactive read-eval-print loop.
type REPL struct {
	client *Client
	config Config
	log    *log.Logger
}

// New creates a REPL instance.
func New(client *Client, cfg Config) *REPL {
	lvl, _ := log.ParseLevel(cfg.LogLevel)
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Level:           lvl,
		Prefix:          "repl",
		ReportTimestamp: true,
	})
	return &REPL{client: client, config: cfg, log: logger}
}

// Run starts the interactive REPL loop.
func (r *REPL) Run() error {
	r.printBanner()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(promptStyle().Render("fzf> "))
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if r.handleBuiltin(line) {
			continue
		}

		// Default: send as action to fzf
		resp, err := r.client.SendAction(line)
		if err != nil {
			r.printError("send failed: %v", err)
		} else if resp != "" {
			fmt.Println(dimStyle().Render(resp))
		} else {
			fmt.Println(okStyle().Render("ok"))
		}
	}

	return scanner.Err()
}

// handleBuiltin processes built-in REPL commands. Returns true if handled.
func (r *REPL) handleBuiltin(line string) bool {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "help", "?":
		r.printHelp()
		return true

	case "quit", "exit", "q":
		fmt.Println(dimStyle().Render("goodbye"))
		os.Exit(0)

	case "state", "get":
		limit, offset := r.config.MaxItems, 0
		if len(parts) > 1 {
			limit, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			offset, _ = strconv.Atoi(parts[2])
		}
		r.printState(limit, offset)
		return true

	case "raw":
		limit, offset := r.config.MaxItems, 0
		if len(parts) > 1 {
			limit, _ = strconv.Atoi(parts[1])
		}
		r.printRawJSON(limit, offset)
		return true

	case "ping":
		if err := r.client.Ping(); err != nil {
			r.printError("ping failed: %v", err)
		} else {
			fmt.Println(okStyle().Render("pong") + " " + dimStyle().Render(r.client.ConnectionInfo()))
		}
		return true

	case "actions":
		r.printActionsRef()
		return true

	case "events":
		r.printEventsRef()
		return true

	case "keys":
		r.printKeysRef()
		return true

	case "info":
		r.printConnectionInfo()
		return true

	default:
		return false
	}
	return false
}

// printState fetches and pretty-prints the fzf state.
func (r *REPL) printState(limit, offset int) {
	state, err := r.client.GetState(limit, offset)
	if err != nil {
		r.printError("get state: %v", err)
		return
	}

	header := headerStyle()
	key := keyStyle()
	val := valStyle()
	dim := dimStyle()

	fmt.Println(header.Render("fzf state"))
	fmt.Println(dim.Render(strings.Repeat("─", 50)))
	fmt.Printf("  %s  %s\n", key.Render("query:"), val.Render(fmt.Sprintf("%q", state.Query)))
	fmt.Printf("  %s  %s\n", key.Render("position:"), val.Render(strconv.Itoa(state.Position)))
	fmt.Printf("  %s  %s\n", key.Render("total:"), val.Render(strconv.Itoa(state.TotalCount)))
	fmt.Printf("  %s  %s\n", key.Render("matches:"), val.Render(strconv.Itoa(state.MatchCount)))
	fmt.Printf("  %s  %s\n", key.Render("selected:"), val.Render(strconv.Itoa(len(state.Selected))))
	fmt.Printf("  %s  %s\n", key.Render("reading:"), val.Render(strconv.FormatBool(state.Reading)))
	fmt.Printf("  %s  %s\n", key.Render("sort:"), val.Render(strconv.FormatBool(state.Sort)))

	if state.Current != nil {
		fmt.Printf("  %s  %s\n", key.Render("current:"), val.Render(state.Current.Text))
	}

	if len(state.Matches) > 0 {
		fmt.Println()
		fmt.Println(header.Render(fmt.Sprintf("matches (%d/%d)", len(state.Matches), state.MatchCount)))
		fmt.Println(dim.Render(strings.Repeat("─", 50)))
		for i, m := range state.Matches {
			idx := dim.Render(fmt.Sprintf("%4d", m.Index))
			fmt.Printf("  %s  %s\n", idx, m.Text)
			if i >= 20 && len(state.Matches) > 25 {
				fmt.Printf("  %s\n", dim.Render(fmt.Sprintf("... and %d more", len(state.Matches)-i-1)))
				break
			}
		}
	}

	if len(state.Selected) > 0 {
		fmt.Println()
		fmt.Println(header.Render(fmt.Sprintf("selected (%d)", len(state.Selected))))
		fmt.Println(dim.Render(strings.Repeat("─", 50)))
		for _, s := range state.Selected {
			fmt.Printf("  %s  %s\n", dim.Render(fmt.Sprintf("%4d", s.Index)), s.Text)
		}
	}
}

// printRawJSON fetches and pretty-prints the raw JSON.
func (r *REPL) printRawJSON(limit, offset int) {
	data, err := r.client.GetStateRaw(limit, offset)
	if err != nil {
		r.printError("get raw: %v", err)
		return
	}

	var pretty json.RawMessage
	if err := json.Unmarshal(data, &pretty); err != nil {
		fmt.Println(string(data))
		return
	}

	indented, err := json.MarshalIndent(pretty, "", strings.Repeat(" ", r.config.JSONIndent))
	if err != nil {
		fmt.Println(string(data))
		return
	}

	if r.config.ColorOutput {
		fmt.Println(colorizeJSON(string(indented)))
	} else {
		fmt.Println(string(indented))
	}
}

// printBanner shows the startup banner.
func (r *REPL) printBanner() {
	banner := headerStyle().Render("fzf-repl")
	conn := dimStyle().Render(r.client.ConnectionInfo())
	fmt.Fprintf(os.Stderr, "%s  %s\n", banner, conn)
	fmt.Fprintln(os.Stderr, dimStyle().Render("Type 'help' for commands, or send fzf actions directly."))
	fmt.Fprintln(os.Stderr)
}

func (r *REPL) printError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, errStyle().Render("error: "+msg))
}

// printHelp shows the built-in command reference.
func (r *REPL) printHelp() {
	h := headerStyle()
	d := dimStyle()
	k := keyStyle()

	fmt.Println(h.Render("Commands"))
	fmt.Println(d.Render(strings.Repeat("─", 50)))
	cmds := []struct{ cmd, desc string }{
		{"help, ?", "Show this help"},
		{"quit, exit, q", "Exit the REPL"},
		{"ping", "Test connection to fzf"},
		{"info", "Show connection details"},
		{"state [limit] [offset]", "Pretty-print fzf state"},
		{"raw [limit]", "Show raw JSON state"},
		{"actions", "List available fzf actions"},
		{"events", "List available fzf events"},
		{"keys", "List common key names"},
		{"<anything else>", "Send as action to fzf (POST)"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s  %s\n", k.Render(fmt.Sprintf("%-28s", c.cmd)), d.Render(c.desc))
	}

	fmt.Println()
	fmt.Println(h.Render("Examples"))
	fmt.Println(d.Render(strings.Repeat("─", 50)))
	examples := []string{
		"up",
		"down",
		"reload(seq 100)",
		"change-query(hello)",
		"change-prompt(search> )",
		"select-all+accept",
		"reload(find . -type f)+change-prompt(files> )",
	}
	for _, e := range examples {
		fmt.Printf("  %s\n", valStyle().Render(e))
	}
}

func (r *REPL) printActionsRef() {
	h := headerStyle()
	d := dimStyle()
	k := keyStyle()

	fmt.Println(h.Render("Common Actions"))
	fmt.Println(d.Render(strings.Repeat("─", 50)))
	actions := []struct{ name, desc string }{
		{"abort", "Exit without selection"},
		{"accept", "Confirm selection"},
		{"up / down", "Move cursor"},
		{"first / last", "Jump to first/last"},
		{"page-up / page-down", "Page navigation"},
		{"select / deselect", "Select/deselect current"},
		{"select-all / deselect-all", "Select/deselect all"},
		{"toggle / toggle-all", "Toggle selection"},
		{"reload(CMD)", "Replace input from command"},
		{"change-query(STR)", "Set query string"},
		{"change-prompt(STR)", "Set prompt"},
		{"change-header(STR)", "Set header"},
		{"change-preview(CMD)", "Set preview command"},
		{"change-preview-window(OPTS)", "Set preview layout"},
		{"execute(CMD)", "Run command (suspends fzf)"},
		{"execute-silent(CMD)", "Run command silently"},
		{"become(CMD)", "Replace fzf with command"},
		{"transform(CMD)", "Dynamic action from command output"},
		{"pos(N)", "Jump to position N"},
		{"put(STR)", "Insert text into query"},
		{"print(STR)", "Add to output queue"},
		{"rebind(KEYS) / unbind(KEYS)", "Manage key bindings"},
		{"toggle-preview", "Show/hide preview"},
		{"toggle-sort", "Toggle sorting"},
		{"clear-query", "Clear search query"},
		{"preview-up / preview-down", "Scroll preview"},
		{"preview-half-page-up/down", "Scroll preview half page"},
	}
	for _, a := range actions {
		fmt.Printf("  %s  %s\n", k.Render(fmt.Sprintf("%-32s", a.name)), d.Render(a.desc))
	}
}

func (r *REPL) printEventsRef() {
	h := headerStyle()
	d := dimStyle()
	k := keyStyle()

	fmt.Println(h.Render("Events"))
	fmt.Println(d.Render(strings.Repeat("─", 50)))
	events := []struct{ name, desc string }{
		{"start", "fzf finder starts"},
		{"load", "Input stream complete"},
		{"result", "Filtering complete"},
		{"change", "Query changes"},
		{"focus", "Focused item changes"},
		{"multi", "Selection set changes"},
		{"one", "Exactly one match"},
		{"zero", "No matches"},
		{"resize", "Terminal resized"},
		{"backward-eof", "Backspace on empty query"},
		{"jump / jump-cancel", "Jump mode events"},
		{"click-header / click-footer", "Mouse clicks on header/footer"},
	}
	for _, e := range events {
		fmt.Printf("  %s  %s\n", k.Render(fmt.Sprintf("%-24s", e.name)), d.Render(e.desc))
	}
}

func (r *REPL) printKeysRef() {
	h := headerStyle()
	d := dimStyle()

	fmt.Println(h.Render("Key Names"))
	fmt.Println(d.Render(strings.Repeat("─", 50)))
	groups := []struct{ name, keys string }{
		{"Control", "ctrl-a..z, ctrl-space, ctrl-/, ctrl-\\"},
		{"Alt", "alt-a..z, alt-0..9, alt-enter, alt-space, alt-bs"},
		{"Function", "f1..f12"},
		{"Navigation", "up, down, left, right, home, end, page-up, page-down"},
		{"Basic", "enter, space, tab, shift-tab, esc, backspace, delete"},
		{"Mouse", "left-click, right-click, double-click, scroll-up/down"},
		{"Modifiers", "shift-, ctrl-, alt-, ctrl-alt- (combine with above)"},
	}
	for _, g := range groups {
		fmt.Printf("  %s  %s\n", keyStyle().Render(fmt.Sprintf("%-12s", g.name)), d.Render(g.keys))
	}
}

func (r *REPL) printConnectionInfo() {
	k := keyStyle()
	v := valStyle()
	d := dimStyle()

	fmt.Println(headerStyle().Render("Connection"))
	fmt.Println(d.Render(strings.Repeat("─", 50)))
	fmt.Printf("  %s  %s\n", k.Render("endpoint:"), v.Render(r.client.ConnectionInfo()))
	if r.client.APIKey != "" {
		fmt.Printf("  %s  %s\n", k.Render("auth:"), v.Render("api-key set"))
	} else {
		fmt.Printf("  %s  %s\n", k.Render("auth:"), d.Render("none"))
	}
	fmt.Printf("  %s  %s\n", k.Render("log-level:"), v.Render(r.config.LogLevel))
}

// RunNonInteractive sends a single action and exits.
func RunNonInteractive(client *Client, action string, w io.Writer) error {
	resp, err := client.SendAction(action)
	if err != nil {
		return err
	}
	if resp != "" {
		fmt.Fprintln(w, resp)
	}
	return nil
}

// Styles
func promptStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
}
func headerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
}
func keyStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("14")) }
func valStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("15")) }
func dimStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("8")) }
func okStyle() lipgloss.Style  { return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) }
func errStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("9")) }

// colorizeJSON adds basic color to JSON output.
func colorizeJSON(s string) string {
	kc := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	sc := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	nc := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	var result strings.Builder
	inKey := false
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			result.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			result.WriteByte(ch)
			escaped = true
			continue
		}

		switch {
		case ch == '"' && !inString && !inKey:
			// Peek ahead: is this a key (followed by :)?
			end := strings.Index(s[i+1:], `"`)
			if end >= 0 && end+i+2 < len(s) {
				rest := strings.TrimSpace(s[i+end+2:])
				if len(rest) > 0 && rest[0] == ':' {
					inKey = true
					result.WriteString(kc.Render(`"`))
					continue
				}
			}
			inString = true
			result.WriteString(sc.Render(`"`))
		case ch == '"' && inKey:
			result.WriteString(kc.Render(`"`))
			inKey = false
		case ch == '"' && inString:
			result.WriteString(sc.Render(`"`))
			inString = false
		case inKey:
			result.WriteString(kc.Render(string(ch)))
		case inString:
			result.WriteString(sc.Render(string(ch)))
		case ch >= '0' && ch <= '9' || ch == '-' || ch == '.':
			result.WriteString(nc.Render(string(ch)))
		case ch == 't' || ch == 'f' || ch == 'n': // true/false/null
			result.WriteString(bc.Render(string(ch)))
		default:
			result.WriteByte(ch)
		}
	}
	return result.String()
}
