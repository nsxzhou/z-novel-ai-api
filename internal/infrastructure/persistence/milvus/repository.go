// Package milvus 提供 Milvus 向量数据库访问层实现
package milvus

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Repository 向量检索仓储
type Repository struct {
	client *Client
}

// NewRepository 创建向量检索仓储
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// SearchParams 检索参数
type SearchParams struct {
	TenantID         string
	ProjectID        string
	QueryVector      []float32
	CurrentStoryTime int64
	TopK             int
	SegmentType      string
	SegmentTypes     []string
}

// SearchResult 检索结果
type SearchResult struct {
	ID          string
	Score       float32
	TextContent string
	ChapterID   string
	StoryTime   int64
}

// CreateCollection 创建集合
func (r *Repository) CreateCollection(ctx context.Context, schema *entity.Schema) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.CreateCollection",
		trace.WithAttributes(attribute.String("collection", schema.CollectionName)))
	defer span.End()

	collName := r.client.CollectionName(schema.CollectionName)
	schema.CollectionName = collName

	err := r.client.milvus.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// CreateIndex 创建 HNSW 索引
func (r *Repository) CreateIndex(ctx context.Context, collection string) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.CreateIndex",
		trace.WithAttributes(attribute.String("collection", collection)))
	defer span.End()

	collName := r.client.CollectionName(collection)

	idx, err := entity.NewIndexHNSW(
		entity.COSINE,
		r.client.config.HNSWM,
		r.client.config.HNSWEfConstruction,
	)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create index: %w", err)
	}

	err = r.client.milvus.CreateIndex(ctx, collName, "vector", idx, false)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// CreatePartition 创建分区
func (r *Repository) CreatePartition(ctx context.Context, collection, tenantID, projectID string) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.CreatePartition",
		trace.WithAttributes(
			attribute.String("collection", collection),
			attribute.String("partition", PartitionName(tenantID, projectID)),
		))
	defer span.End()

	collName := r.client.CollectionName(collection)
	partitionName := PartitionName(tenantID, projectID)

	return r.client.milvus.CreatePartition(ctx, collName, partitionName)
}

// SearchSegments 检索故事片段
func (r *Repository) SearchSegments(ctx context.Context, params *SearchParams) ([]*SearchResult, error) {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return nil, fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.SearchSegments",
		trace.WithAttributes(
			attribute.String("tenant_id", params.TenantID),
			attribute.String("project_id", params.ProjectID),
			attribute.Int("top_k", params.TopK),
		))
	defer span.End()

	collName := r.client.CollectionName(CollectionStorySegments)
	partitionName := PartitionName(params.TenantID, params.ProjectID)

	// 如果分区尚未创建（例如新项目），直接返回空结果，避免 Milvus 报 partition not found。
	if has, err := r.client.milvus.HasPartition(ctx, collName, partitionName); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check partition: %w", err)
	} else if !has {
		return []*SearchResult{}, nil
	}

	// 构建过滤表达式
	filter := fmt.Sprintf(
		`tenant_id == "%s" && project_id == "%s"`,
		params.TenantID, params.ProjectID,
	)

	// 时间过滤（排除未来事件）
	if params.CurrentStoryTime > 0 {
		filter += fmt.Sprintf(` && story_time <= %d`, params.CurrentStoryTime)
	}

	// 类型过滤
	if params.SegmentType != "" {
		filter += fmt.Sprintf(` && segment_type == "%s"`, params.SegmentType)
	} else if len(params.SegmentTypes) > 0 {
		// segment_type 只存在一个字段，使用 OR 条件构建过滤（避免依赖 IN 语法差异）。
		var parts []string
		for _, st := range params.SegmentTypes {
			st = strings.TrimSpace(st)
			if st == "" {
				continue
			}
			parts = append(parts, fmt.Sprintf(`segment_type == "%s"`, st))
		}
		if len(parts) > 0 {
			filter += " && (" + strings.Join(parts, " || ") + ")"
		}
	}

	// 搜索参数
	sp, err := entity.NewIndexHNSWSearchParam(128)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create search param: %w", err)
	}

	// 执行搜索
	results, err := r.client.milvus.Search(ctx,
		collName,
		[]string{partitionName},
		filter,
		[]string{"id", "text_content", "chapter_id", "story_time"},
		[]entity.Vector{entity.FloatVector(params.QueryVector)},
		"vector",
		entity.COSINE,
		params.TopK,
		sp,
	)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 解析结果
	var searchResults []*SearchResult
	for _, result := range results {
		for i := 0; i < result.ResultCount; i++ {
			sr := &SearchResult{
				Score: result.Scores[i],
			}

			// 提取字段值
			if idCol, ok := result.Fields.GetColumn("id").(*entity.ColumnVarChar); ok {
				sr.ID = idCol.Data()[i]
			}
			if textCol, ok := result.Fields.GetColumn("text_content").(*entity.ColumnVarChar); ok {
				sr.TextContent = textCol.Data()[i]
			}
			if chapterCol, ok := result.Fields.GetColumn("chapter_id").(*entity.ColumnVarChar); ok {
				sr.ChapterID = chapterCol.Data()[i]
			}
			if timeCol, ok := result.Fields.GetColumn("story_time").(*entity.ColumnInt64); ok {
				sr.StoryTime = timeCol.Data()[i]
			}

			searchResults = append(searchResults, sr)
		}
	}

	span.SetAttributes(attribute.Int("result_count", len(searchResults)))
	return searchResults, nil
}

