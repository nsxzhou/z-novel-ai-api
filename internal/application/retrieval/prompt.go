package retrieval

import (
	"fmt"
	"strings"
)

// BuildPromptContext 将召回结果格式化为可直接注入 Prompt 的块。
// 约束：尽量短，避免把 score 等调试信息塞进 Prompt。
func BuildPromptContext(segments []Segment, maxSegments int, maxRunesPerSegment int) string {
	if len(segments) == 0 {
		return ""
	}
	if maxSegments <= 0 {
		maxSegments = 10
	}
	if maxRunesPerSegment <= 0 {
		maxRunesPerSegment = 400
	}

	n := len(segments)
	if n > maxSegments {
		n = maxSegments
	}

	lines := make([]string, 0, n+2)
	lines = append(lines, "【召回上下文（可能为空）】")
	for i := 0; i < n; i++ {
		s := segments[i]
		ref := ""
		switch strings.TrimSpace(s.DocType) {
		case "artifact":
			ref = fmt.Sprintf("Artifact:%s %s", strings.TrimSpace(s.ArtifactType), strings.TrimSpace(s.RefPath))
		case "chapter":
			title := strings.TrimSpace(s.ChapterTitle)
			if title == "" {
				title = strings.TrimSpace(s.ChapterID)
			}
			ref = fmt.Sprintf("Chapter:%s", title)
		default:
			ref = "Context"
		}

		txt := compactOneLine(s.Text)
		txt = truncateRunes(txt, maxRunesPerSegment)
		if strings.TrimSpace(txt) == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("[%d] (%s) %s", i+1, ref, txt))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func compactOneLine(s string) string {
	out := strings.ReplaceAll(s, "\r\n", "\n")
	out = strings.ReplaceAll(out, "\r", "\n")
	out = strings.ReplaceAll(out, "\n", " ")
	out = strings.TrimSpace(out)
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}
