package story

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseFoundationPlan 从模型输出中解析 FoundationPlan，并返回“截取后的 JSON 文本”。
func ParseFoundationPlan(rawText string) (*FoundationPlan, string, error) {
	jsonText := extractJSONObject(rawText)
	if strings.TrimSpace(jsonText) == "" {
		return nil, jsonText, fmt.Errorf("empty foundation plan output")
	}

	var plan FoundationPlan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		return nil, jsonText, fmt.Errorf("failed to parse foundation plan json: %w", err)
	}
	return &plan, jsonText, nil
}
