package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	agentruntime "github.com/Tencent/WeKnora/internal/platform/adapters/runtime"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
)

func TestRestartReplayRedactionAndModelHistoryIsolation(t *testing.T) {
	f := setup(t)
	path := filepath.Join(t.TempDir(), "restart.db")
	db, e := app.OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	f.store = db
	f.server.Store = db
	session := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+session, map[string]string{"query": "SO1002需求增加到350"}, 200)
	answer := db.Read().Sessions[session].Messages[1]
	old := f.server
	f.host.Close()
	db.Close()
	restored, e := app.OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { restored.Close() })
	next := &api.Server{Store: restored, Client: f.client, ServiceToken: old.ServiceToken, SigningKey: old.SigningKey}
	host := httptest.NewServer(next.Handler())
	t.Cleanup(host.Close)
	f.host = host
	f.server = next
	f.store = restored
	replay := "/api/v1/sessions/continue-stream/" + session + "?message_id=" + answer.ID
	out := f.request("demo-user", "GET", replay, nil, 200)
	if !bytes.Contains(out, []byte("276.6")) || !bytes.Contains(out, []byte("event: done")) {
		t.Fatal("durable replay failed")
	}
	if len(restored.Read().Usage) != 1 || len(restored.Read().Sessions[session].Messages) != 2 {
		t.Fatal("replay reran the business call")
	}
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-service", nil, 200)
	messages := f.request("demo-user", "GET", "/api/v1/messages/"+session+"/load", nil, 200)
	if bytes.Contains(messages, []byte("276.6")) || !bytes.Contains(messages, []byte("content_redacted")) {
		t.Fatal("revoked historical result leaked")
	}
	metadata := f.request("demo-user", "GET", "/api/v1/sessions/"+session, nil, 200)
	if bytes.Contains(metadata, []byte("276.6")) {
		t.Fatal("session detail bypassed redaction")
	}
	f.request("demo-user", "GET", replay, nil, 403)
	t.Setenv("SSRF_WHITELIST", "127.0.0.1")
	t.Setenv("COGNARA_HISTORY_MODEL_KEY", "local-history-fixture")
	var observed []byte
	var observedMu sync.Mutex
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		observedMu.Lock()
		observed = append(observed, raw...)
		observedMu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"当前消息已处理。\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	models, e := agentruntime.New([]agentruntime.ModelConfig{{ID: "history-model", Name: "历史隔离协议夹具", Model: "fixture", BaseURL: provider.URL + "/v1", APIKeyEnv: "COGNARA_HISTORY_MODEL_KEY", HTTP: provider.Client()}})
	if e != nil {
		t.Fatal(e)
	}
	next.Models = models
	restored.Update(func(st *app.State) error {
		a := st.Apps["channel-replenishment"]
		r, _ := app.Current(a)
		r.ID = "history-release"
		r.Config.ModelPrompt.ModelID = "history-model"
		a.Releases = append(a.Releases, r)
		a.CurrentReleaseID = &r.ID
		st.Apps[a.ID] = a
		return nil
	})
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+session, map[string]string{"query": "你好"}, 200)
	observedMu.Lock()
	defer observedMu.Unlock()
	if len(observed) == 0 {
		t.Fatal("model provider not called")
	}
	if bytes.Contains(observed, []byte("276.6")) || !bytes.Contains(observed, []byte("当前无权访问")) {
		t.Fatal("revoked historical facts were sent to the provider")
	}
}
func TestExceptionTraceRevisionAndTenantIsolation(t *testing.T) {
	f := setup(t)
	req, _ := http.NewRequest("POST", f.host.URL+"/api/v1/cognitive/models/missing/test", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer demo-admin")
	req.Header.Set("X-Request-ID", "forged-browser-id")
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	body := decode(t, raw)
	id := body["request_id"].(string)
	if resp.StatusCode != 503 || id == "forged-browser-id" || id != resp.Header.Get("X-Request-ID") {
		t.Fatal("trace ID not bound to response")
	}
	items := decode(t, f.request("demo-admin", "GET", "/api/v1/cognitive/governance/exceptions", nil, 200))["items"].([]any)
	if len(items) != 1 {
		t.Fatal(items)
	}
	item := items[0].(map[string]any)
	if item["request_id"] != id || item["status"] != "OPEN" {
		t.Fatal(item)
	}
	route := item["route"].(string)
	if bytes.Contains([]byte(route), []byte("missing")) {
		t.Fatal("exception stored arbitrary URL parameters")
	}
	endpoint := "/api/v1/cognitive/governance/exceptions/" + item["id"].(string) + "/status"
	revision := item["revision"]
	f.request("demo-user", "POST", endpoint, map[string]any{"expected_revision": revision, "status": "RESOLVED"}, 403)
	updated := decode(t, f.request("demo-admin", "POST", endpoint, map[string]any{"expected_revision": revision, "status": "ACKNOWLEDGED"}, 200))
	f.request("demo-admin", "POST", endpoint, map[string]any{"expected_revision": revision, "status": "RESOLVED"}, 409)
	updated = decode(t, f.request("demo-admin", "POST", endpoint, map[string]any{"expected_revision": updated["revision"], "status": "RESOLVED"}, 200))
	f.request("demo-admin", "POST", endpoint, map[string]any{"expected_revision": updated["revision"], "status": "OPEN"}, 200)
	f.store.Update(func(st *app.State) error {
		st.Exceptions = append(st.Exceptions, map[string]any{"id": "other-tenant-exception", "tenant_id": "43", "status": "OPEN", "revision": "v1"})
		return nil
	})
	f.request("demo-admin", "POST", "/api/v1/cognitive/governance/exceptions/other-tenant-exception/status", map[string]any{"expected_revision": "v1", "status": "RESOLVED"}, 404)
	list := f.request("demo-admin", "GET", "/api/v1/cognitive/governance/exceptions", nil, 200)
	if bytes.Contains(list, []byte("other-tenant-exception")) {
		t.Fatal("cross-tenant exception exposed")
	}
}
func TestProviderErrorResponseDoesNotExposeRawBody(t *testing.T) {
	w := httptest.NewRecorder()
	api.ErrorResponse(w, &cognitive.RemoteError{Status: 503, Body: `{"error":{"code":"CAPABILITY_UNAVAILABLE","message":"secret-fixture-key SQL private-host"}}`})
	out := w.Body.Bytes()
	if bytes.Contains(out, []byte("secret-fixture-key")) || bytes.Contains(out, []byte("private-host")) {
		t.Fatal("upstream diagnostics leaked")
	}
	decoded := decode(t, out)
	if decoded["request_id"] != w.Header().Get("X-Request-ID") {
		t.Fatal("error trace mismatch")
	}
}
func TestAuthorizationMutationAndAuditCommitTogether(t *testing.T) {
	f := setup(t)
	before := len(f.store.Read().Audit)
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-service", nil, 200)
	st := f.store.Read()
	if len(st.Audit) != before+1 {
		t.Fatal("authorization audit duplicated or missing")
	}
	event := st.Audit[len(st.Audit)-1]
	if event.Action != "authorization.revoke" || event.RequestID == "" {
		t.Fatal(event)
	}
	for _, g := range st.Grants {
		if g.ID == "grant-service" && g.Active {
			t.Fatal("audit committed without revocation")
		}
	}
}
func TestPendingMessagesCannotReplayAsCompleted(t *testing.T) {
	f := setup(t)
	id := f.session()
	f.store.Update(func(st *app.State) error {
		s := st.Sessions[id]
		s.Messages = append(s.Messages, app.Message{ID: "pending-message", SessionID: id, Role: "assistant", Content: "private", Completed: false})
		st.Sessions[id] = s
		return nil
	})
	f.request("demo-user", "GET", "/api/v1/sessions/continue-stream/"+id+"?message_id=pending-message", nil, 409)
}

func TestEventRedeliveryAfterRestartIsDurablyDeduplicated(t *testing.T) {
	f := setup(t)
	path := filepath.Join(t.TempDir(), "events.db")
	db, e := app.OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	f.server.Store = db
	f.store = db
	event := map[string]any{"event_id": "retry-audit-event", "event_type": "service.execution.completed", "occurred_at": "2026-11-15T10:00:00Z", "producer": "E", "subject": map[string]any{"tenant_id": "42", "actor_user_id": "demo-user", "client_id": "channel-replenishment"}, "resource": map[string]string{"type": "SERVICE", "id": "replenishment-gap"}, "action": "service.invoke", "outcome": "success", "correlation": map[string]any{}}
	send := func(body map[string]any, want int) map[string]any {
		t.Helper()
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", f.host.URL+"/internal/v1/audit-events", bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+f.server.ServiceToken)
		req.Header.Set("Idempotency-Key", "retry-audit-event")
		req.Header.Set("X-Request-ID", "trusted-redelivery-request")
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%d %s", resp.StatusCode, raw)
		}
		if resp.Header.Get("X-Request-ID") != "trusted-redelivery-request" {
			t.Fatal("trusted service trace not preserved")
		}
		return decode(t, raw)
	}
	if send(event, 200)["receipt_status"] != "ACCEPTED" {
		t.Fatal("event not accepted")
	}
	db.Close()
	restored, e := app.OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	f.server.Store = restored
	f.store = restored
	if send(event, 200)["receipt_status"] != "DUPLICATE" || len(restored.Read().Audit) != 1 {
		t.Fatal("restart redelivery duplicated audit")
	}
	event["action"] = "different-action"
	send(event, 409)
	if len(restored.Read().Audit) != 1 {
		t.Fatal("conflicting event mutated audit")
	}
}
func TestGovernanceAuditAndUsageDoNotCrossTenants(t *testing.T) {
	f := setup(t)
	f.store.Update(func(st *app.State) error {
		st.Audit = append(st.Audit, app.Event{ID: "tenant43-audit", TenantID: "43", ActorID: "demo-outsider", Action: "private-other-tenant"}, app.Event{ID: "legacy-outside", ActorID: "demo-outsider", Action: "legacy-private"}, app.Event{ID: "our-audit", TenantID: "42", ActorID: "demo-user", Action: "our-action"})
		st.Usage = append(st.Usage, map[string]any{"tenant_id": "43", "capability": "private-other-capability"}, map[string]any{"tenant_id": "42", "capability": "our-capability"})
		return nil
	})
	audit := f.request("demo-admin", "GET", "/api/v1/cognitive/governance/audit", nil, 200)
	usage := f.request("demo-admin", "GET", "/api/v1/cognitive/governance/usage", nil, 200)
	if bytes.Contains(audit, []byte("private")) || !bytes.Contains(audit, []byte("our-action")) || bytes.Contains(usage, []byte("private")) || !bytes.Contains(usage, []byte("our-capability")) {
		t.Fatal("governance records crossed tenant boundary")
	}
}

