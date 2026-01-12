package node

import (
	"strings"

	wfmodel "z-novel-ai-api/internal/workflow/model"
)

func BuildAttachmentsBlock(attachments []wfmodel.TextAttachment) string {
	if len(attachments) == 0 {
		return ""
	}
	lines := make([]string, 0, len(attachments)+1)
	lines = append(lines, "附加材料：")
	for _, a := range attachments {
		name := strings.TrimSpace(a.Name)
		content := strings.TrimSpace(a.Content)
		if content == "" {
			continue
		}
		if name == "" {
			name = "附件"
		}
		lines = append(lines, "- "+name+"\n"+content)
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n\n")
}
