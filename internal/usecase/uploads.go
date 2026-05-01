package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"

	"lms-arvand-backend/internal/domain"
)

type uploadStorage interface {
	Save(uploadType domain.UploadType, originalFilename string, body io.Reader) (string, error)
}

type UploadUseCase struct {
	storage      uploadStorage
	maxSizeBytes int64
}

func NewUploadUseCase(storage uploadStorage, maxSizeBytes int64) *UploadUseCase {
	return &UploadUseCase{
		storage:      storage,
		maxSizeBytes: maxSizeBytes,
	}
}

func (u *UploadUseCase) Upload(ctx context.Context, params domain.UploadParams) (domain.Upload, error) {
	_ = ctx

	normalized, err := normalizeUploadParams(params, u.maxSizeBytes)
	if err != nil {
		return domain.Upload{}, fmt.Errorf("usecase uploads upload: %w", err)
	}

	url, err := u.storage.Save(normalized.Type, normalized.Filename, normalized.Body)
	if err != nil {
		return domain.Upload{}, fmt.Errorf("usecase uploads upload save: %w", err)
	}

	return domain.Upload{
		URL:       url,
		Filename:  normalized.Filename,
		SizeBytes: normalized.SizeBytes,
	}, nil
}

func normalizeUploadParams(params domain.UploadParams, maxSizeBytes int64) (domain.UploadParams, error) {
	params.Type = domain.NormalizeUploadType(string(params.Type))
	params.Filename = strings.TrimSpace(params.Filename)
	params.ContentType = strings.ToLower(strings.TrimSpace(params.ContentType))

	if params.Body == nil {
		return domain.UploadParams{}, domain.ValidationError("file is required")
	}

	if !params.Type.IsValid() {
		return domain.UploadParams{}, domain.ValidationError("upload type is invalid")
	}

	if params.Filename == "" {
		return domain.UploadParams{}, domain.ValidationError("filename is required")
	}

	if params.SizeBytes <= 0 {
		return domain.UploadParams{}, domain.ValidationError("file is empty")
	}

	if maxSizeBytes > 0 && params.SizeBytes > maxSizeBytes {
		return domain.UploadParams{}, domain.ValidationError("file is too large")
	}

	if params.Type.RequiresImageContent() && !strings.HasPrefix(params.ContentType, "image/") {
		return domain.UploadParams{}, domain.ValidationError("only image files are allowed for this upload type")
	}

	if params.Type.RequiresVideoContent() && !strings.HasPrefix(params.ContentType, "video/") {
		return domain.UploadParams{}, domain.ValidationError("only video files are allowed for this upload type")
	}

	return params, nil
}
