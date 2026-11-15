package api

import (
	"strings"
	"testing"
)

func TestEventValidationContractBoundaries(t *testing.T) {
	usage := map[string]any{"event_id": "usage", "occurred_at": "2026-11-15T10:00:00Z", "producer": "E", "tenant_id": "18446744073709551615", "capability": "metric", "count": float64(0), "unit": "CALL"}
	if !validEvent("usage-events", usage) {
		t.Fatal("valid uint64 maximum rejected")
	}
	for _, tenant := range []string{"18446744073709551616", "042", "0", "-1"} {
		usage["tenant_id"] = tenant
		if validEvent("usage-events", usage) {
			t.Fatalf("invalid canonical tenant accepted %s", tenant)
		}
	}
	usage["tenant_id"] = "42"
	usage["count"] = float64(-1)
	if validEvent("usage-events", usage) {
		t.Fatal("negative usage accepted")
	}
	audit := map[string]any{"event_id": "audit", "event_type": "completed", "occurred_at": "2026-11-15T10:00:00Z", "producer": "E", "subject": map[string]any{"tenant_id": "42", "actor_user_id": nil, "client_id": strings.Repeat("a", 128)}, "resource": map[string]any{"type": "ASSET", "id": strings.Repeat("界", 64)}, "action": "execution.invoke", "outcome": "success", "correlation": map[string]any{"turn_id": nil}}
	if !validEvent("audit-events", audit) {
		t.Fatal("valid nullable subject or Unicode limit rejected")
	}
	audit["resource"].(map[string]any)["id"] = strings.Repeat("界", 65)
	if validEvent("audit-events", audit) {
		t.Fatal("resource ID beyond limit accepted")
	}
	audit["resource"].(map[string]any)["id"] = "asset"
	audit["correlation"].(map[string]any)["unknown"] = true
	if validEvent("audit-events", audit) {
		t.Fatal("unknown correlation property accepted")
	}
}
