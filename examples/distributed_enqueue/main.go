package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/PipeOpsHQ/agent-sdk-go/framework/runtime/distributed"
	"github.com/PipeOpsHQ/agent-sdk-go/framework/runtime/queue/redisstreams"
	statesqlite "github.com/PipeOpsHQ/agent-sdk-go/framework/state/sqlite"
)

func main() {
	ctx := context.Background()
	redisAddr := getenv("AGENT_REDIS_ADDR", "127.0.0.1:6379")
	prefix := getenv("AGENT_RUNTIME_QUEUE_PREFIX", "aiag:queue")
	group := getenv("AGENT_RUNTIME_QUEUE_GROUP", "workers")

	store, err := statesqlite.New("./.ai-agent/examples-distributed-state.db")
	if err != nil {
		log.Fatalf("state store setup failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	attempts, err := distributed.NewSQLiteAttemptStore("./.ai-agent/examples-distributed-attempts.db")
	if err != nil {
		log.Fatalf("attempt store setup failed: %v", err)
	}
	defer func() { _ = attempts.Close() }()

	queue, err := redisstreams.New(
		redisAddr,
		redisstreams.WithPassword(strings.TrimSpace(os.Getenv("AGENT_REDIS_PASSWORD"))),
		redisstreams.WithDB(getenvInt("AGENT_REDIS_DB", 0)),
		redisstreams.WithPrefix(prefix),
		redisstreams.WithGroup(group),
	)
	if err != nil {
		log.Fatalf("queue setup failed: %v", err)
	}
	defer func() { _ = queue.Close() }()

	coord, err := distributed.NewCoordinator(store, attempts, queue, nil, distributed.DistributedConfig{})
	if err != nil {
		log.Fatalf("coordinator setup failed: %v", err)
	}

	input := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if input == "" {
		input = "Investigate auth service token validation failures and DB timeouts"
	}

	res, err := coord.SubmitRun(ctx, distributed.SubmitRequest{
		Input:       input,
		Mode:        "run",
		Workflow:    "basic",
		MaxAttempts: 3,
		Metadata: map[string]any{
			"example": "distributed_enqueue",
		},
	})
	if err != nil {
		log.Fatalf("submit run failed: %v", err)
	}

	stats, err := coord.QueueStats(ctx)
	if err != nil {
		log.Fatalf("queue stats failed: %v", err)
	}

	fmt.Printf("submitted run_id=%s session_id=%s message_id=%s\n", res.RunID, res.SessionID, res.MessageID)
	fmt.Printf("queue stats: stream_length=%d pending=%d dlq_length=%d\n", stats.StreamLength, stats.Pending, stats.DLQLength)
	fmt.Println("next step: start worker(s) with `go run ./cmd/ai-agent-framework worker-start --worker-id=w1`")
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
