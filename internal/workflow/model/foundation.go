package model

type FoundationGenerateInput struct {
	ProjectTitle       string
	ProjectDescription string

	Prompt      string
	Attachments []TextAttachment

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}
