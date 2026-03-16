package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

type ESClient struct {
	client *elasticsearch.Client
}

func NewESClient(address, user, password string) (*ESClient, error) {
	// Optimize HTTP Transport for high concurrency
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   32, // Match or exceed maxConcurrency in app.go
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	cfg := elasticsearch.Config{
		Addresses: []string{address},
		Username:  user,
		Password:  password,
		Transport: transport,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating the client: %w", err)
	}

	// Ping the cluster
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting cluster info: %w", err)
	}
	defer res.Body.Close()

	return &ESClient{client: client}, nil
}

func (s *ESClient) GetFields(ctx context.Context, index string) ([]string, error) {
	res, err := s.client.Indices.GetMapping(
		s.client.Indices.GetMapping.WithContext(ctx),
		s.client.Indices.GetMapping.WithIndex(index),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting mapping: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response from ES: %s", res.String())
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode mapping: %w", err)
	}

	// Parsing ES mapping structure: { index_name: { mappings: { properties: { field_name: { ... } } } } }
	idxData, ok := raw[index].(map[string]interface{})
	if !ok {
		// Handle pattern-based index names (if index is a pattern like "logs-*")
		// For simplicity, take the first key if index matches multiple
		for k, v := range raw {
			idxData = v.(map[string]interface{})
			_ = k
			break
		}
	}

	mappings, ok := idxData["mappings"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	properties, ok := mappings["properties"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	var fields []string
	for k := range properties {
		fields = append(fields, k)
	}
	return fields, nil
}
func (s *ESClient) GetInfo() (string, error) {
	res, err := s.client.Info()
	if err != nil {
		return "", fmt.Errorf("error getting cluster info: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("error response from ES: %s", res.String())
	}

	return res.String(), nil
}

// GetVersion returns major and minor version of ES.
// Uses a robust approach to handle various version string formats.
func (s *ESClient) GetVersion() (int, int, error) {
	res, err := s.client.Info()
	if err != nil {
		return 0, 0, fmt.Errorf("ping failed: %w", err)
	}
	defer res.Body.Close()

	var info map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return 0, 0, fmt.Errorf("json decode failed: %w", err)
	}

	version, ok := info["version"].(map[string]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("version block missing in Info() response")
	}

	number, ok := version["number"].(string)
	if !ok {
		return 0, 0, fmt.Errorf("version number string missing")
	}

	var major, minor int
	// Use Sscanf to extract the first two parts of the version string (e.g., "7.10.2" -> 7, 10)
	n, _ := fmt.Sscanf(number, "%d.%d", &major, &minor)
	if n < 1 {
		return 0, 0, fmt.Errorf("invalid version format: %s", number)
	}

	return major, minor, nil
}

func (s *ESClient) GetClient() *elasticsearch.Client {
	return s.client
}
