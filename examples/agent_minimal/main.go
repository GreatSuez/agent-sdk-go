package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	agentfw "github.com/PipeOpsHQ/agent-sdk-go/framework/agent"
	providerfactory "github.com/PipeOpsHQ/agent-sdk-go/framework/providers/factory"
)

func main() {
	ctx := context.Background()
	provider, err := providerfactory.FromEnv(ctx)
	if err != nil {
		log.Fatalf("provider setup failed: %v", err)
	}

	prompt := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if prompt == "" {
		prompt = "Explain defense in depth in 4 bullets."
	}

	a, err := agentfw.New(
		provider,
		agentfw.WithSystemPrompt("You are concise, practical, and security-focused."),
		agentfw.WithMaxIterations(4),
		agentfw.WithMaxOutputTokens(500),
	)
	if err != nil {
		log.Fatalf("agent create failed: %v", err)
	}

	out, err := a.Run(ctx, prompt)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
	fmt.Println(out)
}
