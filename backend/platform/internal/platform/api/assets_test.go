package api_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	agentruntime "github.com/Tencent/WeKnora/internal/platform/adapters/runtime"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricReceiptBindingAndIdempotency(t *testing.T) {
	f := setup(t)
	p, _ := governance.DemoIdentity("demo-user")
	p.ApplicationID = "channel-replenishment"
	p.ReleaseID = "release-seed-1"
	p.TurnID = "metric-turn"
	p.Purpose = "application_turn"
	getContext := func(level string) string {
		t.Helper()
		raw, e := f.client.Do(context.Background(), p, "POST", "/internal/v1/assets/context", map[string]any{"asset_refs": []app.AssetRef{{AssetID: "chip-gap-metric"}}, "level": level}, "")
		if e != nil {
			t.Fatal(e)
		}
		return decode(t, raw)["context_receipt"].(string)
	}
	receipt := getContext("EXECUTION_CONTEXT")
	body := map[string]any{"capability_type": "BASIC_METRIC", "target": map[string]string{"asset_id": "chip-gap-metric"}, "payload": map[string]any{"parameters": map[string]any{"order_id": "SO1002", "target_quantity": 350}}, "preconditions": map[string]string{"context_receipt": receipt}, "correlation_context": map[string]string{"turn_id": p.TurnID, "application_release_id": p.ReleaseID, "assistant_message_id": "metric-message", "tool_call_id": "metric-tool"}}
	run := func(key string) ([]byte, error) {
		return f.client.Do(context.Background(), p, "POST", "/internal/v1/executions", body, key)
	}
	first, e := run("metric-key")
	if e != nil || !bytes.Contains(first, []byte("276.6")) {
		t.Fatalf("%s %v", first, e)
	}
	canonical, _ := json.Marshal(body)
	sum := sha256.Sum256(canonical)
	recovery := p
	recovery.Purpose = "execution_recovery"
	recovered, err := f.client.Do(context.Background(), recovery, "POST", "/internal/v1/execution-requests/lookup", map[string]any{"operation": "DIRECT_EXECUTION", "idempotency_key": "metric-key", "request_digest": fmt.Sprintf("sha256:%x", sum)}, "")
	if err != nil || !bytes.Contains(recovered, []byte("execution_result")) {
		t.Fatalf("recovery digest mismatch: %s %v", recovered, err)
	}
	again, e := run("metric-key")
	if e != nil || !bytes.Equal(first, again) {
		t.Fatal("metric replay changed execution", e)
	}
	if len(f.store.Read().Usage) != 1 {
		t.Fatal("metric replay duplicated billing")
	}
	expect := func(key string, status int) {
		t.Helper()
		_, e := run(key)
		remote, ok := e.(*cognitive.RemoteError)
		if !ok || remote.Status != status {
			t.Fatalf("expected %d got %v", status, e)
		}
	}
	p.Purpose = "application_configuration"
	expect("wrong-delegation-purpose", 401)
	p.Purpose = "application_turn"
	body["payload"] = map[string]any{"parameters": map[string]any{"order_id": "SO1002", "target_quantity": 351}}
	expect("metric-key", 409)
	p.TurnID = "another-turn"
	body["correlation_context"].(map[string]string)["turn_id"] = p.TurnID
	expect("other-turn", 409)
	p.TurnID = "metric-turn"
	body["correlation_context"].(map[string]string)["turn_id"] = p.TurnID
	p.UserID = "demo-engineer"
	expect("other-user", 409)
	p.UserID = "demo-user"
	body["preconditions"] = map[string]string{"context_receipt": "fabricated"}
	expect("forged", 409)
	body["preconditions"] = map[string]string{"context_receipt": getContext("STANDARD")}
	expect("summary-is-not-executable", 409)
	body["preconditions"] = map[string]string{"context_receipt": receipt}
	body["execution_options"] = map[string]int{"max_rows": 0}
	expect("bad-option", 400)
	delete(body, "execution_options")
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-metric-context", nil, 200)
	expect("revoked-context", 403)
}
func TestMetricOnlyAgentEvidenceAndRevocation(t *testing.T) {
	f := setup(t)
	t.Setenv("SSRF_WHITELIST", "127.0.0.1")
	t.Setenv("COGNARA_METRIC_MODEL_KEY", "local-metric-fixture")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []map[string]any `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		hasTool := false
		for _, m := range req.Messages {
			if m["role"] == "tool" {
				hasTool = true
			}
		}
		var delta any
		finish := "stop"
		if !hasTool {
			finish = "tool_calls"
			delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "metric-provider-call", "type": "function", "function": map[string]string{"name": "cognitive_execute", "arguments": `{"asset_id":"chip-gap-metric","parameters":{"order_id":"SO1002","target_quantity":350}}`}}}}
		} else {
			delta = map[string]string{"content": "指标计算已完成，请核对权威结果表及证据。"}
		}
		b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}})
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", b)
	}))
	defer provider.Close()
	models, e := agentruntime.New([]agentruntime.ModelConfig{{ID: "metric-model", Name: "指标模型协议夹具", BaseURL: provider.URL + "/v1", Model: "fixture", APIKeyEnv: "COGNARA_METRIC_MODEL_KEY", HTTP: provider.Client()}})
	if e != nil {
		t.Fatal(e)
	}
	f.server.Models = models
	f.store.Update(func(st *app.State) error {
		a := st.Apps["channel-replenishment"]
		rel, _ := app.Current(a)
		rel.ID = "metric-model-release"
		rel.Config.ModelPrompt.ModelID = "metric-model"
		rel.Config.CapabilityRefs.ToolNames = []string{"cognitive_execute"}
		rel.Config.CapabilityRefs.ServiceRefs = nil
		a.Releases = append(a.Releases, rel)
		a.CurrentReleaseID = &rel.ID
		st.Apps[a.ID] = a
		for i, g := range st.Grants {
			if g.Action == "service.invoke" {
				st.Grants[i].Active = false
			}
		}
		return nil
	})
	id := f.session()
	out := f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "执行SO1002目标350的正式指标"}, 200)
	if !bytes.Contains(out, []byte("276.6")) {
		t.Fatal(string(out))
	}
	msgs := f.store.Read().Sessions[id].Messages
	answer := msgs[1]
	path := "/api/v1/cognitive/evidence/" + answer.ExecutionID + "?session_id=" + id + "&message_id=" + answer.ID
	f.request("demo-user", "GET", path, nil, 200)
	f.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-metric-context", nil, 200)
	f.request("demo-user", "GET", path, nil, 403)
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "再次执行正式指标"}, 503)
	if len(f.store.Read().Sessions[id].Messages) != 2 {
		t.Fatal("denied metric output was persisted")
	}
}
