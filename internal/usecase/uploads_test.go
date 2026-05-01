package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"lms-arvand-backend/internal/domain"
)

type uploadStorageStub struct {
	saveCalled bool
	savedType  domain.UploadType
	savedName  string
	savedBody  string
	returnURL  string
	err        error
}

func (s *uploadStorageStub) Save(uploadType domain.UploadType, originalFilename string, body io.Reader) (string, error) {
	s.saveCalled = true
	s.savedType = uploadType
	s.savedName = originalFilename

	bytes, readErr := io.ReadAll(body)
	if readErr != nil {
		return "", readErr
	}

	s.savedBody = string(bytes)

	if s.err != nil {
		return "", s.err
	}

	if s.returnURL == "" {
		return "/uploads/test/file.bin", nil
	}

	return s.returnURL, nil
}

func TestUploadUseCaseUploadRejectsInvalidType(t *testing.T) {
	t.Parallel()

	uc := NewUploadUseCase(&uploadStorageStub{}, 20<<20)

	_, err := uc.Upload(context.Background(), domain.UploadParams{
		Type:        domain.UploadType("invalid"),
		Filename:    "photo.png",
		ContentType: "image/png",
		SizeBytes:   10,
		Body:        strings.NewReader("data"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUploadUseCaseUploadRejectsTooLargeFile(t *testing.T) {
	t.Parallel()

	uc := NewUploadUseCase(&uploadStorageStub{}, 5)

	_, err := uc.Upload(context.Background(), domain.UploadParams{
		Type:        domain.UploadTypeFile,
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		SizeBytes:   6,
		Body:        strings.NewReader("content"),
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if got := err.Error(); !strings.Contains(got, "file is too large") {
		t.Fatalf("expected size validation message, got %q", got)
	}
}

func TestUploadUseCaseUploadStoresFile(t *testing.T) {
	t.Parallel()

	stub := &uploadStorageStub{returnURL: "/uploads/image/2026-05-01/file.png"}
	uc := NewUploadUseCase(stub, 20<<20)

	result, err := uc.Upload(context.Background(), domain.UploadParams{
		Type:        domain.UploadTypeImage,
		Filename:    "photo.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Body:        strings.NewReader("data"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !stub.saveCalled {
		t.Fatalf("expected storage save call")
	}

	if stub.savedType != domain.UploadTypeImage {
		t.Fatalf("unexpected saved type: %s", stub.savedType)
	}

	if stub.savedName != "photo.png" {
		t.Fatalf("unexpected saved name: %s", stub.savedName)
	}

	if stub.savedBody != "data" {
		t.Fatalf("unexpected saved body: %q", stub.savedBody)
	}

	if result.URL != "/uploads/image/2026-05-01/file.png" {
		t.Fatalf("unexpected url: %s", result.URL)
	}

	if result.Filename != "photo.png" {
		t.Fatalf("unexpected filename: %s", result.Filename)
	}

	if result.SizeBytes != 4 {
		t.Fatalf("unexpected size: %d", result.SizeBytes)
	}
}
