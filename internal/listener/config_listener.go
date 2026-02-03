package listener

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ConfigItem represents a configuration item being monitored
type ConfigItem struct {
	DataID string
	Group  string
	Tenant string
	MD5    string
}

// ChangeHandler is called when a config change is detected
type ChangeHandler func(dataID, group, tenant string) error

// ConfigListener listens for configuration changes from Nacos
type ConfigListener struct {
	serverAddr  string
	username    string
	password    string
	accessToken string
	httpClient  *http.Client
}

// NewConfigListener creates a new configuration listener
func NewConfigListener(serverAddr, username, password string) *ConfigListener {
	return &ConfigListener{
		serverAddr: serverAddr,
		username:   username,
		password:   password,
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // Longer than long-polling timeout
		},
	}
}

// Login gets access token for authentication
func (l *ConfigListener) Login() error {
	loginURL := fmt.Sprintf("http://%s/nacos/v1/auth/login", l.serverAddr)

	data := url.Values{}
	data.Set("username", l.username)
	data.Set("password", l.password)

	resp, err := l.httpClient.PostForm(loginURL, data)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read login response failed: %w", err)
	}

	// Parse accessToken from response (simple JSON parsing)
	token := extractAccessToken(string(body))
	if token == "" {
		return fmt.Errorf("failed to extract access token from response")
	}

	l.accessToken = token
	return nil
}

// StartListening starts long-polling for configuration changes
func (l *ConfigListener) StartListening(items []ConfigItem, handler ChangeHandler, stopCh <-chan struct{}) error {
	// Keep a map of current items and their MD5
	currentItems := make(map[string]*ConfigItem)
	for i := range items {
		key := fmt.Sprintf("%s_%s_%s", items[i].DataID, items[i].Group, items[i].Tenant)
		currentItems[key] = &items[i]
	}

	// Create a context that cancels when stopCh is closed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stopCh
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Build listening configs string
			listeningConfigs := buildListeningConfigs(items)

			// Call listener API
			changedItems, err := l.longPoll(ctx, listeningConfigs)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				// Log error and retry
				fmt.Printf("Long polling error: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Process changes
			if len(changedItems) > 0 {
				for _, changed := range changedItems {
					// Normalize tenant: if empty, try to find it from our items
					normalizedTenant := changed.Tenant
					if normalizedTenant == "" {
						// Find tenant from currentItems based on dataID and group
						for _, item := range currentItems {
							if item.DataID == changed.DataID && item.Group == changed.Group {
								normalizedTenant = item.Tenant
								break
							}
						}
					}
					key := fmt.Sprintf("%s_%s_%s", changed.DataID, changed.Group, normalizedTenant)

					// Fetch latest config
					content, newMD5, err := l.getConfig(changed.DataID, changed.Group, changed.Tenant)
					if err != nil {
						// Check if it's a 404 error (config deleted)
						if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not exist") {
							// Check if MD5 is already empty (already processed deletion)
							if item, ok := currentItems[key]; ok {
								if item.MD5 == "" {
									// Already deleted and MD5 reset, skip
									continue
								}
								// First time seeing deletion, process it
								// Config was deleted, call handler to handle deletion
								if err := handler(changed.DataID, changed.Group, changed.Tenant); err != nil {
									fmt.Printf("Handler failed for %s/%s: %v\n", changed.DataID, changed.Group, err)
								}
								// Reset MD5 to empty so we can detect if skill is recreated
								item.MD5 = ""
							} else {
								// Item not found in map, this shouldn't happen
								fmt.Printf("Warning: item not found in currentItems: %s\n", key)
							}
							continue
						}
						fmt.Printf("Failed to fetch config %s/%s: %v\n", changed.DataID, changed.Group, err)
						continue
					}

					// Check if MD5 actually changed
					if item, ok := currentItems[key]; ok {
						if item.MD5 == newMD5 {
							// MD5 hasn't changed, skip
							continue
						}
					}

					// Call handler
					if err := handler(changed.DataID, changed.Group, changed.Tenant); err != nil {
						fmt.Printf("Handler failed for %s/%s: %v\n", changed.DataID, changed.Group, err)
						continue
					}

					// Update MD5 only if handler succeeds
					if item, ok := currentItems[key]; ok {
						item.MD5 = newMD5
					}

					_ = content // Suppress unused warning
				}
			}
		}
	}
}

