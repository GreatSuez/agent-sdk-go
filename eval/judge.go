package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/PipeOpsHQ/agent-sdk-go/llm"
	"github.com/PipeOpsHQ/agent-sdk-go/types"
)

type Judge interface {
	Score(ctx context.Context, input JudgeInput) (JudgeResult, error)
}

type JudgeInput struct {
	CaseID         string
	Input          string
	Expected       string
	Output         string
	Rubric         string
	Assertions     []Assertion
	RequiredTools  []string
	ForbiddenTools []string
	UsedTools      []string
}

type JudgeResult struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
}

type LLMJudge struct {
	provider llm.Provider
	model    string
}

func NewLLMJudge(provider llm.Provider) (*LLMJudge, error) {
	if provider == nil {
		return nil, fmt.Errorf("judge provider is required")
	}
	return &LLMJudge{provider: provider}, nil
}

func WithJudgeModel(model string) func(*LLMJudge) {
	return func(j *LLMJudge) {
		if j != nil {
			j.model = strings.TrimSpace(model)
		}
	}
}

func (j *LLMJudge) Score(ctx context.Context, input JudgeInput) (JudgeResult, error) {
	if j == nil || j.provider == nil {
		return JudgeResult{}, fmt.Errorf("judge provider is required")
	}
	promptPayload := map[string]any{
		"caseId":         input.CaseID,
		"input":          input.Input,
		"expected":       input.Expected,
		"output":         input.Output,
		"rubric":         input.Rubric,
		"assertions":     input.Assertions,
		"requiredTools":  input.RequiredTools,
		"forbiddenTools": input.ForbiddenTools,
		"usedTools":      input.UsedTools,
	}
	payload, _ := json.Marshal(promptPayload)

	req := types.Request{
		Model: j.model,
		SystemPrompt: "You are an impartial evaluator. Score responses strictly by rubric and constraints. " +
			"Return only JSON with fields: score (0..1 number), reason (short string).",
		Messages: []types.Message{{Role: types.RoleUser, Content: string(payload)}},
		ResponseSchema: map[string]any{
			"type":     "object",
			"required": []any{"score", "reason"},
			"properties": map[string]any{
				"score":  map[string]any{"type": "number"},
				"reason": map[string]any{"type": "string"},
			},
		},
	}
	resp, err := j.provider.Generate(ctx, req)
	if err != nil {
		return JudgeResult{}, fmt.Errorf("judge generate failed: %w", err)
	}
	result, err := parseJudgeResult(resp.Message.Content)
	if err != nil {
		return JudgeResult{}, err
	}
	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 1 {
		result.Score = 1
	}
	return result, nil
}

func parseJudgeResult(content string) (JudgeResult, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return JudgeResult{}, fmt.Errorf("judge returned empty response")
	}

	var out JudgeResult
	if json.Unmarshal([]byte(trimmed), &out) == nil {
		return out, nil
	}

	re := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
	match := re.FindStringSubmatch(trimmed)
	if len(match) == 2 {
		if err := json.Unmarshal([]byte(match[1]), &out); err == nil {
			return out, nil
		}
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(trimmed[start:end+1]), &out); err == nil {
			return out, nil
		}
	}

	return JudgeResult{}, fmt.Errorf("judge returned invalid JSON")
}
