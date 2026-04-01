package kawaii

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Endpoint string

const (
	EndpointHug   Endpoint = "hug"
	EndpointPat   Endpoint = "pat"
	EndpointPoke  Endpoint = "poke"
	EndpointShrug Endpoint = "shrug"
)

type Options struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
}

type Client struct {
	token string
	base  *url.URL
	http  *http.Client
}

func New(opts Options) (*Client, error) {
	token := strings.TrimSpace(opts.Token)
	if token == "" {
		token = "anonymous"
	}

	baseRaw := strings.TrimSpace(opts.BaseURL)
	if baseRaw == "" {
		baseRaw = "https://kawaii.red"
	}

	base, err := url.Parse(baseRaw)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if strings.ToLower(base.Scheme) != "https" {
		return nil, errors.New("kawaii base url must be https")
	}
	if !isAllowedHost(base.Hostname()) {
		return nil, fmt.Errorf("kawaii base host not allowed: %q", base.Hostname())
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		const defaultTimeout = 3 * time.Second
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		token: token,
		base:  base,
		http:  httpClient,
	}, nil
}

func (c *Client) FetchGIF(ctx context.Context, endpoint Endpoint) (string, error) {
	if c == nil || c.base == nil || c.http == nil {
		return "", errors.New("kawaii client not initialized")
	}
	if !isAllowedEndpoint(endpoint) {
		return "", fmt.Errorf("endpoint not allowed: %q", endpoint)
	}

	u := *c.base
	u.Path = "/api/gif/" + string(endpoint)
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Avoid leaking body content; just return status.
		return "", fmt.Errorf("kawaii api status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var out struct {
		Response string `json:"response"`
	}
	if decodeErr := json.NewDecoder(limited).Decode(&out); decodeErr != nil {
		return "", fmt.Errorf("decode: %w", decodeErr)
	}

	gifURL := strings.TrimSpace(out.Response)
	if !strings.HasPrefix(strings.ToLower(gifURL), "https://") {
		return "", errors.New("kawaii returned non-https url")
	}

	return gifURL, nil
}

func isAllowedEndpoint(e Endpoint) bool {
	switch e {
	case EndpointHug, EndpointPat, EndpointPoke, EndpointShrug:
		return true
	default:
		return false
	}
}

const maxResponseBytes = 64 * 1024

func isAllowedHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "kawaii.red":
		return true
	default:
		return false
	}
}
