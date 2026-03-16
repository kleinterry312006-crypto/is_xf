package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

type Aggregator struct {
	client *elasticsearch.Client
}

func NewAggregator(client *elasticsearch.Client) *Aggregator {
	return &Aggregator{client: client}
}

// BuildNestedAggregation creates a nested terms aggregation for the given fields
func (a *Aggregator) BuildNestedAggregation(fields []string) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	return a.wrapAgg(fields, 0)
}

func (a *Aggregator) wrapAgg(fields []string, index int) map[string]interface{} {
	if index >= len(fields) {
		return nil
	}

	field := fields[index]
	aggName := fmt.Sprintf("agg_%s", strings.ReplaceAll(field, ".", "_"))
	
	innerAggs := a.wrapAgg(fields, index+1)
	
	terms := map[string]interface{}{
		"field": field,
		"size":  100, // Adjust size as needed
	}

	agg := map[string]interface{}{
		"terms": terms,
	}

	if innerAggs != nil {
		agg["aggs"] = innerAggs
	}

	return map[string]interface{}{
		aggName: agg,
	}
}

// ExecuteAggregation performs the ES search with the constructed aggregations
func (a *Aggregator) ExecuteAggregation(ctx context.Context, index string, aggs map[string]interface{}) (map[string]interface{}, error) {
	query := map[string]interface{}{
		"size": 0,
		"aggs": aggs,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	res, err := a.client.Search(
		a.client.Search.WithContext(ctx),
		a.client.Search.WithIndex(index),
		a.client.Search.WithBody(strings.NewReader(string(body))),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES error: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
