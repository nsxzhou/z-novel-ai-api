// Package milvus 提供 Milvus 向量数据库访问层实现
package milvus

import (
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	// CollectionStorySegments 故事片段集合
	CollectionStorySegments = "story_segments"
	// CollectionEntityProfiles 实体档案集合
	CollectionEntityProfiles = "entity_profiles"

	// VectorDimension 向量维度
	VectorDimension = 1024
)

// StorySegmentsSchema 故事片段 Collection Schema
func StorySegmentsSchema() *entity.Schema {
	return &entity.Schema{
		CollectionName: CollectionStorySegments,
		Description:    "Story content segments for semantic search",
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				AutoID:     false,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": "1024",
				},
			},
			{
				Name:     "tenant_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "project_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "chapter_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "story_time",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "segment_type",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "32",
				},
			},
			{
				Name:     "text_content",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
		},
	}
}

// EntityProfilesSchema 实体档案 Collection Schema
func EntityProfilesSchema() *entity.Schema {
	return &entity.Schema{
		CollectionName: CollectionEntityProfiles,
		Description:    "Entity profiles for character/item/location search",
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				AutoID:     false,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": "1024",
				},
			},
			{
				Name:     "tenant_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "project_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "entity_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "entity_type",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "32",
				},
			},
			{
				Name:     "name",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "256",
				},
			},
			{
				Name:     "description",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
		},
	}
}

// StorySegment 故事片段数据结构
type StorySegment struct {
	ID          string    `json:"id"`
	Vector      []float32 `json:"vector"`
	TenantID    string    `json:"tenant_id"`
	ProjectID   string    `json:"project_id"`
	ChapterID   string    `json:"chapter_id"`
	StoryTime   int64     `json:"story_time"`
	SegmentType string    `json:"segment_type"`
	TextContent string    `json:"text_content"`
}

// EntityProfile 实体档案数据结构
type EntityProfile struct {
	ID          string    `json:"id"`
	Vector      []float32 `json:"vector"`
	TenantID    string    `json:"tenant_id"`
	ProjectID   string    `json:"project_id"`
	EntityID    string    `json:"entity_id"`
	EntityType  string    `json:"entity_type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// PartitionName 生成分区名称
func PartitionName(tenantID, projectID string) string {
	return "tenant_" + tenantID + "_proj_" + projectID
}
