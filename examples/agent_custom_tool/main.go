package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	agentfw "github.com/PipeOpsHQ/agent-sdk-go/framework/agent"
	providerfactory "github.com/PipeOpsHQ/agent-sdk-go/framework/providers/factory"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/tools"
)

type riskInput struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
}

type riskOutput struct {
	Score int    `json:"score"`
	Tier  string `json:"tier"`
}

func main() {
	ctx := context.Background()
	provider, err := providerfactory.FromEnv(ctx)
	if err != nil {
		log.Fatalf("provider setup failed: %v", err)
	}

	riskTool := tools.NewFuncTool(
		"calculate_risk_score",
		"Calculate a simple risk score from vulnerability counts.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"critical": map[string]any{"type": "integer"},
				"high":     map[string]any{"type": "integer"},
				"medium":   map[string]any{"type": "integer"},
			},
			"required": []string{"critical", "high", "medium"},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			_ = ctx
			var in riskInput
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, err
			}
			score := (in.Critical * 10) + (in.High * 5) + (in.Medium * 2)
			tier := "low"
			switch {
			case score >= 60:
				tier = "critical"
			case score >= 30:
				tier = "high"
			case score >= 15:
				tier = "medium"
			}
			return riskOutput{Score: score, Tier: tier}, nil
		},
	)

	a, err := agentfw.New(
		provider,
		agentfw.WithSystemPrompt("Use tools when available and return compact security recommendations."),
		agentfw.WithTool(riskTool),
		agentfw.WithMaxIterations(4),
	)
	if err != nil {
		log.Fatalf("agent create failed: %v", err)
	}

	prompt := strings.Join([]string{
		"Use calculate_risk_score with critical=2, high=4, medium=3.",
		"Return: score, tier, and top 3 immediate remediation priorities.",
	}, " ")

	result, err := a.RunDetailed(ctx, prompt)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}

	fmt.Printf("run_id=%s session_id=%s\n\n", result.RunID, result.SessionID)
	fmt.Println(result.Output)
}
