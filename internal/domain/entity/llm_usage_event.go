// Package entity 定义领域实体
package entity

import "time"

type LLMUsageEvent struct {
	ID               string    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID         string    `json:"tenant_id" gorm:"type:uuid;index;not null"`
	Provider         string    `json:"provider" gorm:"type:varchar(32);not null"`
	Model            string    `json:"model" gorm:"type:varchar(64);not null"`
	TokensPrompt     int       `json:"tokens_prompt" gorm:"not null;default:0"`
	TokensCompletion int       `json:"tokens_completion" gorm:"not null;default:0"`
	DurationMs       int       `json:"duration_ms" gorm:"not null;default:0"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (LLMUsageEvent) TableName() string {
	return "llm_usage_events"
}

