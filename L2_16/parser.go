package main

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// Parser отвечает за парсинг HTML и извлечение ссылок
type Parser struct {
	linkRegex     *regexp.Regexp
	resourceRegex *regexp.Regexp
}

// NewParser создает новый экземпляр парсера
func NewParser() *Parser {
	// Регулярные выражения для поиска ссылок
	linkRegex := regexp.MustCompile(`(?i)<a\s+[^>]*href\s*=\s*["']([^"']+)["'][^>]*>`)

	// Регулярные выражения для поиска ресурсов
	resourceRegex := regexp.MustCompile(`(?i)(?:src|href)\s*=\s*["']([^"']+)["']`)

	return &Parser{
		linkRegex:     linkRegex,
		resourceRegex: resourceRegex,
	}
}

// ParseLinks извлекает все ссылки из HTML
func (p *Parser) ParseLinks(htmlContent string) []string {
	var links []string
	seen := make(map[string]bool)

	// Находим все ссылки <a href="...">
	matches := p.linkRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			link := strings.TrimSpace(match[1])
			if link != "" && !seen[link] && !isFragment(link) {
				links = append(links, link)
				seen[link] = true
			}
		}
	}

	// Находим все ресурсы (src, href)
	resourceMatches := p.resourceRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range resourceMatches {
		if len(match) > 1 {
			resource := strings.TrimSpace(match[1])
			if resource != "" && !seen[resource] && !isFragment(resource) && isDownloadableResource(resource) {
				links = append(links, resource)
				seen[resource] = true
			}
		}
	}

	// ДОБАВЛЕНО: ищем <img src=...>
	imgRegex := regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
	imgMatches := imgRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range imgMatches {
		if len(match) > 1 {
			img := strings.TrimSpace(match[1])
			if img != "" && !seen[img] && !isFragment(img) && isDownloadableResource(img) {
				links = append(links, img)
				seen[img] = true
			}
		}
	}

	// ДОБАВЛЕНО: ищем <script src=...>
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)
	scriptMatches := scriptRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range scriptMatches {
		if len(match) > 1 {
			script := strings.TrimSpace(match[1])
			if script != "" && !seen[script] && !isFragment(script) && isDownloadableResource(script) {
				links = append(links, script)
				seen[script] = true
			}
		}
	}

	// ДОБАВЛЕНО: ищем <link href=...>
	linkTagRegex := regexp.MustCompile(`(?i)<link[^>]+href=["']([^"']+)["']`)
	linkTagMatches := linkTagRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range linkTagMatches {
		if len(match) > 1 {
			linkHref := strings.TrimSpace(match[1])
			if linkHref != "" && !seen[linkHref] && !isFragment(linkHref) && isDownloadableResource(linkHref) {
				links = append(links, linkHref)
				seen[linkHref] = true
			}
		}
	}

	return links
}

// ProcessHTML обрабатывает HTML, заменяя ссылки на локальные
func (p *Parser) ProcessHTML(htmlContent, baseURL, outputDir string) string {
	// Заменяем все ссылки на ресурсы
	result := p.resourceRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		// Извлекаем URL из атрибута
		parts := regexp.MustCompile(`(?i)(?:src|href)\s*=\s*["']([^"']+)["']`).FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		originalURL := parts[1]

		// Пропускаем внешние ссылки и фрагменты
		if isExternalURL(originalURL) || isFragment(originalURL) {
			return match
		}

		// Преобразуем в абсолютный URL
		absoluteURL := p.resolveURL(baseURL, originalURL)
		if absoluteURL == "" {
			return match
		}

		// Создаем локальный путь
		localPath := p.createLocalPath(absoluteURL)
		if localPath == "" {
			return match
		}

		// Заменяем URL на локальный путь
		return strings.Replace(match, originalURL, localPath, 1)
	})

	return result
}

// resolveURL преобразует относительные URL в абсолютные
func (p *Parser) resolveURL(baseURL, link string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	ref, err := url.Parse(link)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(ref)
	return resolved.String()
}

// createLocalPath создает локальный путь для URL
func (p *Parser) createLocalPath(targetURL string) string {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}

	path := parsedURL.Path

	// Если путь пустой или заканчивается на /, добавляем index.html
	if path == "" || strings.HasSuffix(path, "/") {
		path = filepath.Join(path, "index.html")
	}

	// Если нет расширения, добавляем .html
	if filepath.Ext(path) == "" {
		path += ".html"
	}

	// Убираем ведущий слеш
	path = strings.TrimPrefix(path, "/")

	return path
}

// isFragment проверяет, является ли ссылка фрагментом (якорем)
func isFragment(link string) bool {
	return strings.HasPrefix(link, "#")
}

// isExternalURL проверяет, является ли URL внешним
func isExternalURL(link string) bool {
	return strings.HasPrefix(link, "http://") ||
		strings.HasPrefix(link, "https://") ||
		strings.HasPrefix(link, "//")
}

// isDownloadableResource проверяет, является ли ресурс загружаемым
func isDownloadableResource(resource string) bool {
	if isExternalURL(resource) {
		return false
	}

	// Исключаем протоколы, которые не являются HTTP
	if strings.Contains(resource, ":") && !strings.HasPrefix(resource, "/") {
		return false
	}

	// Исключаем некоторые типы ссылок
	if strings.HasPrefix(resource, "mailto:") ||
		strings.HasPrefix(resource, "tel:") ||
		strings.HasPrefix(resource, "javascript:") {
		return false
	}

	return true
}
