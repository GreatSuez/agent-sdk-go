package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type tmpdirArgs struct {
	Operation string `json:"operation"`
	Prefix    string `json:"prefix,omitempty"`
	Path      string `json:"path,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	Content   string `json:"content,omitempty"`
}

// TmpDirResult contains the result of a tmpdir operation.
type TmpDirResult struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

var (
	tmpdirMu   sync.RWMutex
	tmpdirDirs = make(map[string]string) // path -> prefix
)

func NewTmpDir() Tool {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"enum":        []string{"create", "cleanup", "list", "write_file", "read_file"},
				"description": "Operation: create, cleanup, list, write_file, read_file.",
			},
			"prefix": map[string]any{
				"type":        "string",
				"description": "Prefix for the temp directory name (for create operation).",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Path of the temp directory (for cleanup, write_file, read_file operations).",
			},
			"fileName": map[string]any{
				"type":        "string",
				"description": "File name within the temp directory (for write_file, read_file operations).",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file (for write_file operation).",
			},
		},
		"required": []string{"operation"},
	}

	return NewFuncTool(
		"tmpdir",
		"Create and manage temporary directories. Supports creating temp dirs, writing/reading files, listing managed dirs, and cleanup.",
		schema,
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var in tmpdirArgs
			if err := json.Unmarshal(args, &in); err != nil {
				return nil, fmt.Errorf("invalid tmpdir args: %w", err)
			}

			switch in.Operation {
			case "create":
				return tmpdirCreate(in.Prefix)
			case "cleanup":
				return tmpdirCleanup(in.Path)
			case "list":
				return tmpdirList()
			case "write_file":
				return tmpdirWriteFile(in.Path, in.FileName, in.Content)
			case "read_file":
				return tmpdirReadFile(in.Path, in.FileName)
			default:
				return nil, fmt.Errorf("unsupported operation %q", in.Operation)
			}
		},
	)
}

func tmpdirCreate(prefix string) (*TmpDirResult, error) {
	if prefix == "" {
		prefix = "ai-agent-"
	}

	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return &TmpDirResult{Success: false, Error: fmt.Sprintf("failed to create temp dir: %v", err)}, nil
	}

	tmpdirMu.Lock()
	tmpdirDirs[dir] = prefix
	tmpdirMu.Unlock()

	return &TmpDirResult{
		Success: true,
		Data: map[string]any{
			"path":    dir,
			"prefix":  prefix,
			"message": "temporary directory created",
		},
	}, nil
}

func tmpdirCleanup(path string) (*TmpDirResult, error) {
	if path == "" {
		return &TmpDirResult{Success: false, Error: "path is required"}, nil
	}

	tmpdirMu.Lock()
	_, tracked := tmpdirDirs[path]
	if tracked {
		delete(tmpdirDirs, path)
	}
	tmpdirMu.Unlock()

	if !tracked {
		return &TmpDirResult{Success: false, Error: "path is not a managed temp directory"}, nil
	}

	if err := os.RemoveAll(path); err != nil {
		return &TmpDirResult{Success: false, Error: fmt.Sprintf("failed to remove temp dir: %v", err)}, nil
	}

	return &TmpDirResult{
		Success: true,
		Data: map[string]any{
			"path":    path,
			"message": "temporary directory removed",
		},
	}, nil
}

func tmpdirList() (*TmpDirResult, error) {
	tmpdirMu.RLock()
	defer tmpdirMu.RUnlock()

	dirs := make([]map[string]any, 0, len(tmpdirDirs))
	for path, prefix := range tmpdirDirs {
		entry := map[string]any{
			"path":   path,
			"prefix": prefix,
		}
		if info, err := os.Stat(path); err == nil {
			entry["exists"] = true
			entry["modTime"] = info.ModTime().String()
		} else {
			entry["exists"] = false
		}
		dirs = append(dirs, entry)
	}

	return &TmpDirResult{
		Success: true,
		Data: map[string]any{
			"directories": dirs,
			"count":       len(dirs),
		},
	}, nil
}

func tmpdirWriteFile(dirPath, fileName, content string) (*TmpDirResult, error) {
	if dirPath == "" {
		return &TmpDirResult{Success: false, Error: "path is required"}, nil
	}
	if fileName == "" {
		return &TmpDirResult{Success: false, Error: "fileName is required"}, nil
	}

	tmpdirMu.RLock()
	_, tracked := tmpdirDirs[dirPath]
	tmpdirMu.RUnlock()

	if !tracked {
		return &TmpDirResult{Success: false, Error: "path is not a managed temp directory"}, nil
	}

	filePath := filepath.Join(dirPath, fileName)

	// Create subdirectories if needed
	if dir := filepath.Dir(filePath); dir != dirPath {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return &TmpDirResult{Success: false, Error: fmt.Sprintf("failed to create subdirectory: %v", err)}, nil
		}
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return &TmpDirResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}, nil
	}

	return &TmpDirResult{
		Success: true,
		Data: map[string]any{
			"path":     filePath,
			"fileName": fileName,
			"size":     len(content),
			"message":  "file written",
		},
	}, nil
}

func tmpdirReadFile(dirPath, fileName string) (*TmpDirResult, error) {
	if dirPath == "" {
		return &TmpDirResult{Success: false, Error: "path is required"}, nil
	}
	if fileName == "" {
		return &TmpDirResult{Success: false, Error: "fileName is required"}, nil
	}

	tmpdirMu.RLock()
	_, tracked := tmpdirDirs[dirPath]
	tmpdirMu.RUnlock()

	if !tracked {
		return &TmpDirResult{Success: false, Error: "path is not a managed temp directory"}, nil
	}

	filePath := filepath.Join(dirPath, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return &TmpDirResult{Success: false, Error: fmt.Sprintf("failed to read file: %v", err)}, nil
	}

	return &TmpDirResult{
		Success: true,
		Data: map[string]any{
			"path":     filePath,
			"fileName": fileName,
			"content":  string(data),
			"size":     len(data),
		},
	}, nil
}

// CleanupAllTmpDirs removes all managed temporary directories.
func CleanupAllTmpDirs() {
	tmpdirMu.Lock()
	defer tmpdirMu.Unlock()
	for path := range tmpdirDirs {
		os.RemoveAll(path)
	}
	tmpdirDirs = make(map[string]string)
}
