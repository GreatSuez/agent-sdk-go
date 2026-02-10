package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type CustomHTTPSpec struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Method      string            `json:"method,omitempty"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	TimeoutMS   int               `json:"timeoutMs,omitempty"`
	JSONSchema  map[string]any    `json:"jsonSchema,omitempty"`
}

var (
	customToolMu    sync.RWMutex
	customToolSpecs = map[string]CustomHTTPSpec{}
)

var customToolNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{2,63}$`)

func UpsertCustomHTTPTool(spec CustomHTTPSpec) error {
	normalized, err := normalizeCustomHTTPSpec(spec)
	if err != nil {
		return err
	}

	customToolMu.Lock()
	_, customExists := customToolSpecs[normalized.Name]
	customToolMu.Unlock()

	if ToolExists(normalized.Name) && !customExists {
		return fmt.Errorf("tool %q already exists and is not a runtime custom tool", normalized.Name)
	}

	factory := func() Tool {
		defSchema := normalized.JSONSchema
		if len(defSchema) == 0 {
			defSchema = map[string]any{"type": "object", "additionalProperties": true}
		}
		return NewFuncTool(
			normalized.Name,
			normalized.Description,
			defSchema,
			func(ctx context.Context, args json.RawMessage) (any, error) {
				return executeCustomHTTPTool(ctx, normalized, args)
			},
		)
	}

	if err := UpsertTool(normalized.Name, normalized.Description, factory); err != nil {
		return err
	}

	customToolMu.Lock()
	customToolSpecs[normalized.Name] = normalized
	customToolMu.Unlock()
	return nil
}

func DeleteCustomHTTPTool(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	customToolMu.Lock()
	_, ok := customToolSpecs[name]
	if ok {
		delete(customToolSpecs, name)
	}
	customToolMu.Unlock()
	if !ok {
		return false
	}
	RemoveTool(name)
	return true
}

func ListCustomHTTPTools() []CustomHTTPSpec {
	customToolMu.RLock()
	out := make([]CustomHTTPSpec, 0, len(customToolSpecs))
	for _, spec := range customToolSpecs {
		clone := spec
		if len(spec.Headers) > 0 {
			clone.Headers = map[string]string{}
			for k, v := range spec.Headers {
				clone.Headers[k] = v
			}
		}
		if len(spec.JSONSchema) > 0 {
			clone.JSONSchema = map[string]any{}
			for k, v := range spec.JSONSchema {
				clone.JSONSchema[k] = v
			}
		}
		out = append(out, clone)
	}
	customToolMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func executeCustomHTTPTool(ctx context.Context, spec CustomHTTPSpec, args json.RawMessage) (any, error) {
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "" {
		method = http.MethodPost
	}
	payload := bytes.TrimSpace(args)
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	timeout := spec.TimeoutMS
	if timeout <= 0 {
		timeout = 20000
	}
	if timeout > 120000 {
		timeout = 120000
	}
	if timeout < 1000 {
		timeout = 1000
	}

	requestURL := strings.TrimSpace(spec.URL)
	if method == http.MethodGet {
		requestURL = withQueryFromPayload(requestURL, payload)
	}
	var body io.Reader
	if method != http.MethodGet {
		body = bytes.NewReader(payload)
	}

	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, method, requestURL, body)
	if err != nil {
		return nil, err
	}
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range spec.Headers {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		req.Header.Set(key, strings.TrimSpace(v))
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	headers := map[string]string{}
	for k, values := range resp.Header {
		if len(values) > 0 {
			headers[k] = values[0]
		}
	}

	var parsed any
	if json.Unmarshal(bodyBytes, &parsed) != nil {
		parsed = string(bodyBytes)
	}

	result := map[string]any{
		"status":  resp.StatusCode,
		"headers": headers,
		"body":    parsed,
	}
	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("custom tool endpoint returned %d", resp.StatusCode)
	}
	return result, nil
}

func withQueryFromPayload(rawURL string, payload []byte) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return rawURL
	}
	query := u.Query()
	var obj map[string]any
	if json.Unmarshal(payload, &obj) != nil {
		return rawURL
	}
	for k, v := range obj {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		query.Set(key, fmt.Sprintf("%v", v))
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func normalizeCustomHTTPSpec(spec CustomHTTPSpec) (CustomHTTPSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	if !customToolNamePattern.MatchString(spec.Name) {
		return CustomHTTPSpec{}, fmt.Errorf("invalid custom tool name %q", spec.Name)
	}
	spec.Description = strings.TrimSpace(spec.Description)
	if spec.Description == "" {
		spec.Description = "Runtime custom HTTP tool"
	}
	spec.URL = strings.TrimSpace(spec.URL)
	if spec.URL == "" {
		return CustomHTTPSpec{}, fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(spec.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return CustomHTTPSpec{}, fmt.Errorf("invalid url %q", spec.URL)
	}
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "" {
		method = http.MethodPost
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return CustomHTTPSpec{}, fmt.Errorf("unsupported method %q", method)
	}
	spec.Method = method
	if len(spec.JSONSchema) == 0 {
		spec.JSONSchema = map[string]any{"type": "object", "additionalProperties": true}
	}
	if spec.TimeoutMS < 0 {
		spec.TimeoutMS = 0
	}
	return spec, nil
}
