package http

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"
)

type uploadUseCase interface {
	Upload(ctx context.Context, params domain.UploadParams) (domain.Upload, error)
}

type UploadsHandler struct {
	logger       *slog.Logger
	useCase      uploadUseCase
	maxSizeBytes int64
}

func NewUploadsHandler(logger *slog.Logger, useCase uploadUseCase, maxSizeBytes int64) *UploadsHandler {
	return &UploadsHandler{
		logger:       logger,
		useCase:      useCase,
		maxSizeBytes: maxSizeBytes,
	}
}

func (h *UploadsHandler) CreateUpload(w nethttp.ResponseWriter, r *nethttp.Request) {
	if h.maxSizeBytes > 0 {
		r.Body = nethttp.MaxBytesReader(w, r.Body, h.maxSizeBytes+1<<20)
	}

	if err := r.ParseMultipartForm(h.maxSizeBytes + 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_multipart", "invalid multipart form")
		return
	}

	uploadType := domain.NormalizeUploadType(r.FormValue("type"))
	if !uploadType.IsValid() {
		writeError(w, nethttp.StatusBadRequest, "validation_error", "upload type is invalid")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "validation_error", "file is required")
		return
	}
	defer file.Close()

	if header == nil {
		writeError(w, nethttp.StatusBadRequest, "validation_error", "file is required")
		return
	}

	if h.maxSizeBytes > 0 && header.Size > h.maxSizeBytes {
		writeError(w, nethttp.StatusRequestEntityTooLarge, "file_too_large", "file is too large")
		return
	}

	sniffed, reader, err := sniffFileReader(file)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_file", "invalid file")
		return
	}

	result, err := h.useCase.Upload(r.Context(), domain.UploadParams{
		Type:        uploadType,
		Filename:    header.Filename,
		ContentType: sniffed,
		SizeBytes:   header.Size,
		Body:        reader,
	})
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "upload failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, result); err != nil {
		h.logger.ErrorContext(r.Context(), "upload response failed", slog.String("error", err.Error()))
	}
}

func sniffFileReader(file io.Reader) (string, io.Reader, error) {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return "", nil, err
	}

	sniffed := nethttp.DetectContentType(head[:n])
	reader := io.MultiReader(bytes.NewReader(head[:n]), file)
	return sniffed, reader, nil
}
