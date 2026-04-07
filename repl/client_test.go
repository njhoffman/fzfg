package repl

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// startMockFzf creates a mock fzf HTTP server on a Unix socket.
func startMockFzf(t *testing.T) (string, func()) {
	t.Helper()
	sockPath := filepath.Join(t.TempDir(), "fzf-test.sock")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			state := FzfState{
				Query:      "test",
				Position:   1,
				TotalCount: 42,
				MatchCount: 10,
				Sort:       true,
				Matches: []FzfItem{
					{Index: 0, Text: "hello"},
					{Index: 1, Text: "world"},
				},
			}
			json.NewEncoder(w).Encode(state)
		} else if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "")
		}
	})

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("creating test socket: %v", err)
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	cleanup := func() {
		server.Close()
		listener.Close()
		os.Remove(sockPath)
	}

	return sockPath, cleanup
}

func TestNewSocketClient(t *testing.T) {
	c := NewSocketClient("/tmp/test.sock", "mykey")
	if c.SocketPath != "/tmp/test.sock" {
		t.Errorf("SocketPath = %q, want '/tmp/test.sock'", c.SocketPath)
	}
	if c.APIKey != "mykey" {
		t.Errorf("APIKey = %q, want 'mykey'", c.APIKey)
	}
}

func TestNewTCPClient(t *testing.T) {
	c := NewTCPClient("localhost:6266", "")
	if c.TCPAddr != "localhost:6266" {
		t.Errorf("TCPAddr = %q, want 'localhost:6266'", c.TCPAddr)
	}
}

func TestConnectionInfo(t *testing.T) {
	sc := NewSocketClient("/tmp/test.sock", "")
	if sc.ConnectionInfo() != "unix:/tmp/test.sock" {
		t.Errorf("got %q", sc.ConnectionInfo())
	}

	tc := NewTCPClient("localhost:6266", "")
	if tc.ConnectionInfo() != "tcp:localhost:6266" {
		t.Errorf("got %q", tc.ConnectionInfo())
	}
}

func TestGetState_Mock(t *testing.T) {
	sockPath, cleanup := startMockFzf(t)
	defer cleanup()

	client := NewSocketClient(sockPath, "")
	state, err := client.GetState(100, 0)
	if err != nil {
		t.Fatalf("GetState error: %v", err)
	}

	if state.Query != "test" {
		t.Errorf("Query = %q, want 'test'", state.Query)
	}
	if state.TotalCount != 42 {
		t.Errorf("TotalCount = %d, want 42", state.TotalCount)
	}
	if len(state.Matches) != 2 {
		t.Errorf("len(Matches) = %d, want 2", len(state.Matches))
	}
}

func TestGetStateRaw_Mock(t *testing.T) {
	sockPath, cleanup := startMockFzf(t)
	defer cleanup()

	client := NewSocketClient(sockPath, "")
	data, err := client.GetStateRaw(100, 0)
	if err != nil {
		t.Fatalf("GetStateRaw error: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty raw JSON")
	}

	var state FzfState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Errorf("invalid JSON: %v", err)
	}
}

func TestSendAction_Mock(t *testing.T) {
	sockPath, cleanup := startMockFzf(t)
	defer cleanup()

	client := NewSocketClient(sockPath, "")
	resp, err := client.SendAction("up")
	if err != nil {
		t.Fatalf("SendAction error: %v", err)
	}
	// Mock returns empty string on success
	if resp != "" {
		t.Errorf("expected empty response, got %q", resp)
	}
}

func TestPing_Mock(t *testing.T) {
	sockPath, cleanup := startMockFzf(t)
	defer cleanup()

	client := NewSocketClient(sockPath, "")
	if err := client.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestPing_NoServer(t *testing.T) {
	client := NewSocketClient("/tmp/nonexistent-fzf-test.sock", "")
	if err := client.Ping(); err == nil {
		t.Error("expected Ping to fail with no server")
	}
}
