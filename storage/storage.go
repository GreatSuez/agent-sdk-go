package storage

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type BackupInfo struct {
	Provider string `json:"provider,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Key      string `json:"key,omitempty"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
}

type SaveResult struct {
	Path   string      `json:"path"`
	Bytes  int         `json:"bytes"`
	Backup *BackupInfo `json:"backup,omitempty"`
}

type BackupUploader interface {
	UploadFile(ctx context.Context, localPath string) (*BackupInfo, error)
}

type Manager struct {
	baseDir  string
	uploader BackupUploader
}

var (
	defaultManager *Manager
	defaultOnce    sync.Once
)

func Default() *Manager {
	defaultOnce.Do(func() {
		defaultManager = NewFromEnv()
	})
	return defaultManager
}

func NewFromEnv() *Manager {
	baseDir := strings.TrimSpace(os.Getenv("AGENT_STORAGE_DIR"))
	if baseDir == "" {
		baseDir = "./.ai-agent/generated"
	}
	mgr := &Manager{baseDir: baseDir}
	if uploader, err := newS3UploaderFromEnv(baseDir); err == nil {
		mgr.uploader = uploader
	}
	return mgr
}

func (m *Manager) BaseDir() string {
	if m == nil {
		return "./.ai-agent/generated"
	}
	if strings.TrimSpace(m.baseDir) == "" {
		return "./.ai-agent/generated"
	}
	return m.baseDir
}

func (m *Manager) SaveBytes(ctx context.Context, requestedPath, defaultFileName string, content []byte) (SaveResult, error) {
	path := m.resolveOutputPath(requestedPath, defaultFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return SaveResult{}, err
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return SaveResult{}, err
	}
	result := SaveResult{Path: path, Bytes: len(content)}
	if m != nil && m.uploader != nil {
		backup, err := m.uploader.UploadFile(ctx, path)
		if err != nil {
			result.Backup = &BackupInfo{Provider: "s3", Error: err.Error()}
		} else {
			result.Backup = backup
		}
	}
	return result, nil
}

func (m *Manager) resolveOutputPath(requestedPath, defaultFileName string) string {
	base := m.BaseDir()
	requested := strings.TrimSpace(requestedPath)
	if requested == "" {
		return filepath.Join(base, sanitizeFileName(defaultFileName))
	}
	if filepath.IsAbs(requested) {
		return requested
	}
	clean := filepath.Clean(strings.TrimPrefix(requested, "./"))
	if clean == "." || clean == "" {
		return filepath.Join(base, sanitizeFileName(defaultFileName))
	}
	if strings.HasPrefix(clean, "..") {
		return filepath.Join(base, sanitizeFileName(defaultFileName))
	}
	return filepath.Join(base, clean)
}

var fileNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFileName(name string) string {
	v := strings.TrimSpace(name)
	if v == "" {
		return "artifact.txt"
	}
	v = strings.ReplaceAll(v, " ", "-")
	v = fileNameSanitizer.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-._")
	if v == "" {
		return "artifact.txt"
	}
	if len(v) > 120 {
		v = v[:120]
	}
	return v
}
