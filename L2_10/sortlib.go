package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var monthMap = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// Options defines the set of sorting options used by the sort utility.
type Options struct {
	keyField     int
	numeric      bool
	reverse      bool
	unique       bool
	month        bool
	ignoreBlanks bool
	checkSorted  bool
	humanNumeric bool
	fieldSep     string
}

// Line represents a single input line with its original text and split fields.
type Line struct {
	original string
	fields   []string
}

func parseHumanNumeric(s string) (float64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) == 0 {
		return 0, nil
	}

	multiplier := 1.0
	if strings.HasSuffix(s, "k") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "k")
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	} else if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	}

	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return num * multiplier, nil
}

func compareLines(a, b Line, opts Options) int {
	fieldA := a.original
	fieldB := b.original

	if opts.keyField > 0 {
		if len(a.fields) >= opts.keyField {
			fieldA = a.fields[opts.keyField-1]
		} else {
			fieldA = ""
		}
		if len(b.fields) >= opts.keyField {
			fieldB = b.fields[opts.keyField-1]
		} else {
			fieldB = ""
		}
	}

	if opts.ignoreBlanks {
		fieldA = strings.TrimRight(fieldA, " \t")
		fieldB = strings.TrimRight(fieldB, " \t")
	}

	if opts.month {
		if len(fieldA) >= 3 && len(fieldB) >= 3 {
			monA, okA := monthMap[strings.ToLower(fieldA[:3])]
			monB, okB := monthMap[strings.ToLower(fieldB[:3])]
			if okA && okB {
				if monA != monB {
					return monA - monB
				}
				//return strings.Compare(fieldA, fieldB)
				return strings.Compare(a.original, b.original)
			}
		}
		//return strings.Compare(fieldA, fieldB)
		return strings.Compare(a.original, b.original)
	}

	if opts.humanNumeric {
		numA, errA := parseHumanNumeric(fieldA)
		numB, errB := parseHumanNumeric(fieldB)
		if errA == nil && errB == nil {
			switch {
			case numA < numB:
				return -1
			case numA > numB:
				return 1
			default:
				return 0
			}
		}
		return strings.Compare(fieldA, fieldB)
	}

	if opts.numeric {
		numA, errA := strconv.ParseFloat(fieldA, 64)
		numB, errB := strconv.ParseFloat(fieldB, 64)

		if errA == nil && errB == nil {
			switch {
			case numA < numB:
				return -1
			case numA > numB:
				return 1
			default:
				return 0
			}
		}
		return strings.Compare(fieldA, fieldB)
	}
	return strings.Compare(fieldA, fieldB)
}

func readLines(r io.Reader, opts Options) ([]Line, error) {
	var lines []Line
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		var fields []string
		if opts.fieldSep == "" {
			fields = strings.Fields(line)
		} else {
			fields = strings.Split(line, opts.fieldSep)
		}
		lines = append(lines, Line{original: line, fields: fields})
	}
	return lines, scanner.Err()
}

func checkSorted(lines []Line, opts Options) bool {
	for i := 1; i < len(lines); i++ {
		if opts.reverse {
			if compareLines(lines[i-1], lines[i], opts) < 0 {
				_, err := fmt.Fprintf(os.Stderr, "sort: disorder at line %d\n", i+1)
				if err != nil {
					log.Printf("failed to write to stderr: %v", err)
				}
				return false
			}
		} else {
			if compareLines(lines[i-1], lines[i], opts) > 0 {
				_, err := fmt.Fprintf(os.Stderr, "sort: disorder at line %d\n", i+1)
				if err != nil {
					log.Printf("failed to write to stderr: %v", err)
				}
				return false
			}
		}
	}
	return true
}

func writeLines(w io.Writer, lines []Line, opts Options) {
	seen := make(map[string]bool)
	for _, line := range lines {
		if opts.unique {
			if seen[line.original] {
				continue
			}
			seen[line.original] = true
		}
		if _, err := fmt.Fprintln(w, line.original); err != nil {
			log.Printf("write error: %v", err)
		}
	}
}

func externalSort(r io.Reader, w io.Writer, opts Options) error {
	const chunkSize = 10000

	tempDir, err := os.MkdirTemp("", "sort-chunks")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	var (
		chunkFiles []string
		chunkMu    sync.Mutex
		wg         sync.WaitGroup
		errChan    = make(chan error, 1)
		sem        = make(chan struct{}, runtime.NumCPU())
		chunkID    int
	)

	scanner := bufio.NewScanner(r)

	for {
		var chunk []Line
		for i := 0; i < chunkSize && scanner.Scan(); i++ {
			line := scanner.Text()
			fields := strings.Split(line, opts.fieldSep)
			chunk = append(chunk, Line{original: line, fields: fields})
		}
		if len(chunk) == 0 {
			break
		}

		id := chunkID
		chunkID++

		wg.Add(1)
		sem <- struct{}{}

		go func(chunk []Line, id int) {
			defer wg.Done()
			defer func() { <-sem }()

			if opts.reverse {
				sort.Slice(chunk, func(i, j int) bool {
					return compareLines(chunk[i], chunk[j], opts) > 0
				})
			} else {
				sort.Slice(chunk, func(i, j int) bool {
					return compareLines(chunk[i], chunk[j], opts) < 0
				})
			}

			chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk-%d", id))
			f, err := os.Create(chunkFile)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}

			for _, line := range chunk {
				if _, err := fmt.Fprintln(f, line.original); err != nil {
					f.Close()
					select {
					case errChan <- err:
					default:
					}
					return
				}
			}

			if err := f.Close(); err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}

			chunkMu.Lock()
			chunkFiles = append(chunkFiles, chunkFile)
			chunkMu.Unlock()
		}(chunk, id)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return err
	}

	if len(chunkFiles) == 1 {
		f, err := os.Open(chunkFiles[0])
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	}

	return mergeChunks(chunkFiles, w, opts)
}

func mergeChunks(chunkFiles []string, w io.Writer, opts Options) error {
	files := make([]*os.File, len(chunkFiles))
	scanners := make([]*bufio.Scanner, len(chunkFiles))
	lines := make([]Line, len(chunkFiles))
	valid := make([]bool, len(chunkFiles))

	for i, path := range chunkFiles {
		f, err := os.Open(path)
		if err != nil {
			for j := 0; j < i; j++ {
				files[j].Close()
			}
			return err
		}
		files[i] = f
		scanners[i] = bufio.NewScanner(f)

		if scanners[i].Scan() {
			line := scanners[i].Text()
			fields := strings.Split(line, opts.fieldSep)
			lines[i] = Line{original: line, fields: fields}
			valid[i] = true
		}
	}

	seen := make(map[string]bool)
	for {
		minIdx := -1
		for i, isValid := range valid {
			if !isValid {
				continue
			}

			if minIdx == -1 || (opts.reverse && compareLines(lines[minIdx], lines[i], opts) < 0) ||
				(!opts.reverse && compareLines(lines[minIdx], lines[i], opts) > 0) {
				minIdx = i
			}
		}

		if minIdx == -1 {
			break
		}

		if !opts.unique || !seen[lines[minIdx].original] {
			fmt.Fprintln(w, lines[minIdx].original)
			if opts.unique {
				seen[lines[minIdx].original] = true
			}
		}

		if scanners[minIdx].Scan() {
			line := scanners[minIdx].Text()
			fields := strings.Split(line, opts.fieldSep)
			lines[minIdx] = Line{original: line, fields: fields}
		} else {
			valid[minIdx] = false
		}
	}

	for _, f := range files {
		f.Close()
	}

	return nil
}
