package main

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed easy_solutions_text.json
var easySolutionsTextJSON []byte

var (
	easySolutionsTextOnce sync.Once
	easySolutionsTextMap  map[string]string
)

// SolutionsTextEasy returns a mapping from Woodpecker Easy ID (wpm_easy_001..wpm_easy_222) to solution text descriptions.
func SolutionsTextEasy() map[string]string {
	easySolutionsTextOnce.Do(func() {
		m := make(map[string]string)
		if err := json.Unmarshal(easySolutionsTextJSON, &m); err != nil {
			easySolutionsTextMap = m
			return
		}
		// Trim wpm_easy_222 if it contains spillover from next chapter
		if s, ok := m["wpm_easy_222"]; ok && strings.Contains(s, "Chapter 5") {
			if idx := strings.Index(s, "\nChapter 5"); idx != -1 {
				m["wpm_easy_222"] = strings.TrimSpace(s[:idx])
			}
		}
		easySolutionsTextMap = m
	})
	return easySolutionsTextMap
}
