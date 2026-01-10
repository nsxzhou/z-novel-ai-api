package milvus

import (
	"context"

	"z-novel-ai-api/internal/application/retrieval"
)

type RetrievalVectorRepository struct {
	repo *Repository
}

func NewRetrievalVectorRepository(repo *Repository) *RetrievalVectorRepository {
	return &RetrievalVectorRepository{repo: repo}
}

var _ retrieval.VectorRepository = (*RetrievalVectorRepository)(nil)

func (r *RetrievalVectorRepository) EnsureStorySegmentsCollection(ctx context.Context) error {
	if r == nil || r.repo == nil {
		return retrieval.ErrVectorDisabled
	}
	return r.repo.EnsureStorySegmentsCollection(ctx)
}

func (r *RetrievalVectorRepository) SearchSegments(ctx context.Context, params *retrieval.VectorSearchParams) ([]*retrieval.VectorSearchResult, error) {
	if r == nil || r.repo == nil {
		return nil, retrieval.ErrVectorDisabled
	}
	if params == nil {
		return nil, nil
	}

	out, err := r.repo.SearchSegments(ctx, &SearchParams{
		TenantID:         params.TenantID,
		ProjectID:        params.ProjectID,
		QueryVector:      params.QueryVector,
		CurrentStoryTime: params.CurrentStoryTime,
		TopK:             params.TopK,
		SegmentTypes:     params.SegmentTypes,
	})
	if err != nil {
		return nil, err
	}

	results := make([]*retrieval.VectorSearchResult, 0, len(out))
	for i := range out {
		v := out[i]
		if v == nil {
			continue
		}
		results = append(results, &retrieval.VectorSearchResult{
			ID:          v.ID,
			Score:       v.Score,
			TextContent: v.TextContent,
			ChapterID:   v.ChapterID,
			StoryTime:   v.StoryTime,
		})
	}
	return results, nil
}

func (r *RetrievalVectorRepository) DeleteSegmentsByDocAndType(ctx context.Context, tenantID, projectID, docID, segmentType string) error {
	if r == nil || r.repo == nil {
		return retrieval.ErrVectorDisabled
	}
	return r.repo.DeleteSegmentsByChapterAndType(ctx, tenantID, projectID, docID, segmentType)
}

func (r *RetrievalVectorRepository) InsertSegments(ctx context.Context, tenantID, projectID string, segments []*retrieval.VectorStorySegment) error {
	if r == nil || r.repo == nil {
		return retrieval.ErrVectorDisabled
	}
	if len(segments) == 0 {
		return nil
	}

	out := make([]*StorySegment, 0, len(segments))
	for i := range segments {
		s := segments[i]
		if s == nil {
			continue
		}
		out = append(out, &StorySegment{
			ID:          s.ID,
			TenantID:    s.TenantID,
			ProjectID:   s.ProjectID,
			ChapterID:   s.DocID,
			StoryTime:   s.StoryTime,
			SegmentType: s.SegmentType,
			TextContent: s.TextContent,
			Vector:      s.Vector,
		})
	}
	return r.repo.InsertSegments(ctx, tenantID, projectID, out)
}
