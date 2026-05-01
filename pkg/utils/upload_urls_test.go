package utils

import "testing"

func TestBuildUploadPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{name: "relative storage path", filePath: "orders/2026/04/27/file.docx", want: "/uploads/orders/2026/04/27/file.docx"},
		{name: "already uploads path", filePath: "/uploads/orders/2026/04/27/file.docx", want: "/uploads/orders/2026/04/27/file.docx"},
		{name: "uploads path without leading slash", filePath: "uploads/orders/2026/04/27/file.docx", want: "/uploads/orders/2026/04/27/file.docx"},
		{name: "windows path separators", filePath: "orders\\2026\\04\\27\\file.docx", want: "/uploads/orders/2026/04/27/file.docx"},
		{name: "absolute url passthrough", filePath: "https://files.example.com/orders/file.docx", want: "https://files.example.com/orders/file.docx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildUploadPath(tt.filePath); got != tt.want {
				t.Fatalf("BuildUploadPath(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestBuildUploadURL(t *testing.T) {
	const serverBaseURL = "https://api.example.com/"

	got := BuildUploadURL(serverBaseURL, "orders/2026/04/27/file.docx")
	want := "https://api.example.com/uploads/orders/2026/04/27/file.docx"
	if got != want {
		t.Fatalf("BuildUploadURL() = %q, want %q", got, want)
	}
}
