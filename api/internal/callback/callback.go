package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	client *http.Client
}

func New(timeout time.Duration) *Client {
	return &Client{client: &http.Client{Timeout: timeout}}
}

func (c *Client) Post(ctx context.Context, url string, payload interface{}) error {
	if strings.TrimSpace(url) == "" {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback failed status=%d", resp.StatusCode)
	}
	return nil
}
