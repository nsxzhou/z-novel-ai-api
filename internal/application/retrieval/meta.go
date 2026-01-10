package retrieval

import (
	"encoding/json"
	"strings"
)

const segmentMetaPrefix = "@@meta:"

// SegmentMeta 是写入到 Milvus text_content 的结构化元信息（用于“结构化定位”）。
// 约定：仅用于读写自家写入的段落；不存在时应安全降级。
type SegmentMeta struct {
	DocType string `json:"doc_type,omitempty"` // chapter | artifact

	ChapterID    string `json:"chapter_id,omitempty"`
	ChapterTitle string `json:"chapter_title,omitempty"`

	ArtifactID   string `json:"artifact_id,omitempty"`
	ArtifactType string `json:"artifact_type,omitempty"` // worldview/characters/outline/novel_foundation

	RefPath string `json:"ref_path,omitempty"` // JSON Pointer（RFC6901）或近似路径
}

func encodeSegmentText(meta SegmentMeta, text string) string {
	b, _ := json.Marshal(meta)
	var sb strings.Builder
	sb.Grow(len(segmentMetaPrefix) + len(b) + 1 + len(text))
	sb.WriteString(segmentMetaPrefix)
	sb.Write(b)
	sb.WriteByte('\n')
	sb.WriteString(text)
	return sb.String()
}

func decodeSegmentText(textContent string) (SegmentMeta, string) {
	raw := strings.TrimSpace(textContent)
	if !strings.HasPrefix(raw, segmentMetaPrefix) {
		return SegmentMeta{}, raw
	}
	rest := strings.TrimPrefix(raw, segmentMetaPrefix)
	line, body, ok := strings.Cut(rest, "\n")
	if !ok {
		return SegmentMeta{}, raw
	}
	var meta SegmentMeta
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &meta); err != nil {
		return SegmentMeta{}, strings.TrimSpace(body)
	}
	return meta, strings.TrimSpace(body)
}
