package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type s3Uploader struct {
	bucket     string
	prefix     string
	endpoint   string
	baseDirAbs string
}

func newS3UploaderFromEnv(baseDir string) (BackupUploader, error) {
	bucket := strings.TrimSpace(os.Getenv("AGENT_STORAGE_S3_BUCKET"))
	if bucket == "" {
		return nil, fmt.Errorf("s3 backup disabled")
	}
	endpoint := strings.TrimSpace(os.Getenv("AGENT_STORAGE_S3_ENDPOINT"))
	prefix := strings.Trim(strings.TrimSpace(os.Getenv("AGENT_STORAGE_S3_PREFIX")), "/")
	if _, err := exec.LookPath("aws"); err != nil {
		return nil, fmt.Errorf("aws cli not found for s3 backup")
	}

	absBase, _ := filepath.Abs(baseDir)
	return &s3Uploader{
		bucket:     bucket,
		prefix:     prefix,
		endpoint:   endpoint,
		baseDirAbs: absBase,
	}, nil
}

func (u *s3Uploader) UploadFile(ctx context.Context, localPath string) (*BackupInfo, error) {
	if u == nil {
		return nil, fmt.Errorf("s3 uploader not configured")
	}
	if _, err := os.Stat(localPath); err != nil {
		return nil, err
	}

	key := u.objectKey(localPath)
	uri := fmt.Sprintf("s3://%s/%s", u.bucket, key)
	args := []string{"s3", "cp", localPath, uri, "--only-show-errors"}
	if strings.TrimSpace(u.endpoint) != "" {
		args = append(args, "--endpoint-url", strings.TrimSpace(u.endpoint))
	}
	cmd := exec.CommandContext(ctx, "aws", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("aws s3 cp failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return &BackupInfo{
		Provider: "s3",
		Bucket:   u.bucket,
		Key:      key,
		URL:      uri,
	}, nil
}

func (u *s3Uploader) objectKey(localPath string) string {
	abs, _ := filepath.Abs(localPath)
	rel := filepath.Base(localPath)
	if u.baseDirAbs != "" {
		if r, err := filepath.Rel(u.baseDirAbs, abs); err == nil {
			r = filepath.Clean(r)
			if r != "." && !strings.HasPrefix(r, "..") {
				rel = r
			}
		}
	}
	rel = strings.Trim(strings.ReplaceAll(filepath.ToSlash(rel), " ", "-"), "/")
	if u.prefix == "" {
		return rel
	}
	return u.prefix + "/" + rel
}
