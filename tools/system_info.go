package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type systemInfoArgs struct {
	Action string `json:"action"` // summary, cpu, memory, os, network, uptime
}

type systemInfoResult struct {
	Action string `json:"action"`
	Info   any    `json:"info"`
	Error  string `json:"error,omitempty"`
}

type systemSummary struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	GoVersion    string `json:"goVersion"`
	NumCPU       int    `json:"numCpu"`
	NumGoroutine int    `json:"numGoroutine"`
	Uptime       string `json:"uptime,omitempty"`
	LoadAvg      string `json:"loadAvg,omitempty"`
	KernelInfo   string `json:"kernelInfo,omitempty"`
	MemInfo      string `json:"memInfo,omitempty"`
}

type cpuInfo struct {
	NumCPU    int      `json:"numCpu"`
	ModelName string   `json:"modelName,omitempty"`
	Details   []string `json:"details,omitempty"`
}

type memInfo struct {
	Total     string `json:"total,omitempty"`
	Used      string `json:"used,omitempty"`
	Free      string `json:"free,omitempty"`
	Available string `json:"available,omitempty"`
	SwapTotal string `json:"swapTotal,omitempty"`
	SwapUsed  string `json:"swapUsed,omitempty"`
	Raw       string `json:"raw,omitempty"`
}

type networkInfo struct {
	Hostname   string   `json:"hostname"`
	Interfaces []string `json:"interfaces"`
}

func NewSystemInfo() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"summary", "cpu", "memory", "os", "network", "uptime"},
				"description": "Info to retrieve: summary (overview), cpu, memory, os (kernel/distro), network (interfaces), uptime.",
			},
		},
		"required": []string{"action"},
	}

	return NewFuncTool(
		"system_info",
		"Get system information: hostname, OS, CPU, memory, uptime, network interfaces. Like uname, free, hostnamectl.",
		schema,
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var in systemInfoArgs
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, fmt.Errorf("invalid system_info args: %w", err)
			}
			return executeSystemInfo(ctx, in)
		},
	)
}

func executeSystemInfo(ctx context.Context, in systemInfoArgs) (*systemInfoResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result := &systemInfoResult{Action: in.Action}

	switch in.Action {
	case "summary":
		s := systemSummary{
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		}
		s.Hostname, _ = os.Hostname()
		s.Uptime = runCmd(ctx, "uptime")
		s.KernelInfo = runCmd(ctx, "uname", "-a")

		switch runtime.GOOS {
		case "darwin":
			s.MemInfo = runCmd(ctx, "vm_stat")
		case "linux":
			s.MemInfo = runCmd(ctx, "free", "-h")
		}
		s.LoadAvg = extractLoadAvg(s.Uptime)
		result.Info = s

	case "cpu":
		info := cpuInfo{NumCPU: runtime.NumCPU()}
		switch runtime.GOOS {
		case "darwin":
			info.ModelName = runCmd(ctx, "sysctl", "-n", "machdep.cpu.brand_string")
			info.Details = []string{
				"cores: " + runCmd(ctx, "sysctl", "-n", "hw.physicalcpu"),
				"threads: " + runCmd(ctx, "sysctl", "-n", "hw.logicalcpu"),
			}
		case "linux":
			out := runCmd(ctx, "lscpu")
			if out != "" {
				info.Details = strings.Split(out, "\n")
				for _, line := range info.Details {
					if strings.Contains(line, "Model name") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							info.ModelName = strings.TrimSpace(parts[1])
						}
					}
				}
			}
		}
		result.Info = info

	case "memory":
		m := memInfo{}
		switch runtime.GOOS {
		case "darwin":
			m.Raw = runCmd(ctx, "vm_stat")
			sysMemStr := runCmd(ctx, "sysctl", "-n", "hw.memsize")
			m.Total = strings.TrimSpace(sysMemStr) + " bytes"
		case "linux":
			freeOut := runCmd(ctx, "free", "-h")
			m.Raw = freeOut
			for _, line := range strings.Split(freeOut, "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 4 && fields[0] == "Mem:" {
					m.Total = fields[1]
					m.Used = fields[2]
					m.Free = fields[3]
					if len(fields) >= 7 {
						m.Available = fields[6]
					}
				}
				if len(fields) >= 3 && fields[0] == "Swap:" {
					m.SwapTotal = fields[1]
					m.SwapUsed = fields[2]
				}
			}
		}
		result.Info = m

	case "os":
		info := map[string]string{
			"os":     runtime.GOOS,
			"arch":   runtime.GOARCH,
			"kernel": runCmd(ctx, "uname", "-r"),
		}
		hostname, _ := os.Hostname()
		info["hostname"] = hostname

		switch runtime.GOOS {
		case "darwin":
			info["version"] = runCmd(ctx, "sw_vers", "-productVersion")
			info["build"] = runCmd(ctx, "sw_vers", "-buildVersion")
		case "linux":
			if out := runCmd(ctx, "cat", "/etc/os-release"); out != "" {
				for _, line := range strings.Split(out, "\n") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						key := strings.ToLower(parts[0])
						val := strings.Trim(parts[1], "\"")
						if key == "pretty_name" || key == "version_id" || key == "id" {
							info[key] = val
						}
					}
				}
			}
		}
		result.Info = info

	case "network":
		hostname, _ := os.Hostname()
		ni := networkInfo{Hostname: hostname}

		out := runCmd(ctx, "ifconfig")
		if out == "" {
			out = runCmd(ctx, "ip", "addr")
		}
		if out != "" {
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "inet ") || strings.HasPrefix(line, "inet6 ") {
					ni.Interfaces = append(ni.Interfaces, line)
				}
			}
		}
		result.Info = ni

	case "uptime":
		result.Info = map[string]string{"uptime": runCmd(ctx, "uptime")}

	default:
		return nil, fmt.Errorf("unknown action %q, use: summary, cpu, memory, os, network, uptime", in.Action)
	}

	return result, nil
}

func runCmd(ctx context.Context, name string, args ...string) string {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func extractLoadAvg(uptimeStr string) string {
	if idx := strings.Index(uptimeStr, "load average"); idx >= 0 {
		return strings.TrimSpace(uptimeStr[idx:])
	}
	if idx := strings.Index(uptimeStr, "load averages"); idx >= 0 {
		return strings.TrimSpace(uptimeStr[idx:])
	}
	return ""
}
