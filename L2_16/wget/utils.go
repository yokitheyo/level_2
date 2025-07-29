package wget

import (
	"net/url"
	"path/filepath"
	"strings"
)

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

	fullPath := filepath.Join(baseDir, u.Host, filepath.Clean(path))
	return filepath.Clean(fullPath), nil
}

func isSameDomain(baseURL, targetURL string) bool {
	base, _ := url.Parse(baseURL)
	target, _ := url.Parse(targetURL)
	return base.Hostname() == target.Hostname()
}
