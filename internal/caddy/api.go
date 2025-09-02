package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
}

func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

func (c *Client) UpdateConfig(ctx context.Context, config *CaddyConfig) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	url := fmt.Sprintf("%s/config/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(configBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Updating Caddy configuration", "url", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Successfully updated Caddy configuration")
	return nil
}

func (c *Client) GetConfig(ctx context.Context) (*CaddyConfig, error) {
	url := fmt.Sprintf("%s/config/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("caddy API returned status %d: %s", resp.StatusCode, string(body))
	}

	var config CaddyConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

func (c *Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/config/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("caddy API health check failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) WaitForReady(ctx context.Context, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := c.Health(ctx); err == nil {
			c.logger.Info("Caddy API is ready")
			return nil
		}

		if i < maxRetries-1 {
			c.logger.Debug("Waiting for Caddy API to be ready", "retry", i+1, "max_retries", maxRetries)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
				continue
			}
		}
	}

	return fmt.Errorf("caddy API not ready after %d retries", maxRetries)
}
