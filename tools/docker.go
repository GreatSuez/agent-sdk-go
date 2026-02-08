package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type dockerArgs struct {
	Operation  string            `json:"operation"`
	Image      string            `json:"image,omitempty"`
	Container  string            `json:"container,omitempty"`
	Command    []string          `json:"command,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Ports      []string          `json:"ports,omitempty"`
	Volumes    []string          `json:"volumes,omitempty"`
	Dockerfile string            `json:"dockerfile,omitempty"`
	Tag        string            `json:"tag,omitempty"`
	BuildDir   string            `json:"buildDir,omitempty"`
	Detach     bool              `json:"detach,omitempty"`
	Remove     bool              `json:"remove,omitempty"`
	Tail       string            `json:"tail,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
}

// DockerResult contains the result of a docker operation.
type DockerResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration,omitempty"`
}

func NewDocker() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"enum":        []string{"ps", "images", "run", "stop", "logs", "inspect", "build", "pull", "exec"},
				"description": "Operation: ps, images, run, stop, logs, inspect, build, pull, exec.",
			},
			"image": map[string]any{
				"type":        "string",
				"description": "Docker image name (for run, pull, build operations).",
			},
			"container": map[string]any{
				"type":        "string",
				"description": "Container name or ID (for stop, logs, inspect, exec operations).",
			},
			"command": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Command to run in the container (for run, exec operations).",
			},
			"env": map[string]any{
				"type":        "object",
				"description": "Environment variables as key-value pairs (for run operation).",
			},
			"ports": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Port mappings in host:container format (for run operation).",
			},
			"volumes": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Volume mounts in host:container format (for run operation).",
			},
			"dockerfile": map[string]any{
				"type":        "string",
				"description": "Path to Dockerfile (for build operation). Defaults to './Dockerfile'.",
			},
			"tag": map[string]any{
				"type":        "string",
				"description": "Tag for the built image (for build operation).",
			},
			"buildDir": map[string]any{
				"type":        "string",
				"description": "Build context directory (for build operation). Defaults to '.'.",
			},
			"detach": map[string]any{
				"type":        "boolean",
				"description": "Run container in detached mode (for run operation).",
			},
			"remove": map[string]any{
				"type":        "boolean",
				"description": "Remove container after exit (for run operation).",
			},
			"tail": map[string]any{
				"type":        "string",
				"description": "Number of lines to show from end of logs (for logs operation). Default: 100.",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds. Default: 120. Maximum: 600.",
			},
		},
		"required": []string{"operation"},
	}

	return NewFuncTool(
		"docker",
		"Manage Docker containers and images. List, run, stop, inspect containers; build and pull images; view logs.",
		schema,
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var in dockerArgs
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, fmt.Errorf("invalid docker args: %w", err)
			}

			timeout := in.Timeout
			if timeout <= 0 {
				timeout = 120
			}
			if timeout > 600 {
				timeout = 600
			}

			switch in.Operation {
			case "ps":
				return dockerExec(ctx, timeout, "ps", "--format", "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}\t{{.Ports}}")
			case "images":
				return dockerExec(ctx, timeout, "images", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}")
			case "run":
				return dockerRun(ctx, timeout, in)
			case "stop":
				if in.Container == "" {
					return &DockerResult{Success: false, Error: "container is required for stop"}, nil
				}
				return dockerExec(ctx, timeout, "stop", in.Container)
			case "logs":
				return dockerLogs(ctx, timeout, in)
			case "inspect":
				if in.Container == "" {
					return &DockerResult{Success: false, Error: "container is required for inspect"}, nil
				}
				return dockerExec(ctx, timeout, "inspect", in.Container)
			case "build":
				return dockerBuild(ctx, timeout, in)
			case "pull":
				if in.Image == "" {
					return &DockerResult{Success: false, Error: "image is required for pull"}, nil
				}
				return dockerExec(ctx, timeout, "pull", in.Image)
			case "exec":
				return dockerExecInContainer(ctx, timeout, in)
			default:
				return nil, fmt.Errorf("unsupported operation %q", in.Operation)
			}
		},
	)
}

func dockerExec(ctx context.Context, timeout int, args ...string) (*DockerResult, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &DockerResult{
		Duration: time.Since(start).String(),
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("%v: %s", err, stderr.String())
		result.Output = limitOutput(stdout.String(), 100*1024)
	} else {
		result.Success = true
		result.Output = limitOutput(stdout.String(), 100*1024)
	}

	return result, nil
}

func dockerRun(ctx context.Context, timeout int, in dockerArgs) (*DockerResult, error) {
	if in.Image == "" {
		return &DockerResult{Success: false, Error: "image is required for run"}, nil
	}

	args := []string{"run"}

	if in.Detach {
		args = append(args, "-d")
	}
	if in.Remove {
		args = append(args, "--rm")
	}

	for _, p := range in.Ports {
		args = append(args, "-p", p)
	}
	for _, v := range in.Volumes {
		args = append(args, "-v", v)
	}
	for k, v := range in.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, in.Image)
	args = append(args, in.Command...)

	return dockerExec(ctx, timeout, args...)
}

func dockerLogs(ctx context.Context, timeout int, in dockerArgs) (*DockerResult, error) {
	if in.Container == "" {
		return &DockerResult{Success: false, Error: "container is required for logs"}, nil
	}

	args := []string{"logs"}
	tail := in.Tail
	if tail == "" {
		tail = "100"
	}
	args = append(args, "--tail", tail, in.Container)

	return dockerExec(ctx, timeout, args...)
}

func dockerBuild(ctx context.Context, timeout int, in dockerArgs) (*DockerResult, error) {
	buildDir := in.BuildDir
	if buildDir == "" {
		buildDir = "."
	}

	args := []string{"build"}

	if in.Tag != "" {
		args = append(args, "-t", in.Tag)
	}
	if in.Dockerfile != "" {
		args = append(args, "-f", in.Dockerfile)
	}

	args = append(args, buildDir)

	return dockerExec(ctx, timeout, args...)
}

func dockerExecInContainer(ctx context.Context, timeout int, in dockerArgs) (*DockerResult, error) {
	if in.Container == "" {
		return &DockerResult{Success: false, Error: "container is required for exec"}, nil
	}
	if len(in.Command) == 0 {
		return &DockerResult{Success: false, Error: "command is required for exec"}, nil
	}

	args := []string{"exec", in.Container}
	args = append(args, in.Command...)

	return dockerExec(ctx, timeout, args...)
}

// DockerAvailable checks if docker CLI is available.
func DockerAvailable() bool {
	cmd := exec.Command("docker", "version", "--format", "{{.Client.Version}}")
	return cmd.Run() == nil
}

// init registers a check - we don't want to fail if docker isn't installed
func init() {
	_ = strings.TrimSpace // ensure strings import is used
}
