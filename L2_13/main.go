package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func parseFields(fieldStr string) (map[int]bool, error) {
	fields := make(map[int]bool)
	if fieldStr == "" {
		return nil, fmt.Errorf("fields string is empty")
	}

	parts := strings.Split(fieldStr, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}

			start, err := strconv.Atoi(bounds[0])
			if err != nil || start < 1 {
				return nil, fmt.Errorf("invalid start of range: %s", bounds[0])
			}

			end, err := strconv.Atoi(bounds[1])
			if err != nil || end < 1 {
				return nil, fmt.Errorf("invalid end of range: %s", bounds[1])
			}

			if start > end {
				return nil, fmt.Errorf("start > end in range: %s", part)
			}

			for i := start; i <= end; i++ {
				fields[i] = true
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil || num < 1 {
				return nil, fmt.Errorf("invalid field number: %s", part)
			}
			fields[num] = true
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no valid fields specified")
	}

	return fields, nil
}

func processLine(line string, fields map[int]bool, delim string, separated bool) (string, bool) {
	if separated && !strings.Contains(line, delim) {
		return "", false
	}

	parts := strings.Split(line, delim)
	var result []string

	maxField := 0
	for f := range fields {
		if f > maxField {
			maxField = f
		}
	}

	for i := 1; i <= maxField; i++ {
		if fields[i] {
			if i <= len(parts) {
				result = append(result, parts[i-1])
			}
		}
	}

	if len(result) == 0 {
		return "", false
	}

	return strings.Join(result, delim), true
}

func main() {
	var (
		fieldStr  string
		delim     string
		separated bool
	)

	flag.StringVar(&fieldStr, "f", "", "fields to select (e.g. 1,3-5)")
	flag.StringVar(&delim, "d", "\t", "field delimiter (default is tab)")
	flag.BoolVar(&separated, "s", false, "suppress lines without delimiter")
	flag.Parse()

	if fieldStr == "" {
		fmt.Fprintln(os.Stderr, "Error: flag -f is required")
		os.Exit(1)
	}

	fields, err := parseFields(fieldStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if output, ok := processLine(line, fields, delim, separated); ok {
			fmt.Println(output)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
