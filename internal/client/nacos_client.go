package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
)

// NacosClient represents a Nacos API client
type NacosClient struct {
	ServerAddr  string
	Namespace   string
	Username    string
	Password    string
	AccessToken string
	httpClient  *resty.Client
}

// Config represents a Nacos configuration
type Config struct {
	DataID    string `json:"dataId"`
	Group     string `json:"group"`
	GroupName string `json:"groupName"`
	Content   string `json:"content"`
	Type      string `json:"type"`
}

// ConfigListResponse represents the response of list configs API
type ConfigListResponse struct {
	TotalCount     int      `json:"totalCount"`
	PageNumber     int      `json:"pageNumber"`
	PagesAvailable int      `json:"pagesAvailable"`
	PageItems      []Config `json:"pageItems"`
}

// V3Response represents the v3 API response wrapper
type V3Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// NewNacosClient creates a new Nacos client
func NewNacosClient(serverAddr, namespace, username, password string) *NacosClient {
	// Convert empty namespace to "public"
	if namespace == "" {
		namespace = "public"
	}

	client := &NacosClient{
		ServerAddr: serverAddr,
		Namespace:  namespace,
		Username:   username,
		Password:   password,
		httpClient: resty.New(),
	}

	// Login to get access token
	client.login()

	return client
}

// login authenticates with Nacos and gets an access token
func (c *NacosClient) login() error {
	resp, err := c.httpClient.R().
		SetFormData(map[string]string{
			"username": c.Username,
			"password": c.Password,
		}).
		Post(fmt.Sprintf("http://%s/nacos/v1/auth/login", c.ServerAddr))

	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if resp.StatusCode() == 200 {
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &result); err != nil {
			return err
		}
		if token, ok := result["accessToken"].(string); ok {
			c.AccessToken = token
		}
	}

	return nil
}

// ListConfigs retrieves a list of configurations
func (c *NacosClient) ListConfigs(dataID, groupName, namespaceID string, pageNo, pageSize int) (*ConfigListResponse, error) {
	// Use provided namespace or default to client's namespace
	ns := namespaceID
	if ns == "" {
		ns = c.Namespace
	}

	// Build parameters
	params := url.Values{}

	// Enable fuzzy search only if wildcard is present
	if strings.Contains(dataID, "*") || strings.Contains(groupName, "*") {
		params.Set("search", "blur")
	} else {
		params.Set("search", "accurate")
	}

	params.Set("dataId", dataID)
	params.Set("groupName", groupName)
	params.Set("pageNo", fmt.Sprintf("%d", pageNo))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if ns != "" {
		params.Set("namespaceId", ns)
	}

	if c.AccessToken != "" {
		params.Set("accessToken", c.AccessToken)
	}

	// Try v3 API first
	v3URL := fmt.Sprintf("http://%s/nacos/v3/admin/cs/config/list", c.ServerAddr)
	resp, err := c.httpClient.R().
		SetQueryString(params.Encode()).
		Get(v3URL)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode() == 200 {
		var v3Resp V3Response
		if err := json.Unmarshal(resp.Body(), &v3Resp); err != nil {
			return nil, err
		}

		if v3Resp.Code == 0 {
			var configList ConfigListResponse
			if err := json.Unmarshal(v3Resp.Data, &configList); err != nil {
				return nil, err
			}
			return &configList, nil
		}
	}

	// Fallback to v1 API if v3 fails
	if resp.StatusCode() == 404 {
		return c.listConfigsV1(dataID, groupName, ns, pageNo, pageSize)
	}

	return nil, fmt.Errorf("list configs failed: status=%d, body=%s", resp.StatusCode(), string(resp.Body()))
}

// listConfigsV1 retrieves configs using v1 API
func (c *NacosClient) listConfigsV1(dataID, groupName, namespace string, pageNo, pageSize int) (*ConfigListResponse, error) {
	params := url.Values{}

	// Enable fuzzy search only if wildcard is present
	if strings.Contains(dataID, "*") || strings.Contains(groupName, "*") {
		params.Set("search", "blur")
	} else {
		params.Set("search", "accurate")
	}

	params.Set("dataId", dataID)
	params.Set("group", groupName) // v1 uses 'group' instead of 'groupName'
	params.Set("pageNo", fmt.Sprintf("%d", pageNo))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	if namespace != "" {
		params.Set("tenant", namespace) // v1 uses 'tenant' instead of 'namespaceId'
	}

	if c.AccessToken != "" {
		params.Set("accessToken", c.AccessToken)
	}

	v1URL := fmt.Sprintf("http://%s/nacos/v1/cs/configs", c.ServerAddr)
	resp, err := c.httpClient.R().
		SetQueryString(params.Encode()).
		Get(v1URL)

	if err != nil {
		return nil, fmt.Errorf("v1 request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("v1 list configs failed: status=%d", resp.StatusCode())
	}

	var configList ConfigListResponse
	if err := json.Unmarshal(resp.Body(), &configList); err != nil {
		return nil, err
	}

	return &configList, nil
}

// GetConfig retrieves a specific configuration
func (c *NacosClient) GetConfig(dataID, group string) (string, error) {
	params := url.Values{}
	params.Set("dataId", dataID)
	params.Set("group", group)

	if c.Namespace != "" {
		params.Set("tenant", c.Namespace)
	}

	if c.AccessToken != "" {
		params.Set("accessToken", c.AccessToken)
	}

	url := fmt.Sprintf("http://%s/nacos/v1/cs/configs", c.ServerAddr)
	resp, err := c.httpClient.R().
		SetQueryString(params.Encode()).
		Get(url)

	if err != nil {
		return "", fmt.Errorf("get config failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("get config failed: status=%d", resp.StatusCode())
	}

	return string(resp.Body()), nil
}

// PublishConfig publishes a configuration
func (c *NacosClient) PublishConfig(dataID, group, content string) error {
	params := map[string]string{
		"dataId":  dataID,
		"group":   group,
		"content": content,
	}

	if c.Namespace != "" {
		params["tenant"] = c.Namespace
	}

	if c.AccessToken != "" {
		params["accessToken"] = c.AccessToken
	}

	url := fmt.Sprintf("http://%s/nacos/v1/cs/configs", c.ServerAddr)
	resp, err := c.httpClient.R().
		SetFormData(params).
		Post(url)

	if err != nil {
		return fmt.Errorf("publish config failed: %w", err)
	}

	if resp.StatusCode() != 200 || string(resp.Body()) != "true" {
		return fmt.Errorf("publish config failed: status=%d, body=%s", resp.StatusCode(), string(resp.Body()))
	}

	return nil
}
