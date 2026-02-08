package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	agentfw "github.com/PipeOpsHQ/agent-sdk-go/framework/agent"
	secopsgraph "github.com/PipeOpsHQ/agent-sdk-go/framework/graphs/secops"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/llm"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/observe"
	observesqlite "github.com/PipeOpsHQ/agent-sdk-go/framework/observe/store/sqlite"
	providerfactory "github.com/PipeOpsHQ/agent-sdk-go/framework/providers/factory"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/state"
	statefactory "github.com/PipeOpsHQ/agent-sdk-go/framework/state/factory"
	fwtools "github.com/PipeOpsHQ/agent-sdk-go/framework/tools"
)

const secOpsSystemPrompt = `You are a senior SecOps analyst.

Analyze Trivy findings and redacted logs.
Return compact, actionable output.
For logs, keep to max 3 key issues and max 3 fixes.`

func main() {
	ctx := context.Background()
	input, err := readInput(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	provider, store, observer, closeFn := buildDeps(ctx)
	defer closeFn()

	a, err := buildAgent(provider, store, observer)
	if err != nil {
		log.Fatalf("agent create failed: %v", err)
	}

	exec, err := secopsgraph.NewExecutor(a, secopsgraph.Config{Store: store})
	if err != nil {
		log.Fatalf("secops executor create failed: %v", err)
	}
	exec.SetObserver(observer)

	result, err := exec.Run(ctx, input)
	if err != nil {
		log.Fatalf("secops run failed: %v", err)
	}

	fmt.Printf("run_id=%s session_id=%s\n\n%s\n", result.RunID, result.SessionID, strings.TrimSpace(result.Output))
}

func readInput(args []string) (string, error) {
	if len(args) > 0 {
		path := strings.TrimSpace(args[0])
		if path != "" {
			b, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to read %s: %w", path, err)
			}
			return string(b), nil
		}
	}
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		b, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			return "", fmt.Errorf("failed to read stdin: %w", readErr)
		}
		return string(b), nil
	}
	return "", fmt.Errorf("usage: go run ./framework/examples/secops_sdk <trivy-json-or-log-file> OR cat file | go run ./framework/examples/secops_sdk")
}

func buildDeps(ctx context.Context) (llm.Provider, state.Store, observe.Sink, func()) {
	provider, err := providerfactory.FromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	store, err := statefactory.FromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	observer, closeObserver := buildObserver()
	return provider, store, observer, func() {
		closeObserver()
		_ = store.Close()
	}
}

func buildAgent(provider llm.Provider, store state.Store, observer observe.Sink) (*agentfw.Agent, error) {
	selected, err := fwtools.BuildSelection([]string{"@default"})
	if err != nil {
		return nil, err
	}
	opts := []agentfw.Option{
		agentfw.WithSystemPrompt(secOpsSystemPrompt),
		agentfw.WithStore(store),
		agentfw.WithObserver(observer),
		agentfw.WithMaxIterations(4),
		agentfw.WithMaxOutputTokens(600),
		agentfw.WithRetryPolicy(agentfw.RetryPolicy{MaxAttempts: 2, BaseBackoff: 200 * time.Millisecond, MaxBackoff: 2 * time.Second}),
	}
	for _, t := range selected {
		opts = append(opts, agentfw.WithTool(t))
	}
	return agentfw.New(provider, opts...)
}

func buildObserver() (observe.Sink, func()) {
	traceStore, err := observesqlite.New("./.ai-agent/devui.db")
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
