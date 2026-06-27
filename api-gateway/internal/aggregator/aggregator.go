package aggregator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/qlxion/qlxion-monorepo/api-gateway/internal/proxy"
	"github.com/qlxion/qlxion-monorepo/pkg/response"
)

// Aggregator handles combining responses from multiple services
type Aggregator struct {
	proxy *proxy.ReverseProxy
}

// AggregationRule defines how to aggregate responses
type AggregationRule struct {
	Name      string            `yaml:"name" json:"name"`
	Endpoint  string           `yaml:"endpoint" json:"endpoint"`
	Method    string           `yaml:"method" json:"method"`
	Parts     []AggregationPart `yaml:"parts" json:"parts"`
	MergeType string           `yaml:"merge_type" json:"merge_type"` // merge, replace, array
}

// AggregationPart defines a single part of the aggregation
type AggregationPart struct {
	Key         string            `yaml:"key" json:"key"`
	Service     string            `yaml:"service" json:"service"`
	Path        string            `yaml:"path" json:"path"`
	Method      string            `yaml:"method" json:"method"`
	Headers     map[string]string `yaml:"headers" json:"headers"`
	Required    bool              `yaml:"required" json:"required"`
	Transform   string            `yaml:"transform" json:"transform"` // jq-like transformation
}

// AggregatedResponse represents a combined response
type AggregatedResponse struct {
	Data map[string]interface{} `json:"data"`
}

// NewAggregator creates a new response aggregator
func NewAggregator(p *proxy.ReverseProxy) *Aggregator {
	return &Aggregator{proxy: p}
}

// AggregateWithRequest combines multiple service responses into one
func (a *Aggregator) AggregateWithRequest(w http.ResponseWriter, r *http.Request, parts []proxy.ForwardRequest, mergeType string) {
	results := a.proxy.ForwardMultiple(r.Context(), parts)

	aggregated := make(map[string]interface{})
	hasError := false
	var errorMessages []string

	for _, part := range parts {
		result, ok := results[part.Key]
		if !ok || result.Error != nil {
			hasError = true
			errorMessages = append(errorMessages, fmt.Sprintf("%s: failed", part.Key))
			aggregated[part.Key] = nil
			continue
		}

		if result.StatusCode >= 400 {
			hasError = true
			errorMessages = append(errorMessages, fmt.Sprintf("%s: HTTP %d", part.Key, result.StatusCode))
		}

		var data interface{}
		if err := json.Unmarshal(result.Body, &data); err != nil {
			aggregated[part.Key] = string(result.Body)
		} else {
			aggregated[part.Key] = data
		}
	}

	if hasError {
		resp := response.Success(aggregated)
		resp.Message = "Partial success: " + strings.Join(errorMessages, ", ")
		response.JSON(w, http.StatusOK, resp)
		return
	}

	var finalResponse map[string]interface{}
	switch mergeType {
	case "merge":
		finalResponse = mergeResponses(aggregated)
	case "array":
		finalResponse = map[string]interface{}{"results": aggregated}
	default:
		finalResponse = aggregated
	}

	response.JSONSuccess(w, finalResponse)
}

// FanOutWithRequest aggregates responses and returns array of results
func (a *Aggregator) FanOutWithRequest(w http.ResponseWriter, r *http.Request, parts []proxy.ForwardRequest) {
	results := a.proxy.ForwardMultiple(r.Context(), parts)
	
	var responses []interface{}
	for _, part := range parts {
		result, ok := results[part.Key]
		if !ok || result.Error != nil {
			continue
		}
		
		var data interface{}
		if err := json.Unmarshal(result.Body, &data); err != nil {
			responses = append(responses, string(result.Body))
		} else {
			responses = append(responses, data)
		}
	}
	
	response.JSONSuccess(w, map[string]interface{}{
		"results": responses,
	})
}

// mergeResponses merges nested JSON objects
func mergeResponses(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			for innerKey, innerValue := range v {
				result[innerKey] = innerValue
			}
		default:
			result[key] = value
		}
	}
	
	return result
}
