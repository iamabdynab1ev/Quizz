package validation

import (
	"bytes"
	"strings"
	"testing"

	"mime/multipart"
)

func TestValidateFile_AllowsDocxDetectedAsZip(t *testing.T) {
	data := []byte{'P', 'K', 0x03, 0x04, 0x14, 0x00}
	fileHeader := &multipart.FileHeader{
		Filename: "test.docx",
		Size:     int64(len(data)),
	}

	err := ValidateFile(fileHeader, bytes.NewReader(data), "order_document")
	if err != nil {
		t.Fatalf("expected docx file to be allowed, got %v", err)
	}
}

func TestValidateFile_RejectsZipFile(t *testing.T) {
	data := []byte{'P', 'K', 0x03, 0x04, 0x14, 0x00}
	fileHeader := &multipart.FileHeader{
		Filename: "archive.zip",
		Size:     int64(len(data)),
	}

	err := ValidateFile(fileHeader, bytes.NewReader(data), "order_document")
	if err == nil {
		t.Fatal("expected zip file to be rejected")
	}

	if !strings.Contains(err.Error(), "application/zip") {
		t.Fatalf("expected zip mime in error, got %v", err)
	}
}
