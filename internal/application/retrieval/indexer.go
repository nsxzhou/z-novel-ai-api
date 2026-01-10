package retrieval

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/google/uuid"

	"z-novel-ai-api/internal/domain/entity"
)

const (
	defaultChunkSizeRunes    = 800
	defaultChunkOverlapRunes = 80
	defaultEmbeddingBatch    = 32
	defaultMaxJSONLeaves     = 800
)

type Indexer struct {
	embedder embedding.Embedder
	vector   VectorRepository

	embeddingBatchSize int
	chunkSizeRunes     int
	chunkOverlapRunes  int
}

func NewIndexer(embedder embedding.Embedder, vectorRepo VectorRepository, embeddingBatchSize int) *Indexer {
	bs := embeddingBatchSize
	if bs <= 0 {
		bs = defaultEmbeddingBatch
	}
	return &Indexer{
		embedder:           embedder,
		vector:             vectorRepo,
		embeddingBatchSize: bs,
		chunkSizeRunes:     defaultChunkSizeRunes,
		chunkOverlapRunes:  defaultChunkOverlapRunes,
	}
}

func (i *Indexer) Enabled() bool {
	return i != nil && i.embedder != nil && i.vector != nil
}

func (i *Indexer) ensureReady(ctx context.Context) error {
	if i == nil || i.vector == nil {
		return ErrVectorDisabled
	}
	return i.vector.EnsureStorySegmentsCollection(ctx)
}

func (i *Indexer) IndexChapter(ctx context.Context, tenantID, projectID string, chapter *entity.Chapter) error {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("tenant_id and project_id are required")
	}
	if chapter == nil {
		return fmt.Errorf("chapter is nil")
	}
	if strings.TrimSpace(chapter.ID) == "" {
		return fmt.Errorf("chapter.id is required")
	}
	if !i.Enabled() {
		return ErrVectorDisabled
	}
	if err := i.ensureReady(ctx); err != nil {
		return err
	}

	segmentType := "chapter"
	if err := i.vector.DeleteSegmentsByDocAndType(ctx, tenantID, projectID, chapter.ID, segmentType); err != nil {
		return err
	}

	content := strings.TrimSpace(chapter.ContentText)
	if content == "" {
		// 空正文不写索引；但会先执行删除以避免“旧分片残留”。
		return nil
	}

	chunks := splitByRunes(content, i.chunkSizeRunes, i.chunkOverlapRunes)
	if len(chunks) == 0 {
		return nil
	}

	embedInputs := make([]string, 0, len(chunks))
	segments := make([]*VectorStorySegment, 0, len(chunks))
	storyTime := chapter.StoryTimeStart
	if chapter.StoryTimeEnd > 0 {
		// 若存在 end，优先使用 end 作为“事件已发生”的上界
		storyTime = chapter.StoryTimeEnd
	}

	for _, chunk := range chunks {
		meta := SegmentMeta{
			DocType:      "chapter",
			ChapterID:    chapter.ID,
			ChapterTitle: strings.TrimSpace(chapter.Title),
			RefPath:      "/content_text",
		}
		textContent := encodeSegmentText(meta, strings.TrimSpace(chunk))

		embedText := strings.TrimSpace(chunk)
		if t := strings.TrimSpace(chapter.Title); t != "" {
			embedText = "章节标题：" + t + "\n" + embedText
		}

		embedInputs = append(embedInputs, embedText)
		segments = append(segments, &VectorStorySegment{
			ID:          uuid.NewString(),
			TenantID:    tenantID,
			ProjectID:   projectID,
			DocID:       chapter.ID,
			StoryTime:   storyTime,
			SegmentType: segmentType,
			TextContent: textContent,
		})
	}

	vectors, err := i.embedBatch(ctx, embedInputs)
	if err != nil {
		return err
	}
	for idx := range segments {
		segments[idx].Vector = vectors[idx]
	}
	return i.vector.InsertSegments(ctx, tenantID, projectID, segments)
}

