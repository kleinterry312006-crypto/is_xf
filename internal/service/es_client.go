package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

type ESClient struct {
	client *elasticsearch.Client
}

func NewESClient(address string) (*ESClient, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{address},
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

	// Handle standard or pattern index names
	var idxData map[string]interface{}
	if val, ok := raw[index].(map[string]interface{}); ok {
		idxData = val
	} else {
		for _, v := range raw {
			idxData = v.(map[string]interface{})
			break
		}
	}

	if idxData == nil {
		return nil, nil
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

func (s *ESClient) GetClient() *elasticsearch.Client {
	return s.client
}
