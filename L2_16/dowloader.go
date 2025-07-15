package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Downloader отвечает за загрузку веб-контента
type Downloader struct {
	client    *http.Client
	userAgent string
}

// NewDownloader создает новый экземпляр загрузчика
func NewDownloader(timeout time.Duration, userAgent string) *Downloader {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
		},
	}

	return &Downloader{
		client:    client,
		userAgent: userAgent,
	}
}

// Download загружает контент по URL
func (d *Downloader) Download(targetURL string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Читаем контент
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Получаем тип контента
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = d.detectContentType(targetURL, content)
	}

	return content, contentType, nil
}

// detectContentType определяет тип контента по URL и содержимому
func (d *Downloader) detectContentType(targetURL string, content []byte) string {
	// Проверяем по расширению файла
	if strings.HasSuffix(targetURL, ".html") || strings.HasSuffix(targetURL, ".htm") {
		return "text/html"
	}
	if strings.HasSuffix(targetURL, ".css") {
		return "text/css"
	}
	if strings.HasSuffix(targetURL, ".js") {
		return "application/javascript"
	}
	if strings.HasSuffix(targetURL, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(targetURL, ".jpg") || strings.HasSuffix(targetURL, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(targetURL, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(targetURL, ".svg") {
		return "image/svg+xml"
	}

	// Проверяем содержимое
	if len(content) > 0 {
		contentStr := string(content[:min(len(content), 512)])
		if strings.Contains(strings.ToLower(contentStr), "<!doctype html") ||
			strings.Contains(strings.ToLower(contentStr), "<html") {
			return "text/html"
		}
	}

	return "application/octet-stream"
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