// longPoll performs a long-polling request
func (l *ConfigListener) longPoll(ctx context.Context, listeningConfigs string) ([]ConfigItem, error) {
	listenerURL := fmt.Sprintf("http://%s/nacos/v1/cs/configs/listener", l.serverAddr)

	data := url.Values{}
	data.Set("Listening-Configs", listeningConfigs)

	if l.accessToken != "" {
		listenerURL += "?accessToken=" + l.accessToken
	}

	req, err := http.NewRequestWithContext(ctx, "POST", listenerURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Long-Pulling-Timeout", "30000")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listener returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse changed items
	if len(body) == 0 {
		return nil, nil // No changes
	}

	return parseChangedConfigs(string(body)), nil
}

// getConfig fetches the latest configuration content
func (l *ConfigListener) getConfig(dataID, group, tenant string) (string, string, error) {
	params := url.Values{}
	params.Set("dataId", dataID)
	params.Set("group", group)
	if tenant != "" {
		params.Set("tenant", tenant)
	}
	if l.accessToken != "" {
		params.Set("accessToken", l.accessToken)
	}

	configURL := fmt.Sprintf("http://%s/nacos/v1/cs/configs?%s", l.serverAddr, params.Encode())

	resp, err := l.httpClient.Get(configURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("get config returned status %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// Get MD5 from response header
	contentMD5 := resp.Header.Get("Content-MD5")
	if contentMD5 == "" {
		// Calculate MD5 if not provided
		contentMD5 = calculateMD5(string(content))
	}

	return string(content), contentMD5, nil
}

// buildListeningConfigs builds the Listening-Configs parameter
func buildListeningConfigs(items []ConfigItem) string {
	var parts []string
	for _, item := range items {
		// Format: dataId\x02group\x02md5\x02tenant\x01
		if item.Tenant != "" {
			parts = append(parts, fmt.Sprintf("%s\x02%s\x02%s\x02%s\x01",
				item.DataID, item.Group, item.MD5, item.Tenant))
		} else {
			parts = append(parts, fmt.Sprintf("%s\x02%s\x02%s\x01",
				item.DataID, item.Group, item.MD5))
		}
	}

	result := strings.Join(parts, "")
	return url.QueryEscape(result)
}

// parseChangedConfigs parses the changed config list from response
func parseChangedConfigs(response string) []ConfigItem {
	// URL decode first
	decoded, err := url.QueryUnescape(response)
	if err != nil {
		return nil
	}

	// Split by \x01 (line separator)
	lines := strings.Split(decoded, "\x01")

	var items []ConfigItem
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Split by \x02 (field separator)
		fields := strings.Split(line, "\x02")
		if len(fields) >= 2 {
			item := ConfigItem{
				DataID: fields[0],
				Group:  fields[1],
			}
			if len(fields) >= 4 {
				item.Tenant = fields[3]
			}
			items = append(items, item)
		}
	}

	return items
}

// CalculateMD5 calculates MD5 hash of content (exported for reuse)
func CalculateMD5(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// calculateMD5 is the internal version
func calculateMD5(content string) string {
	return CalculateMD5(content)
}

// extractAccessToken extracts access token from JSON response (simple parsing)
func extractAccessToken(body string) string {
	// Simple JSON parsing for {"accessToken":"xxx",...}
	start := strings.Index(body, `"accessToken":"`)
	if start == -1 {
		return ""
	}
	start += len(`"accessToken":"`)
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		return ""
	}
	return body[start : start+end]
}
