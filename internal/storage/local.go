package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type FileStorage interface {
	Save(uploadType domain.UploadType, originalFilename string, body io.Reader) (string, error)
}

type LocalFileStorage struct {
	baseDir      string
	publicPrefix string
	now          func() time.Time
}

func NewLocalFileStorage(baseDir string) *LocalFileStorage {
	return &LocalFileStorage{
		baseDir:      strings.TrimSpace(baseDir),
		publicPrefix: "/uploads",
		now:          time.Now,
	}
}

func (s *LocalFileStorage) Save(uploadType domain.UploadType, originalFilename string, body io.Reader) (string, error) {
	baseDir := strings.TrimSpace(s.baseDir)
	if baseDir == "" {
		baseDir = "uploads"
	}

	uploadTypeValue := strings.TrimSpace(uploadType.String())
	if uploadTypeValue == "" {
		uploadTypeValue = domain.UploadTypeFile.String()
	}

	ext := filepath.Ext(strings.TrimSpace(originalFilename))
	datePart := s.now().UTC().Format("2006-01-02")
	uniqueName, err := randomHexName(ext)
	if err != nil {
		return "", fmt.Errorf("storage build unique filename: %w", err)
	}

	fullDir := filepath.Join(baseDir, uploadTypeValue, datePart)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return "", fmt.Errorf("storage create upload directory: %w", err)
	}

	filePath := filepath.Join(fullDir, uniqueName)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("storage create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, body); err != nil {
		return "", fmt.Errorf("storage copy file: %w", err)
	}

	return filepath.ToSlash(filepath.Join(s.publicPrefix, uploadTypeValue, datePart, uniqueName)), nil
}

func randomHexName(ext string) (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%s", hex.EncodeToString(raw[:]), ext), nil
}
