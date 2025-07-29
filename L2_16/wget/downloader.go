package wget

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Downloader struct {
	baseURL  string
	maxDepth int
	workers  int
	baseDir  string
	visited  *sync.Map
	client   *http.Client
	queue    chan DownloadJob
	wg       sync.WaitGroup
}

func NewDownloader(url string, depth, workers int, baseDir string) *Downloader {
	return &Downloader{
		baseURL:  url,
		maxDepth: depth,
		workers:  workers,
		baseDir:  baseDir,
		visited:  &sync.Map{},
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
		queue: make(chan DownloadJob, 100),
	}
}

func (d *Downloader) Start() error {
	os.MkdirAll(d.baseDir, 0755)
	d.wg.Add(1)
	d.queue <- DownloadJob{URL: d.baseURL, Depth: d.maxDepth}

	for i := 0; i < d.workers; i++ {
		go d.worker()
	}

	d.wg.Wait()
	close(d.queue)
	return nil
}

func (d *Downloader) worker() {
	for job := range d.queue {
		d.processJob(job)
		d.wg.Done()
	}
}

func (d *Downloader) processJob(job DownloadJob) {
	if job.Depth < 0 {
		return
	}

	if _, loaded := d.visited.LoadOrStore(job.URL, true); loaded {
		return
	}

	filePath, err := urlToFilePath(d.baseDir, job.URL)
	if err != nil {
		log.Printf("path error: %s", job.URL)
		return
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Printf("error make dir: %v", err)
		return
	}

	content, contentType, err := d.fetchContent(job.URL)
	if err != nil {
		log.Printf("loading error %s: %v", job.URL, err)
		return
	}

	if strings.Contains(contentType, "text/html") {
		modifiedContent, links := processHTML(content, job.URL, d.baseDir, filePath)
		content = modifiedContent

		for _, link := range links {
			d.wg.Add(1)
			go func(l string) {
				d.queue <- DownloadJob{
					URL:   l,
					Depth: job.Depth - 1,
				}
			}(link)
		}
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		log.Printf("write error%s: %v", filePath, err)
	}
}

func (d *Downloader) fetchContent(url string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return content, resp.Header.Get("Content-Type"), nil
}
