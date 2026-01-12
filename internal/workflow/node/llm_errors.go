package node

import "strings"

func IsResponseFormatUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "response_format"):
		return true
	case strings.Contains(msg, "json_schema"):
		return true
	case strings.Contains(msg, "unknown parameter") && strings.Contains(msg, "response"):
		return true
	case strings.Contains(msg, "invalid") && strings.Contains(msg, "response"):
		return true
	case strings.Contains(msg, "response_schema"):
		return true
	case strings.Contains(msg, "failed to parse"):
		return true
	default:
		return false
	}
}
