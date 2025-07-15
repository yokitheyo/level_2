package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
)

func unpack(s string) (string, error) {
	var result strings.Builder
	var prevRune rune
	var escaped bool

	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		current := runes[i]

		switch {
		case escaped:
			result.WriteRune(current)
			prevRune = current
			escaped = false

		case current == '\\':
			if escaped {
				result.WriteRune(current)
				prevRune = current
				escaped = false
			} else {
				escaped = true
			}

		case unicode.IsDigit(current):
			if prevRune == 0 {
				return "", errors.New("string starts with number or number after missing character")
			}

			numStr := string(current)
			for j := i + 1; j < len(runes) && unicode.IsDigit(runes[j]); j++ {
				numStr += string(runes[j])
				i = j
			}

			count, err := strconv.Atoi(numStr)
			if err != nil || count <= 0 {
				return "", errors.New("invalid digit sequence")
			}

			result.WriteString(strings.Repeat(string(prevRune), count-1))
			prevRune = 0

		default:
			result.WriteRune(current)
			prevRune = current
		}
	}

	if escaped {
		// Незавершенная escape-последовательность
		return "", errors.New("unterminated escape sequence")
	}
	return result.String(), nil
}

func main() {
	tests := []string{
		"a4bc2d5e",
		"abcd",
		"45",
		"",
		`qwe\4\5`,
		`qwe\45`,
	}

	for _, test := range tests {
		result, err := unpack(test)
		if err != nil {
			log.Printf("input: %q -> error: %v\n", test, err)
		} else {
			fmt.Printf("input: %q -> %q\n", test, result)
		}
	}
}
