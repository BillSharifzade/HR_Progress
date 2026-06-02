package onef

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client polls the 1F user feed.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Configured reports whether the client has a base URL set; when false,
// the scheduler stays idle and manual sync returns an error.
func (c *Client) Configured() bool { return c.baseURL != "" }

// FetchUsers calls GET /app/v1.2/api/publications/action/get1FUsers and
// returns the decoded list. 1F may return either a bare array or an
// object with a top-level "data" field; we tolerate both.
func (c *Client) FetchUsers(ctx context.Context) ([]OneFUser, error) {
	if !c.Configured() {
		return nil, errors.New("1F base URL not configured")
	}
	url := c.baseURL + "/app/v1.2/api/publications/action/get1FUsers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("1F GET: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024)) // 32 MiB cap
	if err != nil {
		return nil, fmt.Errorf("1F read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("1F HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	trimmed := strings.TrimLeft(string(body), " \t\r\n")
	if strings.HasPrefix(trimmed, "[") {
		var users []OneFUser
		if err := json.Unmarshal(body, &users); err != nil {
			return nil, fmt.Errorf("1F decode array: %w", err)
		}
		return users, nil
	}

	// Object wrapper variant: {"data":[...]} or {"users":[...]}.
	var wrapper struct {
		Data  []OneFUser `json:"data"`
		Users []OneFUser `json:"users"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("1F decode object: %w", err)
	}
	if len(wrapper.Data) > 0 {
		return wrapper.Data, nil
	}
	return wrapper.Users, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
