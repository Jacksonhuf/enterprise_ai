package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"github.com/Tencent/WeKnora/internal/platform/mockae"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fixture struct {
	t      *testing.T
	store  *app.Store
	host   *httptest.Server
	client *cognitive.Client
}

func setup(t *testing.T) *fixture {
	t.Helper()
	st, e := app.Open(filepath.Join(t.TempDir(), "state.json"))
	if e != nil {
		t.Fatal(e)
	}
	key := []byte("test-key-not-production")
	c := &cognitive.Client{ServiceToken: "fixture-service", SigningKey: key}
	s := &api.Server{Store: st, Client: c, ServiceToken: "fixture-service", SigningKey: key}
	h := httptest.NewServer(capture(s.Handler()))
	m := httptest.NewServer(capture(mockae.New(h.URL, "fixture-service").Handler()))
	c.BaseURL = m.URL
	t.Cleanup(h.Close)
	t.Cleanup(m.Close)
	return &fixture{t, st, h, c}
}
func (f *fixture) request(identity, method, path string, body any, want int) []byte {
	f.t.Helper()
	var b io.Reader
	if body != nil {
		x, _ := json.Marshal(body)
		b = bytes.NewReader(x)
	}
	r, _ := http.NewRequest(method, f.host.URL+path, b)
	r.Header.Set("Authorization", "Bearer "+identity)
	r.Header.Set("Content-Type", "application/json")
	resp, e := http.DefaultClient.Do(r)
	if e != nil {
		f.t.Fatal(e)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		f.t.Fatalf("%s %s: status%d want%d: %s", method, path, resp.StatusCode, want, out)
	}
	return out
}
func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if e := json.Unmarshal(b, &out); e != nil {
		t.Fatalf("json: %s %v", b, e)
	}
	return out
}
func (f *fixture) session() string {
	return decode(f.t, f.request("demo-user", "POST", "/api/v1/cognitive/apps/channel-replenishment/sessions", map[string]any{}, 200))["session_id"].(string)
}
func TestSupplyChainRoundTripIsolationAndRevocation(t *testing.T) {
	f := setup(t)
	id := f.session()
	path := "/api/v1/agent-chat/" + id
	out := f.request("demo-user", "POST", path, map[string]string{"query": "SO1002需求增加到350后，有多少物料缺口？"}, 200)
	if !bytes.Contains(out, []byte("276.6")) || !bytes.Contains(out, []byte("336.6")) {
		t.Fatalf("wrong business result: %s", out)
	}
	msgs := decode(t, f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load?limit=200", nil, 200))["data"].([]any)
	if len(msgs) != 2 {
		t.Fatal(msgs)
	}
	answer := msgs[1].(map[string]any)
	eid := answer["execution_id"].(string)
	mid := answer["id"].(string)
	evidence := "/api/v1/cognitive/evidence/" + eid + "?session_id=" + id + "&message_id=" + mid
	f.request("demo-user", "GET", evidence, nil, 200)
	f.request("demo-outsider", "GET", evidence, nil, 404)
	f.request("demo-engineer", "GET", "/api/v1/sessions/"+id, nil, 404)
	f.request("demo-user", "GET", "/api/v1/sessions/continue-stream/"+id+"?message_id="+mid, nil, 200)
	if len(f.store.Read().Sessions[id].Messages) != 2 {
		t.Fatal("resume reran turn")
	}
	f.request("demo-user", "POST", path, map[string]string{"query": "SO1002需求增加到350", "actor_user_id": "demo-admin"}, 400)
	var gid string
	for _, g := range f.store.Read().Grants {
		if g.Action == "service.invoke" {
			gid = g.ID
		}
	}
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/"+gid, nil, 200)
	f.request("demo-user", "POST", path, map[string]string{"query": "SO1002需求增加到350"}, 403)
	f.request("demo-user", "GET", evidence, nil, 403)
	if len(f.store.Read().Sessions[id].Messages) != 2 {
		t.Fatal("denied turn persisted output")
	}
	f.request("demo-user", "GET", "/api/v1/cognitive/governance/grants", nil, 403)
	f.request("demo-outsider", "GET", "/api/v1/cognitive/apps", nil, 200)
}
func TestPublishIsImmutableAndNextTurnUsesCurrentRelease(t *testing.T) {
	f := setup(t)
	id := f.session()
	a := f.store.Read().Apps["channel-replenishment"]
	old := *a.CurrentReleaseID
	config := a.Draft
	config.Basic.Name = "新发布名称"
	payloadBytes, _ := json.Marshal(config)
	var payload map[string]any
	json.Unmarshal(payloadBytes, &payload)
	payload["expected_draft_revision"] = a.Revision
	saved := decode(t, f.request("demo-engineer", "PUT", "/api/v1/cognitive/apps/"+a.ID, payload, 200))
	rev := saved["draft_revision"]
	check := decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+a.ID+"/release-checks", map[string]any{"expected_draft_revision": rev}, 200))
	if check["status"] != "PASS" {
		t.Fatal(check)
	}
	req := map[string]any{"check_id": check["check_id"], "expected_draft_revision": rev, "expected_current_release_id": old, "release_note": "版本绑定测试"}
	published := decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+a.ID+"/releases", req, 200))
	newID := published["release_id"].(string)
	if newID == old {
		t.Fatal("release reused")
	}
	f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+a.ID+"/releases", req, 409)
	st := f.store.Read()
	for _, rel := range st.Apps[a.ID].Releases {
		if rel.ID == old && rel.Config.Basic.Name == config.Basic.Name {
			t.Fatal("old release mutated")
		}
	}
	out := f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	if !strings.Contains(string(out), newID) {
		t.Fatal("next turn did not bind current release")
	}
}
func TestInvalidQuantityNeverCalculates(t *testing.T) {
	f := setup(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	out := f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "需求改为0"}, 200)
	if bytes.Contains(out, []byte("276.6")) || !bytes.Contains(out, []byte("未执行计算")) {
		t.Fatal(string(out))
	}
}

