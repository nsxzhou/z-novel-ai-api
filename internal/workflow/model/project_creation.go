package model

import "encoding/json"

// ProjectCreationGenerateInput 定义了项目创建生成器的输入参数
type ProjectCreationGenerateInput struct {
	Stage string
	Draft json.RawMessage

	Prompt      string
	Attachments []TextAttachment

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type ProjectCreationProjectDraft struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Genre       string `json:"genre,omitempty"`
}

type ProjectCreationGenerateOutput struct {
	AssistantMessage string
	NextStage        string
	Draft            json.RawMessage

	Action               string
	RequiresConfirmation bool
	ProposedProject      *ProjectCreationProjectDraft

	Meta LLMUsageMeta
}
