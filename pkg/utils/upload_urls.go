package utils

import "strings"

func BuildUploadPath(filePath string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(filePath, "\\", "/"))
	if normalized == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(normalized, "http://"), strings.HasPrefix(normalized, "https://"):
		return normalized
	case strings.HasPrefix(normalized, "/uploads/"):
		return normalized
	case strings.HasPrefix(normalized, "uploads/"):
		return "/" + normalized
	default:
		return "/uploads/" + strings.TrimLeft(normalized, "/")
	}
}

func BuildUploadURL(baseURL, filePath string) string {
	uploadPath := BuildUploadPath(filePath)
	if uploadPath == "" || strings.HasPrefix(uploadPath, "http://") || strings.HasPrefix(uploadPath, "https://") {
		return uploadPath
	}

	cleanBaseURL := strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if cleanBaseURL == "" {
		return uploadPath
	}

	return cleanBaseURL + uploadPath
}
