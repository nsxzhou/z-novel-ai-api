// Package dto 提供 HTTP 层数据传输对象
package dto

// SearchRequest 检索请求
type SearchRequest struct {
	ProjectID        string           `json:"project_id" binding:"required"`
	Query            string           `json:"query" binding:"required,max=5000"`
	CurrentStoryTime int64            `json:"current_story_time,omitempty"`
	TopK             int              `json:"top_k,omitempty"`
	Options          *RetrievalOption `json:"options,omitempty"`
}

// RetrievalOption 检索选项
type RetrievalOption struct {
	VectorWeight    float64  `json:"vector_weight,omitempty"`  // 默认 0.7
	KeywordWeight   float64  `json:"keyword_weight,omitempty"` // 默认 0.3
	IncludeEntities bool     `json:"include_entities,omitempty"`
	IncludeEvents   bool     `json:"include_events,omitempty"`
	EntityTypes     []string `json:"entity_types,omitempty"`
}

// DebugRetrievalRequest 调试检索请求
type DebugRetrievalRequest struct {
	ProjectID        string           `json:"project_id" binding:"required"`
	Query            string           `json:"query" binding:"required,max=5000"`
	CurrentStoryTime int64            `json:"current_story_time,omitempty"`
	TopK             int              `json:"top_k,omitempty"`
	Options          *RetrievalOption `json:"options,omitempty"`
	IncludeScores    bool             `json:"include_scores,omitempty"`
	IncludeEmbedding bool             `json:"include_embedding,omitempty"`
}

// SearchResponse 检索响应
type SearchResponse struct {
	Segments []*ContextSegment `json:"segments"`
	Entities []*EntityRef      `json:"entities,omitempty"`
	Metadata *RetrievalMeta    `json:"metadata,omitempty"`
}

// ContextSegment 上下文片段
type ContextSegment struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	ChapterID string  `json:"chapter_id,omitempty"`
	StoryTime int64   `json:"story_time,omitempty"`
	Score     float64 `json:"score"`
	Source    string  `json:"source"` // vector, keyword, time
}

// EntityRef 实体引用
type EntityRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// RetrievalMeta 检索元数据
type RetrievalMeta struct {
	TotalSegments       int   `json:"total_segments"`
	TotalEntities       int   `json:"total_entities"`
	RetrievalDurationMs int64 `json:"retrieval_duration_ms"`
}

// DebugRetrievalResponse 调试检索响应
type DebugRetrievalResponse struct {
	SearchResponse
	QueryEmbedding []float32  `json:"query_embedding,omitempty"`
	DebugInfo      *DebugInfo `json:"debug_info,omitempty"`
}

// DebugInfo 调试信息
type DebugInfo struct {
	VectorSearchTime   int64 `json:"vector_search_time_ms"`
	KeywordSearchTime  int64 `json:"keyword_search_time_ms"`
	FusionTime         int64 `json:"fusion_time_ms"`
	TotalCandidates    int   `json:"total_candidates"`
	FilteredCandidates int   `json:"filtered_candidates"`
}
