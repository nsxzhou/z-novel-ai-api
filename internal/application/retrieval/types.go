package retrieval

// SearchInput 本地检索输入。
type SearchInput struct {
	TenantID         string
	ProjectID        string
	Query            string
	CurrentStoryTime int64
	TopK             int

	// SegmentTypes 为空表示不过滤；非空则仅检索指定 segment_type。
	SegmentTypes []string

	IncludeEntities  bool
	IncludeEmbedding bool
}

type Segment struct {
	ID     string
	Text   string
	Score  float64
	Source string

	DocType string

	ChapterID    string
	ChapterTitle string
	StoryTime    int64

	ArtifactID   string
	ArtifactType string
	RefPath      string
}

type EntityRef struct {
	ID   string
	Name string
	Type string
}

type DebugInfo struct {
	VectorSearchTimeMs int64
	EntitySearchTimeMs int64
	TotalCandidates    int
	FilteredCandidates int
}

type SearchOutput struct {
	Segments []Segment
	Entities []EntityRef

	DisabledReason string
	QueryEmbedding []float32
	Debug          *DebugInfo
}
