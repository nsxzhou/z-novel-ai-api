// Package storyutil 提供 story 应用层内部共享的工具函数。
package storyutil

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// ExtractJSONObject 尝试从一段可能包含"前后缀噪音"的文本中提取顶层 JSON（对象或数组）。
// 约定：若无法确认 JSON 有效性，则回退为原始输入（trim 后）。
func ExtractJSONObject(s string) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return raw
	}

	objStart := strings.Index(raw, "{")
	arrStart := strings.Index(raw, "[")
	start := -1
	end := -1
	switch {
	case objStart >= 0 && (arrStart < 0 || objStart < arrStart):
		start = objStart
		end = strings.LastIndex(raw, "}")
	case arrStart >= 0:
		start = arrStart
		end = strings.LastIndex(raw, "]")
	}
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err == nil {
		if d, ok := tok.(json.Delim); ok && (d == '{' || d == '[') {
			return raw
		}
	}

	dec = json.NewDecoder(strings.NewReader(raw))
	for {
		_, e := dec.Token()
		if e != nil {
			if errors.Is(e, io.EOF) {
				break
			}
			return strings.TrimSpace(s)
		}
	}
	return raw
}

// TruncateByRunes 按 rune 数量截断字符串。
func TruncateByRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	n := 0
	for i := range s {
		if n == maxRunes {
			return s[:i]
		}
		n++
	}
	return s
}
