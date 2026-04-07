package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Client connects to an fzf instance via Unix socket or TCP.
type Client struct {
	SocketPath string
	TCPAddr    string
	APIKey     string
	httpClient *http.Client
	baseURL    string
}

// FzfState represents the JSON response from a GET request to fzf.
type FzfState struct {
	Reading    bool      `json:"reading"`
	Progress   int       `json:"progress"`
	Query      string    `json:"query"`
	Position   int       `json:"position"`
	Sort       bool      `json:"sort"`
	TotalCount int       `json:"totalCount"`
	MatchCount int       `json:"matchCount"`
	Current    *FzfItem  `json:"current"`
	Matches    []FzfItem `json:"matches"`
	Selected   []FzfItem `json:"selected"`
}

// FzfItem represents a single item in fzf's results.
type FzfItem struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

// NewSocketClient creates a client connected via Unix socket.
func NewSocketClient(socketPath, apiKey string) *Client {
	c := &Client{
		SocketPath: socketPath,
		APIKey:     apiKey,
		baseURL:    "http://fzf",
	}
	c.httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 5*time.Second)
			},
		},
		Timeout: 10 * time.Second,
	}
	return c
}

// NewTCPClient creates a client connected via TCP.
func NewTCPClient(addr, apiKey string) *Client {
	c := &Client{
		TCPAddr: addr,
		APIKey:  apiKey,
		baseURL: "http://" + addr,
	}
	c.httpClient = &http.Client{Timeout: 10 * time.Second}
	return c
}

// GetState fetches the current fzf state via GET.
func (c *Client) GetState(limit, offset int) (*FzfState, error) {
	url := fmt.Sprintf("%s/?limit=%d&offset=%d", c.baseURL, limit, offset)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to fzf: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fzf returned %d: %s", resp.StatusCode, string(body))
	}

	var state FzfState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &state, nil
}

// GetStateRaw fetches the raw JSON response from fzf.
func (c *Client) GetStateRaw(limit, offset int) ([]byte, error) {
	url := fmt.Sprintf("%s/?limit=%d&offset=%d", c.baseURL, limit, offset)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to fzf: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// SendAction sends an action string to fzf via POST.
func (c *Client) SendAction(action string) (string, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/", strings.NewReader(action))
	if err != nil {
		return "", err
	}
	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending action: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fzf returned %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

// Ping tests the connection to fzf.
func (c *Client) Ping() error {
	_, err := c.GetState(1, 0)
	return err
}

// ConnectionInfo returns a description of the connection.
func (c *Client) ConnectionInfo() string {
	if c.SocketPath != "" {
		return fmt.Sprintf("unix:%s", c.SocketPath)
	}
	return fmt.Sprintf("tcp:%s", c.TCPAddr)
}

func (c *Client) setAuthHeader(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("x-api-key", c.APIKey)
	}
}
