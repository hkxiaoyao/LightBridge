package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type ControllerClient struct {
	HTTPClient *http.Client
}

type VersionInfo struct {
	Version string `json:"version"`
}

func (c ControllerClient) Version(ctx context.Context, instance Instance, secret string) (*VersionInfo, error) {
	if instance.ControllerPort <= 0 || instance.ControllerPort > 65535 {
		return nil, errors.New("controller port is invalid")
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/version", instance.ControllerPort), nil)
	if err != nil {
		return nil, err
	}
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mihomo controller returned %s", resp.Status)
	}
	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
