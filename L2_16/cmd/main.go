package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yokitheyo/level_2/L2_16/wget"
)

func main() {
	url := flag.String("url", "", "URL for download")
	depth := flag.Int("depth", 0, "recursion depth")
	workers := flag.Int("workers", 5, "workers count")
	flag.Parse()

	if *url == "" {
		fmt.Println("URL required")
		os.Exit(1)
	}

	baseDir := filepath.Join(".", filepath.Base(*url))
	dl := wget.NewDownloader(*url, *depth, *workers, baseDir)
	err := dl.Start()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}
