package model

import "time"

type TextAttachment struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type LLMUsageMeta struct {
	Provider         string
	Model            string
	PromptTokens     int
	CompletionTokens int
	Temperature      float64
	GeneratedAt      time.Time
}
