package threads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const BaseURL = "https://graph.threads.net/v1.0"

type Client struct {
	AccessToken string
	HTTPClient  *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		AccessToken: token,
		HTTPClient:  &http.Client{},
	}
}

type MediaContainer struct {
	ID string `json:"id"`
}

func (c *Client) CreateTextPost(text string) (string, error) {
	url := fmt.Sprintf("%s/me/threads", BaseURL)
	payload := map[string]string{
		"media_type": "TEXT",
		"text":       text,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create post: %s", resp.Status)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

// Add more methods for Image, Video, Carousel as documented in SKILL.md
