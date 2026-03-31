package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with nginx-waf-api.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient creates an API client.
func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: timeout},
	}
}

// APIResponse is the standard nginx-waf-api response.
type APIResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ListInfo describes an IP list.
type ListInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Entries int    `json:"entries"`
	ModTime string `json:"mod_time"`
}

// ListDetail describes an IP list with entries.
type ListDetail struct {
	ListInfo
	IPs []string `json:"ips"`
}

// GetLists returns all IP lists.
func (c *Client) GetLists() ([]ListInfo, error) {
	resp, err := c.do("GET", "/api/v1/lists", nil)
	if err != nil {
		return nil, err
	}
	var lists []ListInfo
	if err := json.Unmarshal(resp.Data, &lists); err != nil {
		return nil, fmt.Errorf("decode lists: %w", err)
	}
	return lists, nil
}

// GetList returns a specific list with its entries.
func (c *Client) GetList(name string) (*ListDetail, error) {
	resp, err := c.do("GET", "/api/v1/lists/"+name, nil)
	if err != nil {
		return nil, err
	}
	var detail ListDetail
	if err := json.Unmarshal(resp.Data, &detail); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	return &detail, nil
}

// AddEntry adds an IP/CIDR to a list.
func (c *Client) AddEntry(listName, ip string) error {
	body := fmt.Sprintf(`{"ip":%q}`, ip)
	_, err := c.do("POST", "/api/v1/lists/"+listName+"/entries", strings.NewReader(body))
	return err
}

// RemoveEntry removes an IP/CIDR from a list.
func (c *Client) RemoveEntry(listName, ip string) error {
	_, err := c.do("DELETE", "/api/v1/lists/"+listName+"/entries/"+ip, nil)
	return err
}

// Reload triggers an immediate nginx reload.
func (c *Client) Reload() error {
	_, err := c.do("POST", "/api/v1/reload", nil)
	return err
}

// Health checks the API health endpoint.
func (c *Client) Health() error {
	_, err := c.do("GET", "/health", nil)
	return err
}

func (c *Client) do(method, path string, body io.Reader) (*APIResponse, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(data))
		}
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Status == "error" {
		return nil, fmt.Errorf("api error: %s", apiResp.Message)
	}

	return &apiResp, nil
}
