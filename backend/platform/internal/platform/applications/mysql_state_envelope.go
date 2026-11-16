package applications

import (
	"encoding/json"
	"errors"
	"strings"
)

// Preserve opaque top-level extensions from the exact row locked by this update.
// Known fields are replaced as a whole, including deletions, and their old case
// aliases must not shadow the canonical fields on the next Go decode.
func preserveMySQLStateEnvelope(source, canonical []byte) ([]byte, error) {
	var original, next map[string]json.RawMessage
	if json.Unmarshal(source, &original) != nil || original == nil || json.Unmarshal(canonical, &next) != nil || next == nil {
		return nil, errors.New("invalid MySQL state envelope")
	}
	fields := make([]string, 0, len(next))
	for field := range next {
		fields = append(fields, field)
	}
	for key, value := range original {
		known := false
		for _, field := range fields {
			if strings.EqualFold(key, field) {
				known = true
				break
			}
		}
		if !known {
			next[key] = value
		}
	}
	return json.Marshal(next)
}
