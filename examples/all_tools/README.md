# All-Tools DevOps Agent

A full-stack DevOps agent with access to **all 33 built-in tools** across every bundle.

## Quick Start

```bash
# Run a task
go run . "generate a UUID and base64-encode it"

# Launch DevUI
go run . ui

# DevUI on custom port
go run . ui --ui-addr=:9090
```

## Available Tools (33)

| Bundle | Tools |
|--------|-------|
| `@default` | calculator, json_parser, base64_codec, timestamp_converter, uuid_generator, url_parser, regex_matcher, text_processor |
| `@security` | secret_redactor, hash_generator |
| `@code` | git_repo, code_search, diff_generator |
| `@network` | http_client, web_scraper, curl, dns_lookup, network_utils |
| `@system` | shell_command, file_system, env_vars, tmpdir, process_manager, disk_usage, system_info, log_viewer, archive |
| `@memory` | memory_store |
| `@container` | docker, docker_compose |
| `@kubernetes` | kubectl, k3s |
| `@scheduling` | cron_manager |
| `@linux` | curl, dns_lookup, network_utils, process_manager, disk_usage, system_info, archive, log_viewer |

## Example Prompts

```bash
# Encoding & utilities
go run . "generate a UUID, hash it with SHA256, and base64-encode the result"

# Code analysis
go run . "clone https://github.com/golang/go and find all TODO comments"

# HTTP & web
go run . "GET https://httpbin.org/json and extract the title field"

# File system
go run . "list all Go files in the current directory and count lines of code"

# Security
go run . "scan this for secrets: api_key=sk-abc123 token=ghp_xxxx"

# Docker
go run . "list running docker containers and their resource usage"

# Multi-step
go run . "check if port 8080 is responding, parse the JSON response, and store the status in memory"
```
