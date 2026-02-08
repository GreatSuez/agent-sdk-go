package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	agentfw "github.com/PipeOpsHQ/agent-sdk-go/framework/agent"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/observe"
	observesqlite "github.com/PipeOpsHQ/agent-sdk-go/framework/observe/store/sqlite"
	providerfactory "github.com/PipeOpsHQ/agent-sdk-go/framework/providers/factory"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/state/factory"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/tools"
)

// This example demonstrates using different prompt templates for different use cases.
// Each template creates different agent behaviors suitable for specific roles.

func main() {
	if len(os.Args) < 2 {
		printPromptExamplesUsage()
		os.Exit(1)
	}

	role := strings.ToLower(strings.TrimSpace(os.Args[1]))
	input := "Sample analysis request for testing"
	if len(os.Args) > 2 {
		input = strings.Join(os.Args[2:], " ")
	}

	ctx := context.Background()

	// Setup provider and tools
	provider, err := providerfactory.FromEnv(ctx)
	if err != nil {
		log.Fatalf("provider setup failed: %v", err)
	}

	store, err := factory.FromEnv(ctx)
	if err != nil {
		log.Fatalf("state store setup failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	observer, closeObserver := buildObserver()
	defer closeObserver()

	selectedTools, err := tools.BuildSelection([]string{"@default"})
	if err != nil {
		log.Fatalf("tool selection failed: %v", err)
	}

	// Build agent with selected prompt template
	agent, prompt, err := buildAgentWithTemplate(provider, store, observer, selectedTools, role)
	if err != nil {
		log.Fatalf("agent creation failed: %v", err)
	}

	fmt.Printf("Using %q prompt template\n\n", role)
	fmt.Printf("System Prompt:\n%s\n\n", prompt)
	fmt.Printf("Input: %s\n\n", input)
	fmt.Println("Agent Response:")
	fmt.Println(strings.Repeat("-", 60))

	// Run agent with the input
	result, err := agent.RunDetailed(ctx, input)
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Println(result.Output)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("\nRun ID: %s\nSession ID: %s\n", result.RunID, result.SessionID)
}

func buildAgentWithTemplate(
	provider interface {
		Name() string
	},
	store interface{},
	observer observe.Sink,
	tools []interface{},
	templateName string,
) (*agentfw.Agent, string, error) {
	// Map template names to prompts (simplified version of the prompts.go templates)
	templates := map[string]string{
		"default": "You are a practical AI assistant. Be concise, accurate, and actionable.",
		"analyst": `You are an expert analyst. Your role is to:
- Investigate and understand problems systematically
- Synthesize data into clear, actionable insights
- Support findings with evidence and reasoning
- Provide structured reports with findings, analysis, and recommendations
- Ask clarifying questions when information is ambiguous`,
		"engineer": `You are a senior engineer. Your role is to:
- Design and implement technical solutions
- Prioritize code quality, maintainability, and performance
- Consider edge cases and error handling
- Provide clear technical explanations
- Suggest improvements and best practices
- Use available tools to diagnose and resolve issues`,
		"specialist": `You are a subject matter expert. Your role is to:
- Apply deep domain knowledge to solve complex problems
- Provide authoritative guidance based on best practices
- Explain concepts clearly for different audiences
- Identify risks and recommend mitigations
- Stay focused on the domain's specific requirements`,
		"assistant": `You are a helpful AI assistant. Your role is to:
- Understand user needs clearly before responding
- Provide accurate, complete information
- Break complex tasks into manageable steps
- Use available tools to accomplish goals efficiently
- Follow up to ensure the user is satisfied`,
		"reasoning": `You are a careful reasoner. Your role is to:
- Think through problems step-by-step
- Consider multiple perspectives and approaches
- Identify assumptions and validate them
- Break complex problems into components
- Explain your reasoning clearly
- Revise conclusions if new evidence appears`,
	}

	prompt, ok := templates[templateName]
	if !ok {
		prompt = templates["default"]
	}

	opts := []agentfw.Option{
		agentfw.WithSystemPrompt(prompt),
		agentfw.WithObserver(observer),
		agentfw.WithStore(store.(interface{ Close() error })),
	}

	// Add tools to agent options
	for _, t := range tools {
		if tool, ok := t.(interface{ Definition() interface{} }); ok {
			opts = append(opts, agentfw.WithTool(tool))
		}
	}

	agent, err := agentfw.New(provider.(interface{ Name() string }), opts...)
	return agent, prompt, err
}

func buildObserver() (observe.Sink, func()) {
	dbPath := "./.ai-agent/prompts-example.db"
	traceStore, err := observesqlite.New(dbPath)
	if err != nil {
		return observe.NoopSink{}, func() {}
	}
	async := observe.NewAsyncSink(observe.SinkFunc(func(ctx context.Context, event observe.Event) error {
		return traceStore.SaveEvent(ctx, event)
	}), 256)
	return async, func() {
		async.Close()
		_ = traceStore.Close()
	}
}

func printPromptExamplesUsage() {
	fmt.Println("Prompt Templates Example")
	fmt.Println("Usage: go run ./examples/prompt_templates/main.go <template-name> [input]")
	fmt.Println()
	fmt.Println("Available templates:")
	fmt.Println("  default     - Generic practical AI assistant (minimal, fast)")
	fmt.Println("  analyst     - Data-driven analyst focused on investigation")
	fmt.Println("  engineer    - Technical engineer focused on implementation")
	fmt.Println("  specialist  - Domain specialist with deep expertise")
	fmt.Println("  assistant   - Helpful assistant focused on user support")
	fmt.Println("  reasoning   - Careful reasoner focused on thorough analysis")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./examples/prompt_templates/main.go analyst \"analyze this data\"")
	fmt.Println("  go run ./examples/prompt_templates/main.go engineer \"fix this code\"")
	fmt.Println("  go run ./examples/prompt_templates/main.go reasoning \"think through this\"")
}