var captureMu sync.Mutex

func capture(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := os.Getenv("COGNARA_CONTRACT_CAPTURE")
		if path == "" || !strings.HasPrefix(r.URL.Path, "/internal/") {
			h.ServeHTTP(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		var request, response any
		json.Unmarshal(body, &request)
		json.Unmarshal(rec.Body.Bytes(), &response)
		record := map[string]any{"method": r.Method, "path": r.URL.Path, "request": request, "status": rec.Code, "response": response}
		b, _ := json.Marshal(record)
		captureMu.Lock()
		f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if e == nil {
			f.Write(append(b, '\n'))
			f.Close()
		}
		captureMu.Unlock()
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	})
}

func TestServiceIdempotencyAndUsageCallbacks(t *testing.T) {
	f := setup(t)
	p, _ := governance.DemoIdentity("demo-user")
	p.ApplicationID = "channel-replenishment"
	p.ReleaseID = "release-seed-1"
	p.TurnID = "turn-dedup"
	p.Purpose = "application_turn"
	body := map[string]any{"service_ref": map[string]string{"service_id": "replenishment-gap", "version": "v1.1"}, "channel": "INTERNAL_APPLICATION", "arguments": map[string]any{"order_id": "SO1002", "target_quantity": 350}, "correlation_context": map[string]string{"turn_id": p.TurnID, "application_release_id": p.ReleaseID, "assistant_message_id": "message-dedup", "tool_call_id": "tool-dedup"}}
	first, e := f.client.Do(context.Background(), p, "POST", "/internal/v1/service-executions", body, "same-call-key")
	if e != nil {
		t.Fatal(e)
	}
	again, e := f.client.Do(context.Background(), p, "POST", "/internal/v1/service-executions", body, "same-call-key")
	if e != nil || !bytes.Equal(first, again) {
		t.Fatalf("replay changed result: %v", e)
	}
	if len(f.store.Read().Usage) != 1 {
		t.Fatal("duplicate usage or missing callback")
	}
	body["arguments"] = map[string]any{"order_id": "SO1002", "target_quantity": 351}
	_, e = f.client.Do(context.Background(), p, "POST", "/internal/v1/service-executions", body, "same-call-key")
	remote, ok := e.(*cognitive.RemoteError)
	if !ok || remote.Status != 409 {
		t.Fatalf("conflicting replay accepted %v", e)
	}
}
