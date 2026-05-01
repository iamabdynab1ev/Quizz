package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

func TestLocalFileStorageSave(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	storage := NewLocalFileStorage(baseDir)
	storage.now = func() time.Time {
		return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	}

	url, err := storage.Save(domain.UploadTypeFile, "report.pdf", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.HasPrefix(url, "/uploads/file/2026-05-01/") {
		t.Fatalf("unexpected url: %s", url)
	}

	if !strings.HasSuffix(url, ".pdf") {
		t.Fatalf("expected pdf suffix, got %s", url)
	}

	filePath := filepath.Join(baseDir, filepath.FromSlash(strings.TrimPrefix(url, "/uploads/")))
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected saved file, got %v", err)
	}

	if string(content) != "hello" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}
