package jsonx

import (
	"strings"
)

func ExtractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "```") {
		start := strings.Index(raw, "\n")
		if start >= 0 {
			raw = strings.TrimSpace(raw[start+1:])
		}
		if end := strings.LastIndex(raw, "```"); end >= 0 {
			raw = strings.TrimSpace(raw[:end])
		}
	}
	startObj := strings.Index(raw, "{")
	startArr := strings.Index(raw, "[")
	start := -1
	if startObj >= 0 && startArr >= 0 {
		if startObj < startArr {
			start = startObj
		} else {
			start = startArr
		}
	} else if startObj >= 0 {
		start = startObj
	} else {
		start = startArr
	}
	if start < 0 {
		return ""
	}
	endObj := strings.LastIndex(raw, "}")
	endArr := strings.LastIndex(raw, "]")
	end := endObj
	if endArr > end {
		end = endArr
	}
	if end < start {
		return ""
	}
	return strings.TrimSpace(raw[start : end+1])
}
