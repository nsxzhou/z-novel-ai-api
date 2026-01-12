package model

import (
	"strings"
	"z-novel-ai-api/internal/domain/entity"
)

func IsEmptyWorldSettings(ws entity.WorldSettings) bool {
	if strings.TrimSpace(ws.TimeSystem) != "" {
		return false
	}
	if strings.TrimSpace(ws.Calendar) != "" {
		return false
	}
	return len(ws.Locations) == 0
}
