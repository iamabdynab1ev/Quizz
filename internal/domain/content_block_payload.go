package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type TextContentBlockPayload struct {
	Content MultiLangText `json:"content"`
}

type URLContentBlockPayload struct {
	URL   string        `json:"url"`
	Label MultiLangText `json:"label"`
}

type VideoProvider string

const (
	VideoProviderDirect  VideoProvider = "direct"
	VideoProviderYouTube VideoProvider = "youtube"
)

func (p VideoProvider) IsValid() bool {
	switch p {
	case VideoProviderDirect, VideoProviderYouTube:
		return true
	default:
		return false
	}
}

type VideoContentBlockPayload struct {
	URL             string        `json:"url"`
	Provider        VideoProvider `json:"provider"`
	DurationSeconds int           `json:"duration_seconds,omitempty"`
}

type PhotoContentBlockPayload struct {
	URL     string        `json:"url"`
	Caption MultiLangText `json:"caption,omitempty"`
}

type FileContentBlockPayload struct {
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

func ValidateContentBlockPayload(blockType ContentBlockType, raw json.RawMessage) error {
	switch blockType {
	case ContentBlockTypeText:
		var payload TextContentBlockPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return ValidationError("text payload must be a valid JSON object")
		}

		if err := payload.Content.ValidateRequired(); err != nil {
			return ValidationError("text payload requires content.ru and content.tj")
		}
	case ContentBlockTypeURL:
		var payload URLContentBlockPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return ValidationError("url payload must be a valid JSON object")
		}

		if err := validateHTTPURL(payload.URL); err != nil {
			return ValidationError("url payload requires a valid absolute http/https url")
		}

		if err := payload.Label.ValidateRequired(); err != nil {
			return ValidationError("url payload requires label.ru and label.tj")
		}
	case ContentBlockTypeVideo:
		var payload VideoContentBlockPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return ValidationError("video payload must be a valid JSON object")
		}

		if err := validateHTTPURL(payload.URL); err != nil {
			return ValidationError("video payload requires a valid absolute http/https url")
		}

		if !payload.Provider.IsValid() {
			return ValidationError("video payload provider must be one of: direct, youtube")
		}

		if payload.DurationSeconds < 0 {
			return ValidationError("video payload duration_seconds must be non-negative")
		}
	case ContentBlockTypePhoto:
		var payload PhotoContentBlockPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return ValidationError("photo payload must be a valid JSON object")
		}

		if err := validateHTTPURL(payload.URL); err != nil {
			return ValidationError("photo payload requires a valid absolute http/https url")
		}

		if err := validateOptionalMultiLangText(payload.Caption); err != nil {
			return ValidationError("photo payload caption must include both ru and tj when present")
		}
	case ContentBlockTypeFile:
		var payload FileContentBlockPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return ValidationError("file payload must be a valid JSON object")
		}

		if err := validateHTTPURL(payload.URL); err != nil {
			return ValidationError("file payload requires a valid absolute http/https url")
		}

		if strings.TrimSpace(payload.Filename) == "" {
			return ValidationError("file payload requires filename")
		}

		if payload.SizeBytes < 0 {
			return ValidationError("file payload size_bytes must be non-negative")
		}
	default:
		return ValidationError(fmt.Sprintf("unsupported content block type: %s", blockType))
	}

	return nil
}

func validateOptionalMultiLangText(value MultiLangText) error {
	if value.IsZero() {
		return nil
	}

	return value.ValidateRequired()
}

func validateHTTPURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return err
	}

	if parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("missing scheme or host")
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("unsupported scheme")
	}
}
