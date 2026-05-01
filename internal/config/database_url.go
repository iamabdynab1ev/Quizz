package config

import (
	"fmt"
	"net/url"
	"strings"
)

func DatabaseNameFromURL(databaseURL string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil {
		return "", fmt.Errorf("config parse database url: %w", err)
	}

	databaseName := strings.TrimPrefix(parsedURL.Path, "/")
	if databaseName == "" {
		return "", fmt.Errorf("config database url path is empty")
	}

	return databaseName, nil
}

func DatabaseAdminURL(databaseURL, adminDatabase string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil {
		return "", fmt.Errorf("config parse database url: %w", err)
	}

	adminDatabase = strings.TrimSpace(adminDatabase)
	if adminDatabase == "" {
		return "", fmt.Errorf("config admin database is empty")
	}

	parsedURL.Path = "/" + adminDatabase

	return parsedURL.String(), nil
}
