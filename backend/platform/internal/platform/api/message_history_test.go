package api_test

import (
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/url"
	"testing"
)

func seedHistory(t *testing.T, f *fixture, id string, count int) {
	t.Helper()
	if err := f.store.Update(func(st *app.State) error {
		session := st.Sessions[id]
		for i := 0; i < count; i++ {
			session.Messages = append(session.Messages, app.Message{ID: fmt.Sprintf("history-%03d", i), SessionID: id, Role: "user", Content: fmt.Sprint(i), CreatedAt: "2026-11-15T12:00:00Z", Completed: true})
		}
		st.Sessions[id] = session
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestMessageIDPaginationSameTimeAndConcurrentAppend(t *testing.T) {
	f := setup(t)
	id := f.session()
	seedHistory(t, f, id, 215)
	detail := decode(t, f.request("demo-user", "GET", "/api/v1/sessions/"+id+"?include_messages=false", nil, 200))
	if detail["session"].(map[string]any)["messages"] != nil {
		t.Fatal("metadata read loaded full history")
	}
	cursor := ""
	seen := map[string]bool{}
	batches := [][]string{}
	for {
		result := decode(t, f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load?limit=37&before_message_id="+cursor, nil, 200))
		batch := []string{}
		for _, row := range result["data"].([]any) {
			key := row.(map[string]any)["id"].(string)
			if seen[key] {
				t.Fatal("duplicate", key)
			}
			seen[key] = true
			batch = append(batch, key)
		}
		batches = append(batches, batch)
		if len(batches) == 1 {
			err := f.store.Update(func(st *app.State) error {
				v := st.Sessions[id]
				v.Messages = append(v.Messages, app.Message{ID: "new-after-first-page", SessionID: id, Role: "user", Content: "new", CreatedAt: app.Now(), Completed: true})
				st.Sessions[id] = v
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if !result["has_more"].(bool) {
			if result["next_before_message_id"] != nil {
				t.Fatal("terminal cursor")
			}
			break
		}
		next := result["next_before_message_id"].(string)
		if next == cursor {
			t.Fatal("stuck cursor")
		}
		cursor = next
	}
	if len(seen) != 215 || seen["new-after-first-page"] {
		t.Fatal("history missing or crossed append boundary", len(seen))
	}
	index := 0
	for i := len(batches) - 1; i >= 0; i-- {
		for _, key := range batches[i] {
			if key != fmt.Sprintf("history-%03d", index) {
				t.Fatal("order changed", key, index)
			}
			index++
		}
	}
}
func TestMessageHistoryValidatesCursorAndUsesTimeInstants(t *testing.T) {
	f := setup(t)
	id := f.session()
	seedHistory(t, f, id, 2)
	base := "/api/v1/messages/" + id + "/load?"
	for _, query := range []string{"limit=0", "limit=201", "limit=oops", "before_message_id=unknown", "before_time=bad", "before_time=2026-11-15T12:00:00Z&before_message_id=history-001"} {
		f.request("demo-user", "GET", base+query, nil, 400)
	}
	// This is 12:30 UTC. String ordering would incorrectly put +08 timestamps before UTC.
	result := decode(t, f.request("demo-user", "GET", base+"before_time="+url.QueryEscape("2026-11-15T20:30:00+08:00"), nil, 200))
	if len(result["data"].([]any)) != 2 {
		t.Fatal("timestamp offset not normalized")
	}
	result = decode(t, f.request("demo-user", "GET", base+"before_time="+url.QueryEscape("2026-11-15T12:30:00+08:00"), nil, 200))
	if len(result["data"].([]any)) != 0 {
		t.Fatal("lexical ordering leaked later messages")
	}
	result = decode(t, f.request("demo-user", "GET", base+"before_time=2026-11-15T12:00:00Z", nil, 200))
	if len(result["data"].([]any)) != 0 {
		t.Fatal("time boundary must be exclusive")
	}
	other := f.session()
	f.request("demo-user", "GET", "/api/v1/messages/"+other+"/load?before_message_id=history-001", nil, 400)
	f.request("demo-outsider", "GET", base, nil, 404)
	f.request("demo-engineer", "GET", base, nil, 404)
}
func TestSessionDirectoryAndPagedHistoryRecheckAuthorization(t *testing.T) {
	f := setup(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	f.request("demo-user", "GET", "/api/v1/sessions", nil, 400)
	f.request("demo-outsider", "GET", "/api/v1/sessions?app_id=channel-replenishment", nil, 404)
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-service", nil, 200)
	result := decode(t, f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load?limit=1", nil, 200))
	message := result["data"].([]any)[0].(map[string]any)
	if message["content_redacted"] != true || message["result"] != nil || message["execution_id"] != nil {
		t.Fatal("revoked result leaked", message)
	}
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-app", nil, 200)
	f.request("demo-user", "GET", "/api/v1/sessions?app_id=channel-replenishment", nil, 403)
	f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load?before_message_id=unknown", nil, 403)
}
