package node

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// ExtractJSONObject 尝试从模型输出中截取“第一个完整 JSON 对象/数组”。
// 这是一个容错逻辑：模型可能会在 JSON 前后夹杂多余文本。
func ExtractJSONObject(s string) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return raw
	}

	// 如果模型输出夹杂了其它文本，尽量截取第一个 JSON 值（对象/数组）。
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

	// 简单校验：确保至少能被 Decoder 消费到一个 JSON 起始。
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err == nil {
		if d, ok := tok.(json.Delim); ok && (d == '{' || d == '[') {
			return raw
		}
	}

	// 最后兜底：尝试读取到 EOF 为止，避免调用方误用。
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