func (i *Indexer) IndexArtifactJSON(ctx context.Context, tenantID, projectID string, artifactType entity.ArtifactType, artifactID string, content json.RawMessage) error {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("tenant_id and project_id are required")
	}
	if strings.TrimSpace(artifactID) == "" {
		return fmt.Errorf("artifact_id is required")
	}
	if strings.TrimSpace(string(artifactType)) == "" {
		return fmt.Errorf("artifact_type is required")
	}
	if !i.Enabled() {
		return ErrVectorDisabled
	}
	if err := i.ensureReady(ctx); err != nil {
		return err
	}

	segmentType := ArtifactSegmentType(artifactType)
	if segmentType == "" {
		return fmt.Errorf("unsupported artifact_type: %s", artifactType)
	}

	if err := i.vector.DeleteSegmentsByDocAndType(ctx, tenantID, projectID, artifactID, segmentType); err != nil {
		return err
	}
	if len(content) == 0 || strings.TrimSpace(string(content)) == "" {
		return nil
	}

	var obj any
	if err := json.Unmarshal(content, &obj); err != nil {
		return fmt.Errorf("invalid artifact json: %w", err)
	}

	leaves := make([]jsonLeaf, 0, 128)
	collectJSONLeaves(obj, "", &leaves, defaultMaxJSONLeaves)
	if len(leaves) == 0 {
		return nil
	}

	embedInputs := make([]string, 0, len(leaves))
	segments := make([]*VectorStorySegment, 0, len(leaves))

	for _, leaf := range leaves {
		if strings.TrimSpace(leaf.Text) == "" {
			continue
		}
		chunks := splitByRunes(leaf.Text, i.chunkSizeRunes, i.chunkOverlapRunes)
		if len(chunks) == 0 {
			continue
		}
		for _, chunk := range chunks {
			meta := SegmentMeta{
				DocType:      "artifact",
				ArtifactID:   artifactID,
				ArtifactType: string(artifactType),
				RefPath:      leaf.Path,
			}
			textContent := encodeSegmentText(meta, strings.TrimSpace(chunk))
			embedText := "构件类型：" + string(artifactType) + "\n路径：" + leaf.Path + "\n内容：" + strings.TrimSpace(chunk)

			embedInputs = append(embedInputs, embedText)
			segments = append(segments, &VectorStorySegment{
				ID:          uuid.NewString(),
				TenantID:    tenantID,
				ProjectID:   projectID,
				DocID:       artifactID,
				StoryTime:   0,
				SegmentType: segmentType,
				TextContent: textContent,
			})
		}
	}

	if len(segments) == 0 {
		return nil
	}

	vectors, err := i.embedBatch(ctx, embedInputs)
	if err != nil {
		return err
	}
	for idx := range segments {
		segments[idx].Vector = vectors[idx]
	}
	return i.vector.InsertSegments(ctx, tenantID, projectID, segments)
}

// ArtifactSegmentType 将 ArtifactType 映射为 Milvus segment_type（用于过滤/删除/检索）。
func ArtifactSegmentType(t entity.ArtifactType) string {
	switch t {
	case entity.ArtifactTypeNovelFoundation:
		return "artifact_novel_foundation"
	case entity.ArtifactTypeWorldview:
		return "artifact_worldview"
	case entity.ArtifactTypeCharacters:
		return "artifact_characters"
	case entity.ArtifactTypeOutline:
		return "artifact_outline"
	default:
		return ""
	}
}

func AllArtifactSegmentTypes() []string {
	return []string{
		ArtifactSegmentType(entity.ArtifactTypeNovelFoundation),
		ArtifactSegmentType(entity.ArtifactTypeWorldview),
		ArtifactSegmentType(entity.ArtifactTypeCharacters),
		ArtifactSegmentType(entity.ArtifactTypeOutline),
	}
}

type jsonLeaf struct {
	Path string
	Text string
}

func collectJSONLeaves(v any, path string, out *[]jsonLeaf, limit int) {
	if out == nil {
		return
	}
	if limit > 0 && len(*out) >= limit {
		return
	}

	switch vv := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(vv))
		for k := range vv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			collectJSONLeaves(vv[k], joinJSONPointer(path, k), out, limit)
			if limit > 0 && len(*out) >= limit {
				return
			}
		}
	case []any:
		for idx := range vv {
			collectJSONLeaves(vv[idx], joinJSONPointer(path, fmt.Sprintf("%d", idx)), out, limit)
			if limit > 0 && len(*out) >= limit {
				return
			}
		}
	case string:
		s := strings.TrimSpace(vv)
		if s != "" {
			*out = append(*out, jsonLeaf{Path: normalizeJSONPointer(path), Text: s})
		}
	case float64, bool, nil:
		// 仅在有明确路径时记录，避免空路径的“孤立值”污染。
		if strings.TrimSpace(path) == "" {
			return
		}
		b, _ := json.Marshal(vv)
		s := strings.TrimSpace(string(b))
		if s != "" && s != "null" {
			*out = append(*out, jsonLeaf{Path: normalizeJSONPointer(path), Text: s})
		}
	default:
		// 其他类型（理论上 json.Unmarshal 不会出现）
	}
}

func joinJSONPointer(base, token string) string {
	t := strings.ReplaceAll(token, "~", "~0")
	t = strings.ReplaceAll(t, "/", "~1")
	if strings.TrimSpace(base) == "" {
		return "/" + t
	}
	return base + "/" + t
}

func normalizeJSONPointer(p string) string {
	if strings.TrimSpace(p) == "" {
		return "/"
	}
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}

func (i *Indexer) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if i == nil || i.embedder == nil {
		return nil, ErrVectorDisabled
	}
	if len(texts) == 0 {
		return nil, nil
	}

	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += i.embeddingBatchSize {
		end := start + i.embeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		v64, err := i.embedder.EmbedStrings(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		for _, vec := range v64 {
			f32 := make([]float32, 0, len(vec))
			for _, x := range vec {
				f32 = append(f32, float32(x))
			}
			out = append(out, f32)
		}
	}
	return out, nil
}
