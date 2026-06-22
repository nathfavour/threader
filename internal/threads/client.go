package threads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	// Step 1: Create Media Container
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
		return "", fmt.Errorf("failed to create container: %s", resp.Status)
	}

	var container struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
		return "", err
	}

	// Step 2: Publish the Container
	return c.PublishContainer(container.ID)
}

func (c *Client) CreateImageContainer(imageURL, text string) (string, error) {
	url := fmt.Sprintf("%s/me/threads", BaseURL)
	payload := map[string]string{
		"media_type": "IMAGE",
		"image_url":  imageURL,
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
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("failed to create image container (status %d): %v", resp.StatusCode, errResp)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (c *Client) PublishContainer(containerID string) (string, error) {
	url := fmt.Sprintf("%s/me/threads_publish", BaseURL)
	payload := map[string]string{
		"creation_id": containerID,
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
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("failed to publish container (status %d): %v", resp.StatusCode, errResp)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

type Post struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Username  string `json:"username"`
	Permalink string `json:"permalink"`
	Timestamp string `json:"timestamp"`
}

func (c *Client) SearchPosts(query string) ([]Post, error) {
	apiURL := fmt.Sprintf("%s/keyword_search?q=%s&fields=id,text,username,permalink,timestamp", BaseURL, url.QueryEscape(query))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("search failed (status %d): %v", resp.StatusCode, errResp)
	}

	var result struct {
		Data []Post `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

func (c *Client) CreateReply(text string, replyToID string) (string, error) {
	apiURL := fmt.Sprintf("%s/me/threads", BaseURL)
	payload := map[string]string{
		"media_type":  "TEXT",
		"text":        text,
		"reply_to_id": replyToID,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("failed to create reply container (status %d): %v", resp.StatusCode, errResp)
	}

	var container struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
		return "", err
	}

	return c.PublishContainer(container.ID)
}

func (c *Client) CreateImageReply(imageURL, text, replyToID string) (string, error) {
	apiURL := fmt.Sprintf("%s/me/threads", BaseURL)
	payload := map[string]string{
		"media_type":  "IMAGE",
		"image_url":   imageURL,
		"text":        text,
		"reply_to_id": replyToID,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("failed to create image reply container (status %d): %v", resp.StatusCode, errResp)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return c.PublishContainer(result.ID)
}
