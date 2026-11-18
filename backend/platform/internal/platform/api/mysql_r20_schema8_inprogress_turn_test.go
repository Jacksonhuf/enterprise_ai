package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	_ "github.com/go-sql-driver/mysql"
)

// TestR20Schema8DualHostContinueStreamInProgressTurn ensures both API hosts reject
// continue-stream for assistant messages that are not marked completed (R14 probe).
func TestR20Schema8DualHostContinueStreamInProgressTurn(t *testing.T) {
	f, dsn, second := schema8DualHTTPFixture(t)
	id := f.session()
	const pendingID = "schema8-in-progress-assistant"
	snap := f.store.Read()
	sess, found := snap.Sessions[id]
	if !found {
		t.Fatal("session missing after create", id)
	}
	msg := app.Message{ID: pendingID, SessionID: id, Role: "assistant", Content: "partial", Completed: false, CreatedAt: app.Now()}
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := json.Marshal(map[string]any{"role": msg.Role, "created_at": msg.CreatedAt, "is_completed": false})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ordinal := len(sess.Messages) + 1
	if _, err = db.ExecContext(context.Background(),
		`INSERT INTO cognara_fg_session_messages(tenant_id,session_id,message_id,ordinal,metadata,raw_payload) VALUES(?,?,?,?,?,?)`,
		sess.TenantID, id, pendingID, ordinal, string(meta), payload); err != nil {
		t.Fatal("seed in-progress message", err)
	}
	path := "/api/v1/sessions/continue-stream/" + id + "?message_id=" + pendingID
	for _, base := range []string{f.host.URL, second.URL} {
		req, _ := http.NewRequest("GET", base+path, nil)
		req.Header.Set("Authorization", "Bearer demo-user")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 409 {
			t.Fatal("in-progress message must not replay", base, resp.StatusCode)
		}
	}
}

// TestR20Schema8InProgressTurnDisconnectThenCompletedReplay documents that client
// disconnect before persist leaves no replayable turn; a later completed chat can replay.
func TestR20Schema8InProgressTurnDisconnectThenCompletedReplay(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
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
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
	}
	if len(f.store.Read().Sessions[id].Messages) != before {
		t.Fatal("disconnect must not leave partial assistant row for replay")
	}
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	answer := f.store.Read().Sessions[id].Messages[1]
	if !answer.Completed {
		t.Fatal("expected completed assistant message")
	}
	path := "/api/v1/sessions/continue-stream/" + id + "?message_id=" + answer.ID
	for _, base := range []string{f.host.URL, second.URL} {
		out := getBytes(t, "demo-user", base+path)
		if len(out) == 0 {
			t.Fatal("completed replay failed after prior disconnect", base)
		}
	}
}