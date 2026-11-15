package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func fixtureModel(t *testing.T, toolArgs string) (*Registry, *atomic.Int32) {
	t.Helper()
	t.Setenv("COGNARA_TEST_MODEL_KEY", "local-protocol-fixture-key")
	t.Setenv("SSRF_WHITELIST", "127.0.0.1")
	calls := &atomic.Int32{}
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer local-protocol-fixture-key" {
			t.Errorf("bad protocol or auth")
		}
		var req struct {
			Stream   bool             `json:"stream"`
			Tools    []any            `json:"tools"`
			Messages []map[string]any `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		calls.Add(1)
		if !req.Stream {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		hasResult := false
		for _, m := range req.Messages {
			if m["role"] == "tool" {
				hasResult = true
			}
		}
		var chunk any
		if !hasResult {
			if len(req.Tools) != 1 {
				t.Errorf("expected one controlled tool, got %d", len(req.Tools))
			}
			chunk = map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "provider-call", "type": "function", "function": map[string]any{"name": "cognitive_call_service", "arguments": toolArgs}}}}, "finish_reason": "tool_calls"}}}
		} else {
			chunk = map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": "芯片原始缺口为276.6，需核对历史时点，不承诺交期。"}, "finish_reason": "stop"}}}
		}
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", b)
	}))
	t.Cleanup(host.Close)
	models, e := New([]ModelConfig{{ID: "fixture-model", Name: "协议验证模型", Model: "fixture", BaseURL: host.URL + "/v1", APIKeyEnv: "COGNARA_TEST_MODEL_KEY", HTTP: host.Client()}})
	if e != nil {
		t.Fatal(e)
	}
	return models, calls
}
func runtimeRequest(t *testing.T) Request {
	st, e := app.Open("")
	if e != nil {
		t.Fatal(e)
	}
	rel, _ := app.Current(st.Read().Apps["channel-replenishment"])
	rel.Config.ModelPrompt.ModelID = "fixture-model"
	p, _ := governance.DemoIdentity("demo-user")
	p.ApplicationID = rel.AppID
	p.ReleaseID = rel.ID
	p.TurnID = "turn-test"
	p.Purpose = "application_turn"
	return Request{Principal: p, Release: rel, Session: app.Session{ID: "session-test"}, MessageID: "message-test", Query: "SO1002需求增加到350"}
}
func TestReusedEngineCallsControlledService(t *testing.T) {
	models, calls := fixtureModel(t, `{"service_id":"replenishment-gap","arguments":{"order_id":"SO1002","target_quantity":350}}`)
	var serviceCalls atomic.Int32
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceCalls.Add(1)
		if r.URL.Path != "/internal/v1/service-executions" || r.Header.Get("Idempotency-Key") == "" || r.Header.Get("X-Delegation-Token") == "" {
			t.Error("missing contract binding")
		}
		var input map[string]any
		json.NewDecoder(r.Body).Decode(&input)
		ref := input["service_ref"].(map[string]any)
		if ref["version"] != "v1.1" {
			t.Error("version not server-bound")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"SUCCESS","result":{"summary":"原始缺口276.6"},"step_executions":[{"execution_id":"execution-test"}]}`)
	}))
	defer host.Close()
	engine := &Engine{Models: models, Client: &cognitive.Client{BaseURL: host.URL, ServiceToken: "fixture", SigningKey: []byte("fixture-key")}, Authorize: func(governance.Principal, string, string, string) bool { return true }}
	out, e := engine.Run(context.Background(), runtimeRequest(t))
	if e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 2 || serviceCalls.Load() != 1 || out.ExecutionID != "execution-test" || !strings.Contains(out.Content, "276.6") {
		t.Fatalf("engine did not complete model/tool/model loop: %+v", out)
	}
	if e = models.Test(context.Background(), "fixture-model"); e != nil {
		t.Fatal(e)
	}
	if models.List()[1].Status != "VERIFIED" {
		t.Fatal("connection test not recorded")
	}
}
func TestUnselectedServiceAndIdentityInjectionNeverDispatch(t *testing.T) {
	for _, args := range []string{`{"service_id":"unselected","arguments":{"order_id":"SO1002","target_quantity":350}}`, `{"service_id":"replenishment-gap","tenant_id":"other","arguments":{"order_id":"SO1002","target_quantity":350}}`} {
		models, _ := fixtureModel(t, args)
		engine := &Engine{Models: models, Client: &cognitive.Client{BaseURL: "http://127.0.0.1:1"}, Authorize: func(governance.Principal, string, string, string) bool { return true }}
		if _, e := engine.Run(context.Background(), runtimeRequest(t)); e == nil {
			t.Fatal("unsafe tool call accepted")
		}
	}
}
func TestRegistryNoCredentialLeakAndDefaultDeny(t *testing.T) {
	models, _ := fixtureModel(t, "")
	b, _ := json.Marshal(models.List())
	if strings.Contains(string(b), "local-protocol-fixture-key") {
		t.Fatal("credential leaked")
	}
	if _, e := models.Client("unknown"); e == nil {
		t.Fatal("unknown model accepted")
	}
	t.Setenv("COGNARA_TEST_MODEL_KEY", "")
	if _, e := models.Client("fixture-model"); e == nil {
		t.Fatal("missing credential accepted")
	}
	if models.List()[1].Status != "MISSING_CREDENTIAL" {
		t.Fatal("missing credential not visible")
	}
	if _, e := New([]ModelConfig{{ID: "invalid", Name: "bad", Model: "bad", BaseURL: "https://user:password@example.com", APIKeyEnv: "KEY"}}); e == nil {
		t.Fatal("embedded credential URL accepted")
	}
}
