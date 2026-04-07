package repl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TmuxAvailable checks if tmux is running and accessible.
func TmuxAvailable() bool {
	return os.Getenv("TMUX") != ""
}

// CreateFzfPane creates a new tmux pane running fzf with --listen on the given socket.
// Returns the socket path.
func CreateFzfPane(socketPath, splitDirection string, extraArgs []string) (string, error) {
	if socketPath == "" {
		socketPath = filepath.Join(os.TempDir(), fmt.Sprintf("fzf-%d.sock", os.Getpid()))
	}

	// Build fzf command
	args := []string{"--listen=" + socketPath}
	args = append(args, extraArgs...)
	fzfCmd := "fzf " + strings.Join(args, " ")

	// Create split pane
	tmuxArgs := []string{"split-window"}
	if splitDirection != "" {
		tmuxArgs = append(tmuxArgs, splitDirection)
	} else {
		tmuxArgs = append(tmuxArgs, "-h") // horizontal split by default
	}
	tmuxArgs = append(tmuxArgs, fzfCmd)

	cmd := exec.Command("tmux", tmuxArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("creating tmux pane: %w", err)
	}

	return socketPath, nil
}

// ListTmuxSessions returns available tmux sessions.
func ListTmuxSessions() ([]string, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

// ListTmuxWindows returns windows for a given session.
func ListTmuxWindows(session string) ([]string, error) {
	out, err := exec.Command("tmux", "list-windows", "-t", session, "-F",
		"#{window_index}: #{window_name}").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}