func TestMalformedEventNeverCreatesReceiptOrAudit(t *testing.T) {
	f := setup(t)
	event := map[string]any{"event_id": "malformed-event", "producer": "E", "subject": map[string]any{"tenant_id": "42", "actor_user_id": "demo-user", "client_id": nil}, "private_unknown_field": "not-to-be-recorded"}
	raw, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", f.host.URL+"/internal/v1/audit-events", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+f.server.ServiceToken)
	req.Header.Set("Idempotency-Key", "malformed-event")
	req.Header.Set("X-Request-ID", "event-validation-request")
	response, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	if response.StatusCode != 400 || len(f.store.Read().Audit) != 0 || len(f.store.Read().Receipts) != 0 {
		t.Fatal("malformed event accepted or partially persisted")
	}
}

func TestUsageRedeliveryAfterRestartDoesNotDoubleCount(t *testing.T) {
	f := setup(t)
	path := filepath.Join(t.TempDir(), "usage.db")
	db, e := app.OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	f.store = db
	f.server.Store = db
	event := map[string]any{"event_id": "usage-retry", "occurred_at": "2026-11-15T10:00:00Z", "producer": "E", "tenant_id": "42", "application_id": "channel-replenishment", "capability": "metric", "count": 1, "unit": "CALL"}
	send := func() string {
		t.Helper()
		body, _ := json.Marshal(event)
		req, _ := http.NewRequest("POST", f.host.URL+"/internal/v1/usage-events", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+f.server.ServiceToken)
		req.Header.Set("Idempotency-Key", "usage-retry")
		req.Header.Set("X-Request-ID", "usage-redelivery-request")
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Fatalf("usage %d %s", resp.StatusCode, raw)
		}
		return decode(t, raw)["receipt_status"].(string)
	}
	if send() != "ACCEPTED" {
		t.Fatal("usage not accepted")
	}
	db.Close()
	restored, e := app.OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	f.store = restored
	f.server.Store = restored
	if send() != "DUPLICATE" || len(restored.Read().Usage) != 1 {
		t.Fatal("usage counted again after restart")
	}
}

func TestUnknownHistoryAuthorizationProfileIsHidden(t *testing.T) {
	f := setup(t)
	id := f.session()
	f.store.Update(func(st *app.State) error {
		s := st.Sessions[id]
		s.Messages = append(s.Messages, app.Message{ID: "future-message", SessionID: id, Role: "assistant", Content: "future private result", Completed: true, ReleaseID: "release-seed-1", AuthorizationProfile: 2})
		st.Sessions[id] = s
		return nil
	})
	raw := f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load", nil, 200)
	if bytes.Contains(raw, []byte("future private result")) || !bytes.Contains(raw, []byte("content_redacted")) {
		t.Fatal("unknown authorization profile exposed content")
	}
	f.request("demo-user", "GET", "/api/v1/sessions/continue-stream/"+id+"?message_id=future-message", nil, 403)
}
