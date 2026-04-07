package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/njhoffman/fzfg/repl"
)

func main() {
	socketPath := flag.String("socket", "", "Path to fzf Unix socket (default: $FZF_SOCK)")
	tcpAddr := flag.String("tcp", "", "TCP address of fzf (e.g. localhost:6266, default: $FZF_PORT)")
	apiKey := flag.String("api-key", "", "API key for authentication (default: $FZF_API_KEY)")
	action := flag.String("send", "", "Send action and exit (non-interactive)")
	getState := flag.Bool("state", false, "Get fzf state as JSON and exit")
	limit := flag.Int("limit", 100, "Limit for state queries")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	createPane := flag.Bool("create-pane", false, "Create a tmux pane with fzf --listen")
	paneSocket := flag.String("pane-socket", "", "Socket path for new fzf pane")
	paneArgs := flag.String("pane-args", "", "Extra args for fzf in new pane (comma-separated)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `fzf-repl - Interactive REPL for controlling fzf via socket

USAGE:
  fzf-repl [options]              Start interactive REPL
  fzf-repl -send "action"         Send action and exit
  fzf-repl -state                 Get fzf state as JSON

OPTIONS:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
ENVIRONMENT:
  FZF_SOCK      Path to fzf Unix socket
  FZF_PORT      TCP port of fzf listener
  FZF_API_KEY   Authentication key

EXAMPLES:
  export FZF_SOCK=/tmp/fzf.sock
  fzf --listen=$FZF_SOCK &
  fzf-repl

  fzf-repl -send "reload(seq 100)+change-prompt(hundred> )"
  fzf-repl -state -limit 50 | jq .
  fzf-repl -create-pane
`)
	}
	flag.Parse()

	// Resolve API key
	key := *apiKey
	if key == "" {
		key = os.Getenv("FZF_API_KEY")
	}

	// Handle --create-pane
	if *createPane {
		if !repl.TmuxAvailable() {
			fmt.Fprintln(os.Stderr, "error: tmux is not available")
			os.Exit(1)
		}
		var extra []string
		if *paneArgs != "" {
			extra = strings.Split(*paneArgs, ",")
		}
		sockPath, err := repl.CreateFzfPane(*paneSocket, "", extra)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Created fzf pane with socket: %s\n", sockPath)
		fmt.Printf("export FZF_SOCK=%s\n", sockPath)
		return
	}

	// Resolve connection
	client := resolveClient(*socketPath, *tcpAddr, key)
	if client == nil {
		fmt.Fprintln(os.Stderr, "error: no fzf connection found")
		fmt.Fprintln(os.Stderr, "Set FZF_SOCK or FZF_PORT, or use -socket/-tcp flags")
		if repl.TmuxAvailable() {
			fmt.Fprintln(os.Stderr, "Or use -create-pane to start fzf in a tmux pane")
		}
		os.Exit(1)
	}

	// Non-interactive: send action
	if *action != "" {
		if err := repl.RunNonInteractive(client, *action, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Non-interactive: get state
	if *getState {
		data, err := client.GetStateRaw(*limit, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Interactive REPL
	cfg := repl.DefaultConfig()
	cfg.LogLevel = *logLevel
	cfg.MaxItems = *limit

	r := repl.New(client, cfg)
	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func resolveClient(socketFlag, tcpFlag, apiKey string) *repl.Client {
	// Explicit socket flag
	if socketFlag != "" {
		return repl.NewSocketClient(socketFlag, apiKey)
	}

	// Explicit TCP flag
	if tcpFlag != "" {
		return repl.NewTCPClient(tcpFlag, apiKey)
	}

	// Environment: FZF_SOCK
	if sock := os.Getenv("FZF_SOCK"); sock != "" {
		return repl.NewSocketClient(sock, apiKey)
	}

	// Environment: FZF_PORT
	if port := os.Getenv("FZF_PORT"); port != "" {
		return repl.NewTCPClient("localhost:"+port, apiKey)
	}

	return nil
}
