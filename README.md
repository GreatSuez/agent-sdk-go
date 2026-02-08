# PipeOps Agent Framework

Core Go framework for building agentic applications with:

- provider-agnostic agent runtime
- static graph orchestration + resume
- SQLite/Redis/Hybrid state backends
- observability sink
- tool registry and bundles

## Quick Start

From repository root:

```bash
go run ./framework run -- "Explain zero trust in 3 bullets"
```

Graph run:

```bash
go run ./framework graph-run --workflow=basic -- "Summarize this alert"
```

Resume graph run:

```bash
go run ./framework graph-resume <run-id>
```

List sessions/runs:

```bash
go run ./framework sessions
```

## Required Provider Environment

Example with Ollama:

```bash
export AGENT_PROVIDER=ollama
export OLLAMA_BASE_URL=http://127.0.0.1:11434
export OLLAMA_MODEL=llama3.1:8b
```

Or Gemini:

```bash
export AGENT_PROVIDER=gemini
export GEMINI_API_KEY=your_key
```

## Common Environment

- `AGENT_STATE_BACKEND=sqlite|redis|hybrid` (default `sqlite`)
- `AGENT_SQLITE_PATH=./.ai-agent/state.db`
- `AGENT_DEVUI_DB_PATH=./.ai-agent/devui.db`
- `AGENT_TOOLS=@default`
- `AGENT_SYSTEM_PROMPT=...`
- `AGENT_OBSERVE_ENABLED=true|false`
