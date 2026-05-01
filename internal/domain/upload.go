package domain

import (
	"io"
	"strings"
)

type UploadType string

const (
	UploadTypeImage  UploadType = "image"
	UploadTypeVideo  UploadType = "video"
	UploadTypeFile   UploadType = "file"
	UploadTypeAvatar UploadType = "avatar"
)

func (t UploadType) IsValid() bool {
	switch t {
	case UploadTypeImage, UploadTypeVideo, UploadTypeFile, UploadTypeAvatar:
		return true
	default:
		return false
	}
}

func (t UploadType) String() string {
	return string(t)
}

func (t UploadType) RequiresImageContent() bool {
	switch t {
	case UploadTypeImage, UploadTypeAvatar:
		return true
	default:
		return false
	}
}

func (t UploadType) RequiresVideoContent() bool {
	return t == UploadTypeVideo
}

type UploadParams struct {
	Type        UploadType
	Filename    string
	ContentType string
	SizeBytes   int64
	Body        io.Reader
}

type Upload struct {
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
}

func NormalizeUploadType(value string) UploadType {
	return UploadType(strings.ToLower(strings.TrimSpace(value)))
}
