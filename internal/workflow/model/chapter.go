package model

type ChapterGenerateInput struct {
	ProjectTitle       string
	ProjectDescription string

	ChapterTitle   string
	ChapterOutline string

	RetrievedContext string

	TargetWordCount int
	WritingStyle    string
	POV             string

	Provider string
	Model    string

	Temperature *float32
	MaxTokens   *int
}

type ChapterGenerateOutput struct {
	Content string
	Meta    LLMUsageMeta
}
