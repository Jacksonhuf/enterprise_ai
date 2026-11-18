package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestR20Schema8AgentChatCanceledContextBeforePersist verifies that an already-canceled
// client context does not leave a half-written turn on schema8 (r.Context checked before persist).
func TestR20Schema8AgentChatCanceledContextBeforePersist(t *testing.T) {
	f, _, _ := schema8DualHTTPFixture(t)
	id := f.session()
	before := len(f.store.Read().Sessions[id].Messages)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	raw, _ := json.Marshal(map[string]string{"query": "SO1002需求增加到350"})
	req, err := http.NewRequestWithContext(ctx, "POST", f.host.URL+"/api/v1/agent-chat/"+id, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer demo-user")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	after := len(f.store.Read().Sessions[id].Messages)
	if after != before {
		t.Fatal("canceled chat persisted messages", before, after)
	}
}

// TestR20Schema8AgentChatExpiresDuringUpstreamCall uses a sub-second deadline so the
// cognitive/service leg may abort before persistence (client disconnect class).
func TestR20Schema8AgentChatExpiresDuringUpstreamCall(t *testing.T) {
	f, _, _ := schema8DualHTTPFixture(t)
	id := f.session()
	before := len(f.store.Read().Sessions[id].Messages)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	raw, _ := json.Marshal(map[string]string{"query": "SO1002需求增加到350"})
	req, err := http.NewRequestWithContext(ctx, "POST", f.host.URL+"/api/v1/agent-chat/"+id, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer demo-user")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			t.Fatal("expected non-success on expired context", resp.StatusCode)
		}
	}
	time.Sleep(50 * time.Millisecond)
	after := len(f.store.Read().Sessions[id].Messages)
	if after != before {
		t.Fatal("expired chat persisted messages", before, after)
	}
}