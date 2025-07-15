package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config содержит настройки для скрапера
type Config struct {
	StartURL      string
	MaxDepth      int
	MaxWorkers    int
	OutputDir     string
	Timeout       time.Duration
	UserAgent     string
	RespectRobots bool
}

// Scraper основная структура для веб-скрапинга
type Scraper struct {
	config     Config
	visited    map[string]bool
	visitedMux sync.RWMutex
	queue      chan URLItem
	wg         sync.WaitGroup
	downloader *Downloader
	parser     *Parser
	baseDomain string
}

// URLItem представляет URL для обработки
type URLItem struct {
	URL   string
	Depth int
}

// NewScraper создает новый экземпляр скрапера
func NewScraper(config Config) (*Scraper, error) {
	parsedURL, err := url.Parse(config.StartURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	// Создаем выходную директорию
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	downloader := NewDownloader(config.Timeout, config.UserAgent)
	parser := NewParser()

	return &Scraper{
		config:     config,
		visited:    make(map[string]bool),
		queue:      make(chan URLItem, 1000),
		downloader: downloader,
		parser:     parser,
		baseDomain: parsedURL.Host,
	}, nil
}

// Start запускает процесс скрапинга
func (s *Scraper) Start() error {
	log.Printf("Starting scraper for %s with max depth %d", s.config.StartURL, s.config.MaxDepth)

	// Запускаем воркеры
	for i := 0; i < s.config.MaxWorkers; i++ {
		go s.worker()
	}

	// Добавляем стартовый URL в очередь
	s.queue <- URLItem{URL: s.config.StartURL, Depth: 0}

	// Ждем завершения всех задач
	s.wg.Wait()
	close(s.queue)

	log.Println("Scraping completed successfully")
	return nil
}

// worker обрабатывает URL из очереди
func (s *Scraper) worker() {
	for item := range s.queue {
		s.wg.Add(1)
		s.processURL(item)
		s.wg.Done()
	}
}

// processURL обрабатывает один URL
func (s *Scraper) processURL(item URLItem) {
	// Проверяем, не посещали ли мы уже этот URL
	s.visitedMux.RLock()
	if s.visited[item.URL] {
		s.visitedMux.RUnlock()
		return
	}
	s.visitedMux.RUnlock()

	// Отмечаем как посещенный
	s.visitedMux.Lock()
	s.visited[item.URL] = true
	s.visitedMux.Unlock()

	// Проверяем домен
	if !s.isSameDomain(item.URL) {
		return
	}

	log.Printf("Processing: %s (depth: %d)", item.URL, item.Depth)

	// Скачиваем контент
	content, contentType, err := s.downloader.Download(item.URL)
	if err != nil {
		log.Printf("Failed to download %s: %v", item.URL, err)
		return
	}

	// Сохраняем файл
	filePath, err := s.saveFile(item.URL, content, contentType)
	if err != nil {
		log.Printf("Failed to save %s: %v", item.URL, err)
		return
	}

	log.Printf("Saved: %s -> %s", item.URL, filePath)

	// Если это HTML и мы не достигли максимальной глубины, парсим ссылки
	if strings.Contains(contentType, "text/html") && item.Depth < s.config.MaxDepth {
		s.parseAndQueueLinks(item.URL, string(content), item.Depth)
	}
}

// isSameDomain проверяет, принадлежит ли URL тому же домену
func (s *Scraper) isSameDomain(targetURL string) bool {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	return parsedURL.Host == s.baseDomain
}

// parseAndQueueLinks парсит HTML и добавляет найденные ссылки в очередь
func (s *Scraper) parseAndQueueLinks(baseURL, htmlContent string, currentDepth int) {
	links := s.parser.ParseLinks(htmlContent)

	for _, link := range links {
		absoluteURL := s.resolveURL(baseURL, link)
		if absoluteURL != "" {
			select {
			case s.queue <- URLItem{URL: absoluteURL, Depth: currentDepth + 1}:
			default:
				// Очередь переполнена, пропускаем
			}
		}
	}
}

// resolveURL преобразует относительные URL в абсолютные
func (s *Scraper) resolveURL(baseURL, link string) string {
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

// saveFile сохраняет контент в файл
func (s *Scraper) saveFile(targetURL string, content []byte, contentType string) (string, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", err
	}

	// Создаем путь для файла
	filePath := s.createFilePath(parsedURL, contentType)
	fullPath := filepath.Join(s.config.OutputDir, filePath)

	// Создаем директории
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Если это HTML, обрабатываем ссылки
	if strings.Contains(contentType, "text/html") {
		processedContent := s.parser.ProcessHTML(string(content), targetURL, s.config.OutputDir)
		content = []byte(processedContent)
	}

	// Сохраняем файл
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// createFilePath создает локальный путь для файла
func (s *Scraper) createFilePath(parsedURL *url.URL, contentType string) string {
	path := parsedURL.Path

	// Если путь пустой или заканчивается на /, добавляем index.html
	if path == "" || strings.HasSuffix(path, "/") {
		path = filepath.Join(path, "index.html")
	}

	// Если нет расширения и это HTML, добавляем .html
	if filepath.Ext(path) == "" && strings.Contains(contentType, "text/html") {
		path += ".html"
	}

	// Убираем ведущий слеш
	path = strings.TrimPrefix(path, "/")

	return path
}

func main() {
	var (
		startURL      = flag.String("url", "", "Starting URL to scrape")
		maxDepth      = flag.Int("depth", 2, "Maximum depth of recursion")
		maxWorkers    = flag.Int("workers", 5, "Maximum number of concurrent workers")
		outputDir     = flag.String("output", "scraped_site", "Output directory")
		timeout       = flag.Duration("timeout", 30*time.Second, "Request timeout")
		userAgent     = flag.String("user-agent", "WebScraper/1.0", "User-Agent header")
		respectRobots = flag.Bool("respect-robots", false, "Respect robots.txt")
	)
	flag.Parse()

	if *startURL == "" {
		fmt.Println("Usage: webscraper -url <URL> [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := Config{
		StartURL:      *startURL,
		MaxDepth:      *maxDepth,
		MaxWorkers:    *maxWorkers,
		OutputDir:     *outputDir,
		Timeout:       *timeout,
		UserAgent:     *userAgent,
		RespectRobots: *respectRobots,
	}

	scraper, err := NewScraper(config)
	if err != nil {
		log.Fatalf("Failed to create scraper: %v", err)
	}

	if err := scraper.Start(); err != nil {
		log.Fatalf("Scraping failed: %v", err)
	}
}
