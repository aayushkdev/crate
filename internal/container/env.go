package container

import (
	"fmt"
	"strings"
)

func mergeEnv(base []string, overrides []string) ([]string, error) {
	merged := append([]string(nil), base...)
	index := make(map[string]int, len(merged)+len(overrides))

	for i, entry := range merged {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}
		index[key] = i
	}

	for _, entry := range overrides {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid env %q: expected KEY=value", entry)
		}
		if strings.Contains(key, "\x00") || strings.Contains(entry, "\x00") {
			return nil, fmt.Errorf("invalid env %q: contains NUL byte", entry)
		}

		if i, ok := index[key]; ok {
			merged[i] = entry
			continue
		}
		index[key] = len(merged)
		merged = append(merged, entry)
	}

	return merged, nil
}
