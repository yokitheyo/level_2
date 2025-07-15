package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type config struct {
	after      int
	before     int
	count      bool
	ignoreCase bool
	invert     bool
	fixed      bool
	lineNum    bool
	context    int
	pattern    string
	filename   string
}

type result struct {
	line    string
	num     int
	matched bool
}

func parseFlags() config {
	cfg := config{}
	flag.IntVar(&cfg.after, "A", 0, "print N lines after match")
	flag.IntVar(&cfg.before, "B", 0, "print N lines before match")
	flag.IntVar(&cfg.context, "C", 0, "print N lines before and after match (overrides A and B)")
	flag.BoolVar(&cfg.count, "c", false, "print count of matching lines")
	flag.BoolVar(&cfg.ignoreCase, "i", false, "ignore case")
	flag.BoolVar(&cfg.invert, "v", false, "invert match")
	flag.BoolVar(&cfg.fixed, "F", false, "fixed string match (no regex)")
	flag.BoolVar(&cfg.lineNum, "n", false, "print line numbers")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "pattern required")
		os.Exit(1)
	}
	cfg.pattern = args[0]
	if len(args) > 1 {
		cfg.filename = args[1]
	}

	if cfg.context > 0 {
		cfg.after = cfg.context
		cfg.before = cfg.context
	}
	return cfg
}

func readLines(cfg config) ([]string, error) {
	var scanner *bufio.Scanner
	if cfg.filename != "" {
		file, err := os.Open(cfg.filename)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func compileRegex(cfg config) (*regexp.Regexp, error) {
	if cfg.fixed {
		pattern := regexp.QuoteMeta(cfg.pattern)
		if cfg.ignoreCase {
			return regexp.Compile("(?i)" + pattern)
		}
		return regexp.Compile(pattern)
	}

	if cfg.ignoreCase {
		return regexp.Compile("(?i)" + cfg.pattern)
	}
	return regexp.Compile(cfg.pattern)
}

func processLines(lines []string, cfg config, re *regexp.Regexp) ([]result, []bool) {
	results := []result{}
	matches := make([]bool, len(lines))

	for i, line := range lines {
		matched := false
		if cfg.fixed {
			if cfg.ignoreCase {
				matched = strings.Contains(strings.ToLower(line), strings.ToLower(cfg.pattern))
			} else {
				matched = strings.Contains(line, cfg.pattern)
			}
		} else {
			matched = re.MatchString(line)
		}

		if cfg.invert {
			matched = !matched
		}
		matches[i] = matched
	}

	printLines := make([]bool, len(lines))
	for i, matched := range matches {
		if matched {
			start := max(0, i-cfg.before)
			for j := start; j < i; j++ {
				printLines[j] = true
			}

			printLines[i] = true
			end := min(len(lines)-1, i+cfg.after)
			for j := i + 1; j <= end; j++ {
				printLines[j] = true
			}
		}
	}

	for i := 0; i < len(lines); i++ {
		if printLines[i] {
			results = append(results, result{
				line:    lines[i],
				num:     i + 1,
				matched: matches[i],
			})
		}
	}

	return results, matches
}

func printResults(results []result, matches []bool, cfg config) {
	if cfg.count {
		count := 0
		for _, m := range matches {
			if m {
				count++
			}
		}
		fmt.Println(count)
		return
	}

	lastPrinted := -1
	for _, r := range results {
		if cfg.before > 0 || cfg.after > 0 || cfg.context > 0 {
			if lastPrinted >= 0 && r.num > lastPrinted+1 {
				fmt.Println("--")
			}
			lastPrinted = r.num
		}

		if cfg.lineNum {
			if r.matched {
				fmt.Printf("%d:%s\n", r.num, r.line)
			} else {
				fmt.Printf("%d-%s\n", r.num, r.line)
			}
		} else {
			fmt.Println(r.line)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	cfg := parseFlags()

	re, err := compileRegex(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid pattern: %v\n", err)
		os.Exit(1)
	}

	lines, err := readLines(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	results, matches := processLines(lines, cfg, re)
	printResults(results, matches, cfg)
}
