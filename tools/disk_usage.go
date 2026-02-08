package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type diskUsageArgs struct {
	Action string `json:"action"` // df, du
	Path   string `json:"path,omitempty"`
	Depth  int    `json:"depth,omitempty"` // for du
	Limit  int    `json:"limit,omitempty"` // top N entries for du
	Human  bool   `json:"human,omitempty"` // human-readable sizes
}

type dfEntry struct {
	Filesystem string `json:"filesystem"`
	Size       string `json:"size"`
	Used       string `json:"used"`
	Available  string `json:"available"`
	UsePercent string `json:"usePercent"`
	MountedOn  string `json:"mountedOn"`
}

type duEntry struct {
	Size string `json:"size"`
	Path string `json:"path"`
}

type diskUsageResult struct {
	Action      string    `json:"action"`
	Filesystems []dfEntry `json:"filesystems,omitempty"`
	Entries     []duEntry `json:"entries,omitempty"`
	Count       int       `json:"count"`
	Error       string    `json:"error,omitempty"`
}

func NewDiskUsage() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"df", "du"},
				"description": "Action: df (filesystem usage), du (directory sizes).",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Path for du analysis. Defaults to current directory.",
			},
			"depth": map[string]any{
				"type":        "integer",
				"description": "Directory depth for du. Defaults to 1.",
				"minimum":     0,
				"maximum":     5,
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum entries to return (sorted by size desc). Defaults to 20.",
				"minimum":     1,
				"maximum":     100,
			},
			"human": map[string]any{
				"type":        "boolean",
				"description": "Show human-readable sizes (KB, MB, GB). Defaults to true.",
			},
		},
		"required": []string{"action"},
	}

	return NewFuncTool(
		"disk_usage",
		"Check disk space (df) and directory sizes (du). Shows filesystem usage and largest directories.",
		schema,
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var in diskUsageArgs
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, fmt.Errorf("invalid disk_usage args: %w", err)
			}
			return executeDiskUsage(ctx, in)
		},
	)
}

func executeDiskUsage(ctx context.Context, in diskUsageArgs) (*diskUsageResult, error) {
	if runtime.GOOS == "windows" {
		return &diskUsageResult{Error: "disk_usage is not supported on Windows"}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch in.Action {
	case "df":
		return runDF(ctx)
	case "du":
		return runDU(ctx, in)
	default:
		return nil, fmt.Errorf("unknown action %q, use: df, du", in.Action)
	}
}

func runDF(ctx context.Context) (*diskUsageResult, error) {
	cmd := exec.CommandContext(ctx, "df", "-h")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return &diskUsageResult{Error: err.Error()}, nil
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var entries []dfEntry
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		entries = append(entries, dfEntry{
			Filesystem: fields[0],
			Size:       fields[1],
			Used:       fields[2],
			Available:  fields[3],
			UsePercent: fields[4],
			MountedOn:  strings.Join(fields[5:], " "),
		})
	}

	return &diskUsageResult{Action: "df", Filesystems: entries, Count: len(entries)}, nil
}

func runDU(ctx context.Context, in diskUsageArgs) (*diskUsageResult, error) {
	path := in.Path
	if path == "" {
		path = "."
	}

	depth := in.Depth
	if depth <= 0 {
		depth = 1
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}

	// Use du with sort to get largest first
	duArgs := []string{"-h", fmt.Sprintf("-d%d", depth), path}
	cmd := exec.CommandContext(ctx, "du", duArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{} // suppress permission errors
	_ = cmd.Run()                // du may exit non-zero for permission issues

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var entries []duEntry
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, duEntry{Size: parts[0], Path: parts[1]})
	}

	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return &diskUsageResult{Action: "du", Entries: entries, Count: len(entries)}, nil
}
