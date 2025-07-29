package wget

import (
	"bytes"
	"net/url"
	"path/filepath"

	"golang.org/x/net/html"
)

// Извлекает и заменяет ссылки в HTML
func processHTML(content []byte, baseURL, baseDir, currentPath string) ([]byte, []string) {
	var links []string
	doc, _ := html.Parse(bytes.NewReader(content))

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			processNode(n, &links, baseURL, baseDir, currentPath)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var buf bytes.Buffer
	html.Render(&buf, doc)
	return buf.Bytes(), links
}

// Обрабатывает HTML-тег
func processNode(n *html.Node, links *[]string, baseURL, baseDir, currentPath string) {
	attrs := map[string]string{
		"a":      "href",
		"link":   "href",
		"script": "src",
		"img":    "src",
		"iframe": "src",
	}

	attr, ok := attrs[n.Data]
	if !ok {
		return
	}

	for i, a := range n.Attr {
		if a.Key != attr {
			continue
		}

		absURL := resolveURL(baseURL, a.Val)
		if !isSameDomain(baseURL, absURL) {
			continue
		}

		localPath, _ := urlToFilePath(baseDir, absURL)
		relPath, _ := filepath.Rel(filepath.Dir(currentPath), localPath)
		n.Attr[i].Val = filepath.ToSlash(relPath)
		*links = append(*links, absURL)
	}
}

// Преобразует относительный URL в абсолютный
func resolveURL(baseURL, target string) string {
	base, _ := url.Parse(baseURL)
	rel, _ := url.Parse(target)
	return base.ResolveReference(rel).String()
}
