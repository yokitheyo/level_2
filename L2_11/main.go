package main

import (
	"fmt"
	"sort"
	"strings"
)

func normalize(s string) string {
	s = strings.ToLower(s)
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	return string(runes)
}

func findAnagramGroups(words []string) map[string][]string {
	anagramMap := make(map[string][]string)

	for _, word := range words {
		normalized := normalize(word)
		anagramMap[normalized] = append(anagramMap[normalized], strings.ToLower(word))
	}

	result := make(map[string][]string)
	for _, group := range anagramMap {
		if len(group) < 2 {
			continue
		}
		sort.Strings(group)
		result[group[0]] = group
	}
	return result
}

func main() {
	input := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}
	result := findAnagramGroups(input)

	for key, group := range result {
		fmt.Printf("%q: %q\n", key, group)
	}
}
