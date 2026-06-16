package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Request struct {
	Type    string      `json:"type"`
	Method  string      `json:"method"`
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

type Response struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type QueryPayload struct {
	Content string `json:"content"`
	Intent  string `json:"intent,omitempty"`
}

type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
}

func NewClient() *Client {
	home, _ := os.UserHomeDir()
	return &Client{
		socketPath: filepath.Join(home, ".vibeauracle", "vibeaura.sock"),
	}
}

func (c *Client) getConn() (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn, nil
	}

	var conn net.Conn
	var err error
	maxRetries := 2
	for i := 0; i < maxRetries; i++ {
		conn, err = net.DialTimeout("unix", c.socketPath, 2*time.Second)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to vibeauracle UDS: %w", err)
	}

	c.conn = conn
	return c.conn, nil
}

func (c *Client) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) call(method string, payload interface{}) (json.RawMessage, error) {
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(60 * time.Second))
	defer conn.SetDeadline(time.Time{})

	reqID := fmt.Sprintf("threader-%d", time.Now().UnixNano())
	req := Request{
		Type:    "request",
		Method:  method,
		ID:      reqID,
		Payload: payload,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		c.closeConn()
		return nil, fmt.Errorf("failed to write to UDS: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	if scanner.Scan() {
		var resp Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		if resp.ID != reqID {
			return nil, fmt.Errorf("response ID mismatch")
		}
		if resp.Type == "error" {
			return nil, fmt.Errorf("vibeauracle error: %s", string(resp.Payload))
		}
		return resp.Payload, nil
	}

	if err := scanner.Err(); err != nil {
		c.closeConn()
		return nil, err
	}

	return nil, fmt.Errorf("no response from vibeauracle")
}

func (c *Client) Query(content string, intent string) (string, error) {
	payload := QueryPayload{
		Content: content,
		Intent:  intent,
	}
	raw, err := c.call("query", payload)
	if err != nil {
		return "", err
	}

	var result struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err == nil && result.Content != "" {
		return result.Content, nil
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil && str != "" {
		return str, nil
	}

	return string(raw), nil
}

func (c *Client) Embed(content string) ([]float64, error) {
	raw, err := c.call("embed", map[string]string{"content": content})
	if err != nil {
		return nil, err
	}

	var result []float64
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) VaultGet(key string) (string, error) {
	raw, err := c.call("vault_get", map[string]string{"key": key})
	if err != nil {
		return "", err
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}

	return result.Value, nil
}

func (c *Client) VaultSet(key, value string) error {
	_, err := c.call("vault_set", map[string]string{"key": key, "value": value})
	return err
}