// HybridSearchParams 混合检索参数
type HybridSearchParams struct {
	TenantID         string
	ProjectID        string
	QueryVector      []float32
	QueryText        string
	CurrentStoryTime int64
	TopK             int
	VectorWeight     float32
	KeywordWeight    float32
}

// HybridSearch 混合检索（语义 + 关键词）
func (r *Repository) HybridSearch(ctx context.Context, params *HybridSearchParams) ([]*SearchResult, error) {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return nil, fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.HybridSearch",
		trace.WithAttributes(
			attribute.String("tenant_id", params.TenantID),
			attribute.String("project_id", params.ProjectID),
			attribute.Int("top_k", params.TopK),
		))
	defer span.End()

	// 1. 向量检索
	vectorResults, err := r.SearchSegments(ctx, &SearchParams{
		TenantID:         params.TenantID,
		ProjectID:        params.ProjectID,
		QueryVector:      params.QueryVector,
		CurrentStoryTime: params.CurrentStoryTime,
		TopK:             params.TopK * 2, // 多召回用于重排
	})
	if err != nil {
		return nil, err
	}

	// 2. 如果没有关键词权重，直接返回向量结果
	if params.KeywordWeight <= 0 || params.QueryText == "" {
		if len(vectorResults) > params.TopK {
			vectorResults = vectorResults[:params.TopK]
		}
		return vectorResults, nil
	}

	// 3. 融合重排（RRF - Reciprocal Rank Fusion）
	merged := r.fusionRank(vectorResults, nil, params.VectorWeight, params.KeywordWeight)

	// 4. 返回 TopK
	if len(merged) > params.TopK {
		merged = merged[:params.TopK]
	}

	span.SetAttributes(attribute.Int("result_count", len(merged)))
	return merged, nil
}

// fusionRank RRF 融合重排
func (r *Repository) fusionRank(vecResults, kwResults []*SearchResult, vecWeight, kwWeight float32) []*SearchResult {
	scores := make(map[string]float32)
	results := make(map[string]*SearchResult)

	k := float32(60) // RRF 常数

	// 向量结果评分
	for i, res := range vecResults {
		score := vecWeight / (k + float32(i+1))
		scores[res.ID] += score
		results[res.ID] = res
	}

	// 关键词结果评分
	for i, res := range kwResults {
		score := kwWeight / (k + float32(i+1))
		scores[res.ID] += score
		if _, ok := results[res.ID]; !ok {
			results[res.ID] = res
		}
	}

	// 排序
	var merged []*SearchResult
	for id, res := range results {
		res.Score = scores[id]
		merged = append(merged, res)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	return merged
}

// InsertSegments 插入故事片段
func (r *Repository) InsertSegments(ctx context.Context, tenantID, projectID string, segments []*StorySegment) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.InsertSegments",
		trace.WithAttributes(
			attribute.String("tenant_id", tenantID),
			attribute.String("project_id", projectID),
			attribute.Int("count", len(segments)),
		))
	defer span.End()

	if len(segments) == 0 {
		return nil
	}

	collName := r.client.CollectionName(CollectionStorySegments)
	partitionName := PartitionName(tenantID, projectID)

	// 确保分区存在
	has, _ := r.client.milvus.HasPartition(ctx, collName, partitionName)
	if !has {
		if err := r.CreatePartition(ctx, CollectionStorySegments, tenantID, projectID); err != nil {
			return err
		}
	}

	// 准备数据
	ids := make([]string, len(segments))
	vectors := make([][]float32, len(segments))
	tenantIDs := make([]string, len(segments))
	projectIDs := make([]string, len(segments))
	chapterIDs := make([]string, len(segments))
	storyTimes := make([]int64, len(segments))
	segmentTypes := make([]string, len(segments))
	textContents := make([]string, len(segments))

	for i, seg := range segments {
		ids[i] = seg.ID
		vectors[i] = seg.Vector
		tenantIDs[i] = seg.TenantID
		projectIDs[i] = seg.ProjectID
		chapterIDs[i] = seg.ChapterID
		storyTimes[i] = seg.StoryTime
		segmentTypes[i] = seg.SegmentType
		textContents[i] = seg.TextContent
	}

	// 构建列
	idCol := entity.NewColumnVarChar("id", ids)
	vectorCol := entity.NewColumnFloatVector("vector", VectorDimension, vectors)
	tenantCol := entity.NewColumnVarChar("tenant_id", tenantIDs)
	projectCol := entity.NewColumnVarChar("project_id", projectIDs)
	chapterCol := entity.NewColumnVarChar("chapter_id", chapterIDs)
	timeCol := entity.NewColumnInt64("story_time", storyTimes)
	typeCol := entity.NewColumnVarChar("segment_type", segmentTypes)
	textCol := entity.NewColumnVarChar("text_content", textContents)

	// 插入
	_, err := r.client.milvus.Insert(ctx, collName, partitionName,
		idCol, vectorCol, tenantCol, projectCol, chapterCol, timeCol, typeCol, textCol)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to insert segments: %w", err)
	}

	return nil
}

