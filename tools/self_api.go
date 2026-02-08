package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type selfAPIArgs struct {
	Method   string         `json:"method"`
	Path     string         `json:"path"`
	Body     map[string]any `json:"body,omitempty"`
	QueryStr string         `json:"query,omitempty"`
}

// NewSelfAPI creates a tool that lets the agent call its own DevUI API.
// baseURL is the server's listen address (e.g. "http://127.0.0.1:7070").
func NewSelfAPI(baseURL string) Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"method": map[string]any{
				"type":        "string",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
				"description": "HTTP method.",
			},
			"path": map[string]any{
				"type": "string",
				"description": `API path to call. Available endpoints include:
- GET  /api/v1/flows              — List registered flows
- GET  /api/v1/reflect            — Discover all actions (flows, tools, workflows, skills)
- POST /api/v1/actions/run        — Run an action by key (e.g. {"key":"/flow/code-reviewer","input":{"input":"review this"}})
- GET  /api/v1/runs               — List recent runs
- GET  /api/v1/runs/{id}          — Get run details
- GET  /api/v1/skills             — List installed skills
- POST /api/v1/skills             — Install skill from GitHub (body: {"repoUrl":"owner/repo"})
- DELETE /api/v1/skills/{name}    — Remove a skill
- GET  /api/v1/cron/jobs          — List scheduled jobs
- POST /api/v1/cron/jobs          — Create cron job (body: {"name","cronExpr","input",...})
- DELETE /api/v1/cron/jobs/{name} — Remove a cron job
- POST /api/v1/cron/jobs/{name}/trigger — Trigger a cron job now
- GET  /api/v1/tools/catalog      — List available tools
- GET  /api/v1/tools/registry     — Tool registry with schemas
- GET  /api/v1/workflows          — List workflow bindings
- GET  /api/v1/workflows/registry — Workflow registry
- POST /api/v1/playground/run     — Run playground (body: {"input","flow","workflow","tools",...})
- GET  /api/v1/metrics/summary    — Get metrics summary
- GET  /api/v1/runtime/workers    — List runtime workers
- GET  /api/v1/runtime/queues     — List queues
- GET  /api/v1/audit/logs         — View audit logs
- POST /api/v1/commands/execute   — Execute CLI command`,
			},
			"body": map[string]any{
				"type":        "object",
				"description": "JSON request body for POST/PUT/PATCH requests.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Query string (without leading ?), e.g. 'limit=10&status=completed'.",
			},
		},
		"required": []string{"method", "path"},
	}

	client := &http.Client{Timeout: 60 * time.Second}

	return NewFuncTool(
		"self_api",
		"Call the agent's own DevUI API to manage cron jobs, skills, flows, runs, tools, workflows, runtime, and more. The agent can introspect and control itself.",
		schema,
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var in selfAPIArgs
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, fmt.Errorf("invalid self_api args: %w", err)
			}

			if in.Path == "" {
				return nil, fmt.Errorf("path is required")
			}
			if !strings.HasPrefix(in.Path, "/") {
				in.Path = "/" + in.Path
			}

			url := strings.TrimRight(baseURL, "/") + in.Path
			if in.QueryStr != "" {
				url += "?" + in.QueryStr
			}

			var bodyReader io.Reader
			if in.Body != nil && (in.Method == "POST" || in.Method == "PUT" || in.Method == "PATCH") {
				b, err := json.Marshal(in.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal body: %w", err)
				}
				bodyReader = bytes.NewReader(b)
			}

			req, err := http.NewRequestWithContext(ctx, in.Method, url, bodyReader)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}
			if bodyReader != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("API call failed: %w", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512KB limit
			if err != nil {
				return nil, fmt.Errorf("failed to read response: %w", err)
			}

			// Try to parse as JSON for clean output
			var jsonResp any
			if err := json.Unmarshal(respBody, &jsonResp); err == nil {
				return map[string]any{
					"status":     resp.StatusCode,
					"statusText": resp.Status,
					"body":       jsonResp,
				}, nil
			}

			// Return raw text if not JSON
			return map[string]any{
				"status":     resp.StatusCode,
				"statusText": resp.Status,
				"body":       string(respBody),
			}, nil
		},
	)
}
