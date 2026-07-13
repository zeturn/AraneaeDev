package control

import (
	"encoding/json"
	"errors"
	"strings"
)

func normalizeMetadataJSON(raw any) (string, error) {
	if raw == nil {
		return "", nil
	}
	switch typed := raw.(type) {
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return "", nil
		}
		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return "", errors.New("metadata string must be valid JSON")
		}
		return value, nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", errors.New("metadata must be JSON serializable")
		}
		if string(data) == "null" {
			return "", nil
		}
		return string(data), nil
	}
}

func parseMetadataJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
