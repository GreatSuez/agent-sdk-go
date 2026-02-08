# SDK Examples

This folder contains runnable examples for common SDK use cases.

## 1) Minimal Agent
Path: `framework/examples/agent_minimal`

Use case:
- Smallest possible runtime setup using provider factory + `agent.Run`.

Run:
```bash
go run ./framework/examples/agent_minimal "Explain least privilege in 3 bullets"
```

## 2) Agent with Custom Tool
Path: `framework/examples/agent_custom_tool`

Use case:
- Add a custom business tool (`calculate_risk_score`) and let the model invoke it.

Run:
```bash
go run ./framework/examples/agent_custom_tool
```

## 3) Graph + Resume (No LLM Required)
Path: `framework/examples/graph_resume`

Use case:
- Build a static graph with deterministic nodes.
- Persist checkpoints to SQLite.
- Resume a completed run by `run_id`.

Run:
```bash
go run ./framework/examples/graph_resume "critical findings in checkout service"
```

## 4) Distributed Enqueue
Path: `framework/examples/distributed_enqueue`

Use case:
- Submit a run into Redis Streams via distributed coordinator.
- Inspect queue stats.

Prerequisite:
- Redis running and reachable via `AGENT_REDIS_ADDR`.

Run:
```bash
go run ./framework/examples/distributed_enqueue "Investigate payment API timeout spikes"
```

## 5) SecOps Workflow via SDK
Path: `framework/examples/secops_sdk`

Use case:
- Run Trivy report or logs through `secops-static` graph using SDK runtime.

Run with file:
```bash
go run ./framework/examples/secops_sdk sample-data/trivy-report.json
go run ./framework/examples/secops_sdk sample-data/app.log
```

Run with stdin:
```bash
cat sample-data/app.log | go run ./framework/examples/secops_sdk
```

## 6) Quickstart (Hybrid Store + Graph)
Path: `framework/examples/sdk_quickstart`

Use case:
- End-to-end quickstart: provider, store, observer, single run, and graph run.

Run:
```bash
go run ./framework/examples/sdk_quickstart
```

## Environment setup
Pick one template from repo root and export vars:
- `.env.local.example`
- `.env.ollama.example`
- `.env.gemini.example`
- `.env.openai.example`
- `.env.azureopenai.example`
- `.env.distributed.example`
