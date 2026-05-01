package domain

import (
	"encoding/json"
	"testing"
)

func TestValidateContentBlockPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		blockType ContentBlockType
		payload   string
		wantErr   bool
	}{
		{
			name:      "text valid",
			blockType: ContentBlockTypeText,
			payload:   `{"content":{"ru":"Текст","tj":"Матн"}}`,
		},
		{
			name:      "text missing locale",
			blockType: ContentBlockTypeText,
			payload:   `{"content":{"ru":"Текст","tj":""}}`,
			wantErr:   true,
		},
		{
			name:      "url valid",
			blockType: ContentBlockTypeURL,
			payload:   `{"url":"https://example.com","label":{"ru":"Ссылка","tj":"Пайванд"}}`,
		},
		{
			name:      "url invalid scheme",
			blockType: ContentBlockTypeURL,
			payload:   `{"url":"ftp://example.com","label":{"ru":"Ссылка","tj":"Пайванд"}}`,
			wantErr:   true,
		},
		{
			name:      "video valid",
			blockType: ContentBlockTypeVideo,
			payload:   `{"url":"https://example.com/video.mp4","provider":"direct","duration_seconds":120}`,
		},
		{
			name:      "video invalid provider",
			blockType: ContentBlockTypeVideo,
			payload:   `{"url":"https://example.com/video.mp4","provider":"vimeo"}`,
			wantErr:   true,
		},
		{
			name:      "photo valid without caption",
			blockType: ContentBlockTypePhoto,
			payload:   `{"url":"https://example.com/image.jpg"}`,
		},
		{
			name:      "file valid",
			blockType: ContentBlockTypeFile,
			payload:   `{"url":"https://example.com/file.pdf","filename":"file.pdf","size_bytes":1024}`,
		},
		{
			name:      "file missing filename",
			blockType: ContentBlockTypeFile,
			payload:   `{"url":"https://example.com/file.pdf"}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateContentBlockPayload(tt.blockType, json.RawMessage(tt.payload))
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
