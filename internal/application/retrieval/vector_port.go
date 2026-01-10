package retrieval

import "context"

// VectorRepository 定义应用层对“向量存储/检索”的最小依赖（port）。
// 由基础设施层提供具体实现（例如 Milvus）。
type VectorRepository interface {
	EnsureStorySegmentsCollection(ctx context.Context) error
	SearchSegments(ctx context.Context, params *VectorSearchParams) ([]*VectorSearchResult, error)
	DeleteSegmentsByDocAndType(ctx context.Context, tenantID, projectID, docID, segmentType string) error
	InsertSegments(ctx context.Context, tenantID, projectID string, segments []*VectorStorySegment) error
}

type VectorSearchParams struct {
	TenantID         string
	ProjectID        string
	QueryVector      []float32
	CurrentStoryTime int64
	TopK             int
	SegmentTypes     []string
}

type VectorSearchResult struct {
	ID          string
	Score       float32
	TextContent string
	ChapterID   string
	StoryTime   int64
}

type VectorStorySegment struct {
	ID          string
	TenantID    string
	ProjectID   string
	DocID       string
	StoryTime   int64
	SegmentType string
	TextContent string
	Vector      []float32
}
