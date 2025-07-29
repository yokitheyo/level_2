package wget

import (
	"net/url"
	"path/filepath"
	"strings"
)

// Преобразует URL в локальный путь
func urlToFilePath(baseDir, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	path := u.Path
	if path == "" {
		path = "/index.html"
	} else if strings.HasSuffix(path, "/") {
		path += "index.html"
	}

	// Формируем путь: baseDir + host + path
	fullPath := filepath.Join(baseDir, u.Host, filepath.Clean(path))
	return filepath.Clean(fullPath), nil
}

// Проверяет, принадлежит ли URL тому же домену
func isSameDomain(baseURL, targetURL string) bool {
	base, _ := url.Parse(baseURL)
	target, _ := url.Parse(targetURL)
	return base.Hostname() == target.Hostname()
}