// DeleteSegmentsByChapter 删除章节的所有片段
func (r *Repository) DeleteSegmentsByChapter(ctx context.Context, tenantID, projectID, chapterID string) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.DeleteSegmentsByChapter",
		trace.WithAttributes(
			attribute.String("chapter_id", chapterID),
		))
	defer span.End()

	collName := r.client.CollectionName(CollectionStorySegments)
	partitionName := PartitionName(tenantID, projectID)

	if has, err := r.client.milvus.HasPartition(ctx, collName, partitionName); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to check partition: %w", err)
	} else if !has {
		return nil
	}

	filter := fmt.Sprintf(`chapter_id == "%s"`, chapterID)

	err := r.client.milvus.Delete(ctx, collName, partitionName, filter)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete segments: %w", err)
	}

	return nil
}

// DeleteSegmentsByChapterAndType 删除指定 chapter_id + segment_type 的片段（同一 project 分区内）。
func (r *Repository) DeleteSegmentsByChapterAndType(ctx context.Context, tenantID, projectID, chapterID, segmentType string) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	chapterID = strings.TrimSpace(chapterID)
	segmentType = strings.TrimSpace(segmentType)
	if chapterID == "" || segmentType == "" {
		return nil
	}

	ctx, span := tracer.Start(ctx, "milvus.DeleteSegmentsByChapterAndType",
		trace.WithAttributes(
			attribute.String("chapter_id", chapterID),
			attribute.String("segment_type", segmentType),
		))
	defer span.End()

	collName := r.client.CollectionName(CollectionStorySegments)
	partitionName := PartitionName(tenantID, projectID)

	if has, err := r.client.milvus.HasPartition(ctx, collName, partitionName); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to check partition: %w", err)
	} else if !has {
		return nil
	}

	filter := fmt.Sprintf(`chapter_id == "%s" && segment_type == "%s"`, chapterID, segmentType)
	if err := r.client.milvus.Delete(ctx, collName, partitionName, filter); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete segments: %w", err)
	}
	return nil
}

// RebuildIndex 重建索引
func (r *Repository) RebuildIndex(ctx context.Context, collection string) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}
	ctx, span := tracer.Start(ctx, "milvus.RebuildIndex",
		trace.WithAttributes(attribute.String("collection", collection)))
	defer span.End()

	collName := r.client.CollectionName(collection)

	// 1. 释放集合
	if err := r.client.milvus.ReleaseCollection(ctx, collName); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to release collection: %w", err)
	}

	// 2. 删除旧索引
	if err := r.client.milvus.DropIndex(ctx, collName, "vector"); err != nil {
		// 忽略索引不存在的错误
	}

	// 3. 创建新索引
	if err := r.CreateIndex(ctx, collection); err != nil {
		return err
	}

	// 4. 重新加载集合
	return r.client.milvus.LoadCollection(ctx, collName, false)
}

// EnsureStorySegmentsCollection 确保 story_segments 集合与索引可用（不存在则创建）。
// 约束：不会做 drop/rebuild 等破坏性操作。
func (r *Repository) EnsureStorySegmentsCollection(ctx context.Context) error {
	if r == nil || r.client == nil || r.client.milvus == nil {
		return fmt.Errorf("milvus client not configured")
	}

	exists, err := r.client.HasCollection(ctx, CollectionStorySegments)
	if err != nil {
		return err
	}
	if !exists {
		if err := r.CreateCollection(ctx, StorySegmentsSchema()); err != nil {
			return err
		}
		// 新建集合时创建索引；若失败，允许后续由运维介入。
		_ = r.CreateIndex(ctx, CollectionStorySegments)
	}

	// 尝试确保集合已加载（若已加载，Milvus 会返回成功）
	return r.client.LoadCollection(ctx, CollectionStorySegments)
}
