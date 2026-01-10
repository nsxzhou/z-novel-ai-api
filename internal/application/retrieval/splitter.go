package retrieval

import "strings"

func splitByRunes(s string, maxRunes int, overlapRunes int) []string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return nil
	}
	if maxRunes <= 0 {
		return []string{raw}
	}
	if overlapRunes < 0 {
		overlapRunes = 0
	}
	runes := []rune(raw)
	if len(runes) <= maxRunes {
		return []string{raw}
	}
	step := maxRunes - overlapRunes
	if step <= 0 {
		step = maxRunes
	}

	out := make([]string, 0, (len(runes)/step)+1)
	for start := 0; start < len(runes); start += step {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end >= len(runes) {
			break
		}
	}
	return out
}
