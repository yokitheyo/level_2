package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
)

func main() {
	var opts Options

	flag.IntVar(&opts.keyField, "k", 0, "sort by field number (1-based)")
	flag.BoolVar(&opts.numeric, "n", false, "sort numerically")
	flag.BoolVar(&opts.reverse, "r", false, "reverse sort order")
	flag.BoolVar(&opts.unique, "u", false, "output unique lines only")
	flag.BoolVar(&opts.month, "M", false, "sort by month name")
	flag.BoolVar(&opts.ignoreBlanks, "b", false, "ignore trailing blanks")
	flag.BoolVar(&opts.checkSorted, "c", false, "check if input is sorted")
	flag.BoolVar(&opts.humanNumeric, "h", false, "sort human-readable numbers")
	flag.StringVar(&opts.fieldSep, "t", "\t", "field separator")
	flag.Parse()

	log.Printf("Starting sort with options: %+v", opts)

	if opts.keyField < 0 {
		fmt.Fprintln(os.Stderr, "sort: invalid field number")
		os.Exit(1)
	}

	var r io.Reader = os.Stdin
	if flag.NArg() > 0 {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "sort: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		r = file
	}

	const externalSortThreshold = 100 * 1024 * 1024

	useExternalSort := false
	if flag.NArg() > 0 {
		fileInfo, err := os.Stat(flag.Arg(0))
		if err == nil && !fileInfo.IsDir() && fileInfo.Size() > externalSortThreshold {
			useExternalSort = true
			log.Printf("Using external sort for large file: %s (%d bytes)", flag.Arg(0), fileInfo.Size())
		}
	}

	if opts.checkSorted {
		lines, err := readLines(r, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sort: %v\n", err)
			os.Exit(1)
		}

		if checkSorted(lines, opts) {
			os.Exit(0)
		}
		os.Exit(1)
	} else if useExternalSort {
		if err := externalSort(r, os.Stdout, opts); err != nil {
			fmt.Fprintf(os.Stderr, "sort: %v\n", err)
			os.Exit(1)
		}
	} else {
		lines, err := readLines(r, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sort: %v\n", err)
			os.Exit(1)
		}

		if opts.reverse {
			sort.Slice(lines, func(i, j int) bool {
				return compareLines(lines[i], lines[j], opts) > 0
			})
		} else {
			sort.Slice(lines, func(i, j int) bool {
				return compareLines(lines[i], lines[j], opts) < 0
			})
		}

		writeLines(os.Stdout, lines, opts)
	}

	log.Printf("Sort completed successfully")
}
