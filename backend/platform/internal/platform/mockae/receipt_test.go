package mockae

import (
	"testing"
	"time"
)

func TestExecutionContextReceiptScopeAndExpiry(t *testing.T) {
	now := time.Now()
	p := map[string]any{"tenant_id": "42", "actor_user_id": "user", "application_id": "app", "application_release_id": "release", "turn_id": "turn"}
	r := contextReceipt{Binding: contextBinding(p), Versions: map[string]string{"metric": "v1"}, Expires: now.Add(time.Minute), Level: "EXECUTION_CONTEXT"}
	if !r.valid(p, "metric", now) {
		t.Fatal("valid receipt rejected")
	}
	for _, key := range []string{"tenant_id", "actor_user_id", "application_id", "application_release_id", "turn_id"} {
		v := p[key]
		p[key] = "other"
		if r.valid(p, "metric", now) {
			t.Fatalf("receipt accepted wrong %s", key)
		}
		p[key] = v
	}
	if r.valid(p, "metric", r.Expires) || r.valid(p, "metric", now.Add(time.Hour)) || r.valid(p, "other-metric", now) {
		t.Fatal("expired or wrong-asset receipt accepted")
	}
	r.Level = "STANDARD"
	if r.valid(p, "metric", now) {
		t.Fatal("read-only context accepted for execution")
	}
}
