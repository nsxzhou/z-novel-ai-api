package retrieval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/embedding"

	"z-novel-ai-api/internal/domain/repository"
	"z-novel-ai-api/internal/infrastructure/persistence/milvus"
)

type Engine struct {
	embedder embedding.Embedder
	vector   *milvus.Repository
	entity   repository.EntityRepository

	embeddingBatchSize int
}

func NewEngine(embedder embedding.Embedder, vectorRepo *milvus.Repository, entityRepo repository.EntityRepository, embeddingBatchSize int) *Engine {
	bs := embeddingBatchSize
	if bs <= 0 {
		bs = defaultEmbeddingBatch
	}
	return &Engine{
		embedder:            embedder,
		vector:              vectorRepo,
		entity:              entityRepo,
		embeddingBatchSize:  bs,
	}
}

func (e *Engine) Enabled() bool {
	return e != nil && e.embedder != nil && e.vector != nil
}

func (e *Engine) ensureReady(ctx context.Context) error {
	if e == nil || e.vector == nil {
		return ErrVectorDisabled
	}
	return e.vector.EnsureStorySegmentsCollection(ctx)
}

func (e *Engine) Search(ctx context.Context, in SearchInput) (*SearchOutput, error) {
	return e.search(ctx, in, false)
}

func (e *Engine) DebugSearch(ctx context.Context, in SearchInput) (*SearchOutput, error) {
	return e.search(ctx, in, true)
}

func (e *Engine) search(ctx context.Context, in SearchInput, forceDebug bool) (*SearchOutput, error) {
	if in.TopK <= 0 {
		in.TopK = 10
	}
	if in.TopK > 50 {
		in.TopK = 50
	}
	in.Query = strings.TrimSpace(in.Query)
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	if in.TenantID == "" || in.ProjectID == "" {
		return nil, fmt.Errorf("tenant_id and project_id are required")
	}
	if in.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	out := &SearchOutput{
		Segments: nil,
		Entities: nil,
	}

	var dbg *DebugInfo
	if forceDebug {
		dbg = &DebugInfo{}
	}

	// 1) 向量召回（可降级）
	if e.Enabled() {
		if err := e.ensureReady(ctx); err != nil {
			out.DisabledReason = err.Error()
		} else {
			start := time.Now()
			emb, err := e.embedQuery(ctx, in.Query)
			if err != nil {
				out.DisabledReason = err.Error()
			} else {
				if in.IncludeEmbedding {
					out.QueryEmbedding = emb
				}

				results, err := e.vector.SearchSegments(ctx, &milvus.SearchParams{
					TenantID:         in.TenantID,
					ProjectID:        in.ProjectID,
					QueryVector:      emb,
					CurrentStoryTime: in.CurrentStoryTime,
					TopK:             in.TopK,
					SegmentTypes:     in.SegmentTypes,
				})
				if err != nil {
					out.DisabledReason = err.Error()
				} else {
					out.Segments = make([]Segment, 0, len(results))
					for _, r := range results {
						if r == nil {
							continue
						}
						meta, text := decodeSegmentText(r.TextContent)
						seg := Segment{
							ID:     strings.TrimSpace(r.ID),
							Text:   strings.TrimSpace(text),
							Score:  1 - float64(r.Score), // 将“距离”转换为更直观的相似度（COSINE: distance=1-cos）
							Source: "vector",

							DocType:       strings.TrimSpace(meta.DocType),
							ChapterID:     strings.TrimSpace(meta.ChapterID),
							ChapterTitle:  strings.TrimSpace(meta.ChapterTitle),
							StoryTime:     r.StoryTime,
							ArtifactID:    strings.TrimSpace(meta.ArtifactID),
							ArtifactType:  strings.TrimSpace(meta.ArtifactType),
							RefPath:       strings.TrimSpace(meta.RefPath),
						}

						// 兼容：历史数据可能没有 meta，回退使用 Milvus 字段
						if seg.DocType == "" && strings.TrimSpace(r.ChapterID) != "" {
							seg.DocType = "chapter"
						}
						if seg.DocType == "chapter" && seg.ChapterID == "" {
							seg.ChapterID = strings.TrimSpace(r.ChapterID)
						}
						if seg.DocType == "artifact" && seg.ArtifactID == "" {
							seg.ArtifactID = strings.TrimSpace(r.ChapterID)
						}

						out.Segments = append(out.Segments, seg)
					}
					if dbg != nil {
						dbg.VectorSearchTimeMs = time.Since(start).Milliseconds()
						dbg.TotalCandidates = len(out.Segments)
						dbg.FilteredCandidates = len(out.Segments)
					}
				}
			}
		}
	} else {
		out.DisabledReason = ErrVectorDisabled.Error()
	}

	// 2) 结构化定位：实体名称搜索（可选）
	if in.IncludeEntities && e != nil && e.entity != nil {
		start := time.Now()
		entities, err := e.entity.SearchByName(ctx, in.ProjectID, in.Query, in.TopK)
		if err == nil && len(entities) > 0 {
			out.Entities = make([]EntityRef, 0, len(entities))
			for _, ent := range entities {
				if ent == nil {
					continue
				}
				out.Entities = append(out.Entities, EntityRef{
					ID:   ent.ID,
					Name: ent.Name,
					Type: string(ent.Type),
				})
			}
		}
		if dbg != nil {
			dbg.EntitySearchTimeMs = time.Since(start).Milliseconds()
		}
	}

	if dbg != nil {
		out.Debug = dbg
	}
	return out, nil
}

func (e *Engine) embedQuery(ctx context.Context, query string) ([]float32, error) {
	if e == nil || e.embedder == nil {
		return nil, ErrVectorDisabled
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("query is empty")
	}
	v64, err := e.embedder.EmbedStrings(ctx, []string{q})
	if err != nil {
		return nil, err
	}
	if len(v64) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}
	vec := v64[0]
	out := make([]float32, 0, len(vec))
	for _, x := range vec {
		out = append(out, float32(x))
	}
	return out, nil
}
