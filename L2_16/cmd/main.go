package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yokitheyo/level_2/L2_16/wget"
)

func main() {
	url := flag.String("url", "", "URL для загрузки")
	depth := flag.Int("depth", 0, "Глубина рекурсии")
	workers := flag.Int("workers", 5, "Число параллельных загрузчиков")
	flag.Parse()

	if *url == "" {
		fmt.Println("Необходимо указать URL")
		os.Exit(1)
	}

	baseDir := filepath.Join(".", filepath.Base(*url))
	dl := wget.NewDownloader(*url, *depth, *workers, baseDir)
	err := dl.Start()
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		os.Exit(1)
	}
}
