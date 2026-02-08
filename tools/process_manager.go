package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type processManagerArgs struct {
	Action string `json:"action"` // list, find, info, top
	Name   string `json:"name,omitempty"`
	PID    int    `json:"pid,omitempty"`
	User   string `json:"user,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	SortBy string `json:"sortBy,omitempty"` // cpu, mem, pid, name
}

type processInfo struct {
	PID     int     `json:"pid"`
	Name    string  `json:"name"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpuPercent"`
	Memory  float64 `json:"memPercent"`
	VSZ     string  `json:"vsz"`
	RSS     string  `json:"rss"`
	Status  string  `json:"status"`
	Started string  `json:"started"`
	Command string  `json:"command"`
}

type processResult struct {
	Action    string        `json:"action"`
	Processes []processInfo `json:"processes,omitempty"`
	Count     int           `json:"count"`
	System    *systemStats  `json:"system,omitempty"`
	Error     string        `json:"error,omitempty"`
}

type systemStats struct {
	Uptime     string `json:"uptime"`
	LoadAvg    string `json:"loadAvg"`
	TotalProcs int    `json:"totalProcesses"`
}

func NewProcessManager() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list", "find", "info", "top"},
				"description": "Action: list (all processes), find (by name), info (by PID), top (resource hogs).",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Process name to search for (used with 'find' action).",
			},
			"pid": map[string]any{
				"type":        "integer",
				"description": "Process ID (used with 'info' action).",
			},
			"user": map[string]any{
				"type":        "string",
				"description": "Filter processes by user.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum processes to return. Defaults to 20.",
				"minimum":     1,
				"maximum":     100,
			},
			"sortBy": map[string]any{
				"type":        "string",
				"enum":        []string{"cpu", "mem", "pid", "name"},
				"description": "Sort order for 'top' action. Defaults to cpu.",
			},
		},
		"required": []string{"action"},
	}

	return NewFuncTool(
		"process_manager",
		"List, find, and inspect running processes. Get top CPU/memory consumers. Like ps, top, pgrep.",
		schema,
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var in processManagerArgs
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, fmt.Errorf("invalid process_manager args: %w", err)
			}
			return executeProcessManager(ctx, in)
		},
	)
}

func executeProcessManager(ctx context.Context, in processManagerArgs) (*processResult, error) {
	if runtime.GOOS == "windows" {
		return &processResult{Error: "process_manager is not supported on Windows"}, nil
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	switch in.Action {
	case "list":
		return psCommand(ctx, "", in.User, limit)
	case "find":
		if in.Name == "" {
			return nil, fmt.Errorf("name is required for 'find' action")
		}
		return psCommand(ctx, in.Name, in.User, limit)
	case "info":
		if in.PID == 0 {
			return nil, fmt.Errorf("pid is required for 'info' action")
		}
		return psInfoByPID(ctx, in.PID)
	case "top":
		sortBy := in.SortBy
		if sortBy == "" {
			sortBy = "cpu"
		}
		return psTop(ctx, sortBy, limit)
	default:
		return nil, fmt.Errorf("unknown action %q", in.Action)
	}
}

func psCommand(ctx context.Context, nameFilter, userFilter string, limit int) (*processResult, error) {
	args := []string{"ax", "-o", "pid,user,%cpu,%mem,vsz,rss,stat,start,comm"}
	cmd := exec.CommandContext(ctx, "ps", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return &processResult{Error: err.Error()}, nil
	}

	procs := parsePSOutput(out.String(), nameFilter, userFilter, limit)
	return &processResult{Action: "list", Processes: procs, Count: len(procs)}, nil
}

func psInfoByPID(ctx context.Context, pid int) (*processResult, error) {
	cmd := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "pid,user,%cpu,%mem,vsz,rss,stat,start,command")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return &processResult{Error: fmt.Sprintf("process %d not found: %v", pid, err)}, nil
	}

	procs := parsePSOutput(out.String(), "", "", 1)
	if len(procs) == 0 {
		return &processResult{Error: fmt.Sprintf("process %d not found", pid)}, nil
	}

	// Also get /proc info if available
	cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if len(cmdline) > 0 {
		procs[0].Command = strings.ReplaceAll(string(cmdline), "\x00", " ")
	}

	return &processResult{Action: "info", Processes: procs, Count: 1}, nil
}

func psTop(ctx context.Context, sortBy string, limit int) (*processResult, error) {
	sortFlag := "%cpu"
	switch sortBy {
	case "mem":
		sortFlag = "%mem"
	case "pid":
		sortFlag = "pid"
	}

	args := []string{"ax", "-o", "pid,user,%cpu,%mem,vsz,rss,stat,start,comm", "--sort=-" + sortFlag}
	cmd := exec.CommandContext(ctx, "ps", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return &processResult{Error: err.Error()}, nil
	}

	procs := parsePSOutput(out.String(), "", "", limit)

	// Get system stats
	sys := &systemStats{TotalProcs: countLines(out.String()) - 1}
	if uptimeOut, err := exec.CommandContext(ctx, "uptime").Output(); err == nil {
		sys.Uptime = strings.TrimSpace(string(uptimeOut))
	}

	return &processResult{Action: "top", Processes: procs, Count: len(procs), System: sys}, nil
}

func parsePSOutput(output, nameFilter, userFilter string, limit int) []processInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil
	}

	var procs []processInfo
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		name := fields[8]
		user := fields[1]

		if nameFilter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(nameFilter)) {
			continue
		}
		if userFilter != "" && !strings.EqualFold(user, userFilter) {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)

		procs = append(procs, processInfo{
			PID:     pid,
			User:    user,
			CPU:     cpu,
			Memory:  mem,
			VSZ:     fields[4],
			RSS:     fields[5],
			Status:  fields[6],
			Started: fields[7],
			Name:    name,
			Command: strings.Join(fields[8:], " "),
		})

		if len(procs) >= limit {
			break
		}
	}
	return procs
}

func countLines(s string) int {
	return strings.Count(s, "\n")
}
