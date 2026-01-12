package foundation

import (
	"encoding/json"
	"fmt"
	"strings"

	storymodel "z-novel-ai-api/internal/application/story/model"
	"z-novel-ai-api/internal/application/story/storyutil"
)

// ParseFoundationPlan 从模型输出中解析 FoundationPlan，并返回“截取后的 JSON 文本”。
func ParseFoundationPlan(rawText string) (*storymodel.FoundationPlan, string, error) {
	jsonText := storyutil.ExtractJSONObject(rawText)
	if strings.TrimSpace(jsonText) == "" {
		return nil, jsonText, fmt.Errorf("empty foundation plan output")
	}

	var plan storymodel.FoundationPlan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		return nil, jsonText, fmt.Errorf("failed to parse foundation plan json: %w", err)
	}
	return &plan, jsonText, nil
}
