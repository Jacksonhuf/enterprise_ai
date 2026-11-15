package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/types"
	"net/http"
	"strings"
	"sync"
)

type assetTool struct {
	mu   sync.Mutex
	kind string
	run  *serviceTool
}

func (t *assetTool) Name() string { return t.kind }
func (t *assetTool) Description() string {
	switch t.kind {
	case "cognitive_discover_assets":
		return "发现当前发布已引用且获得授权的资产，只返回摘要，不读取正文。"
	case "cognitive_get_asset_context":
		return "读取已引用且已授权资产的正式业务口径、版本与约束，不运行指标。"
	default:
		return "执行已引用正式指标；由服务端先读取当前上下文并绑定上下文凭据。不接受SQL、URL、身份或版本。"
	}
}
func (t *assetTool) Parameters() json.RawMessage {
	switch t.kind {
	case "cognitive_discover_assets":
		return json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"query":{"type":"string","maxLength":200}}}`)
	case "cognitive_get_asset_context":
		return json.RawMessage(`{"type":"object","additionalProperties":false,"required":["asset_id"],"properties":{"asset_id":{"type":"string"}}}`)
	default:
		return json.RawMessage(`{"type":"object","additionalProperties":false,"required":["asset_id","parameters"],"properties":{"asset_id":{"type":"string"},"parameters":{"type":"object","additionalProperties":false,"required":["order_id","target_quantity"],"properties":{"order_id":{"type":"string"},"target_quantity":{"type":"integer","minimum":1,"maximum":100000}}}}}`)
	}
}
func (t *assetTool) allowed(id, action string) bool {
	selected := false
	for _, a := range t.run.request.Release.Config.CapabilityRefs.AssetRefs {
		if a.AssetID == id {
			selected = true
		}
	}
	return selected && t.run.authorize != nil && t.run.authorize(t.run.request.Principal, "ASSET", id, action)
}
func (t *assetTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	bad := func(err error) (*types.ToolResult, error) {
		t.run.mu.Lock()
		t.run.failure = err
		t.run.mu.Unlock()
		return nil, fmt.Errorf("资产能力调用失败；未生成业务结论")
	}
	t.run.mu.Lock()
	if t.run.calls >= 8 {
		t.run.mu.Unlock()
		return bad(fmt.Errorf("tool budget exceeded"))
	}
	t.run.calls++
	t.run.mu.Unlock()
	switch t.kind {
	case "cognitive_discover_assets":
		var in struct {
			Query string `json:"query"`
		}
		if strictDecode(args, &in) != nil || len([]rune(in.Query)) > 200 {
			return bad(fmt.Errorf("invalid discovery parameters"))
		}
		raw, e := t.run.client.Do(ctx, t.run.request.Principal, http.MethodPost, "/internal/v1/assets/discover", map[string]any{"query": in.Query, "limit": 100}, "")
		if e != nil {
			return bad(e)
		}
		var out struct {
			Items []map[string]any `json:"items"`
		}
		if json.Unmarshal(raw, &out) != nil {
			return bad(fmt.Errorf("invalid discovery response"))
		}
		items := []map[string]any{}
		for _, a := range out.Items {
			id, _ := a["asset_id"].(string)
			if t.allowed(id, "asset.discover") {
				items = append(items, a)
				t.record(id, "asset.discover")
			}
		}
		b, _ := json.Marshal(map[string]any{"items": items, "next_cursor": nil})
		return &types.ToolResult{Success: true, Output: string(b)}, nil
	case "cognitive_get_asset_context":
		var in struct {
			ID string `json:"asset_id"`
		}
		if strictDecode(args, &in) != nil || !t.allowed(in.ID, "asset.context.read") {
			return bad(fmt.Errorf("asset not selected or not authorized"))
		}
		raw, e := t.context(ctx, in.ID, "STANDARD")
		if e != nil {
			return bad(e)
		}
		if !t.allowed(in.ID, "asset.context.read") {
			return bad(fmt.Errorf("asset authorization revoked"))
		}
		var response struct {
			Items []map[string]any `json:"items"`
		}
		if json.Unmarshal(raw, &response) != nil || len(response.Items) != 1 || response.Items[0]["asset_id"] != in.ID || response.Items[0]["resolved_version"] == "" {
			return bad(fmt.Errorf("asset context response mismatch"))
		}
		safe, _ := json.Marshal(map[string]any{"items": response.Items})
		t.record(in.ID, "asset.context.read")
		return &types.ToolResult{Success: true, Output: string(safe)}, nil
	default:
		var in struct {
			ID         string `json:"asset_id"`
			Parameters struct {
				Order    string `json:"order_id"`
				Quantity int    `json:"target_quantity"`
			} `json:"parameters"`
		}
		if strictDecode(args, &in) != nil || in.ID != "chip-gap-metric" || in.Parameters.Order == "" || in.Parameters.Quantity < 1 || in.Parameters.Quantity > 100000 || !t.allowed(in.ID, "execution.invoke") || !t.allowed(in.ID, "asset.context.read") {
			return bad(fmt.Errorf("invalid or unauthorized metric invocation"))
		}
		contextRaw, e := t.context(ctx, in.ID, "EXECUTION_CONTEXT")
		if e != nil {
			return bad(e)
		}
		var c struct {
			Receipt string `json:"context_receipt"`
			Items   []struct {
				ID      string `json:"asset_id"`
				Version string `json:"resolved_version"`
				Ready   string `json:"execution_readiness"`
			} `json:"items"`
		}
		if json.Unmarshal(contextRaw, &c) != nil || c.Receipt == "" || len(c.Items) != 1 || c.Items[0].ID != in.ID || c.Items[0].Version == "" || c.Items[0].Ready != "READY" {
			return bad(fmt.Errorf("execution context not ready"))
		}
		id := app.ID("metric")
		body := map[string]any{"capability_type": "BASIC_METRIC", "target": map[string]string{"asset_id": in.ID}, "payload": map[string]any{"parameters": in.Parameters}, "preconditions": map[string]string{"context_receipt": c.Receipt}, "correlation_context": map[string]string{"turn_id": t.run.request.Principal.TurnID, "application_release_id": t.run.request.Release.ID, "assistant_message_id": t.run.request.MessageID, "tool_call_id": id}}
		raw, e := t.run.client.Do(ctx, t.run.request.Principal, http.MethodPost, "/internal/v1/executions", body, id)
		if e != nil {
			return bad(e)
		}
		var out struct {
			ID     string          `json:"execution_id"`
			Status string          `json:"status"`
			Result json.RawMessage `json:"result"`
		}
		if json.Unmarshal(raw, &out) != nil || out.Status != "SUCCESS" || out.ID == "" || !t.allowed(in.ID, "execution.invoke") || !t.allowed(in.ID, "asset.context.read") {
			return bad(fmt.Errorf("metric failed or authorization changed"))
		}
		// Keep the existing message result shape for the UI while retaining the authoritative execution ID.
		b, _ := json.Marshal(map[string]any{"status": out.Status, "result": out.Result, "step_executions": []map[string]string{{"execution_id": out.ID}}, "target_asset_id": in.ID})
		t.run.mu.Lock()
		t.run.last = b
		t.run.executionID = out.ID
		t.run.mu.Unlock()
		t.record(in.ID, "execution.invoke")
		t.record(in.ID, "asset.context.read")
		return &types.ToolResult{Success: true, Output: string(raw)}, nil
	}
}
func (t *assetTool) context(ctx context.Context, id, level string) (json.RawMessage, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("asset required")
	}
	return t.run.client.Do(ctx, t.run.request.Principal, http.MethodPost, "/internal/v1/assets/context", map[string]any{"asset_refs": []app.AssetRef{{AssetID: id}}, "level": level}, "")
}

func (t *assetTool) record(id, action string) {
	t.run.mu.Lock()
	defer t.run.mu.Unlock()
	if t.run.resources == nil {
		t.run.resources = map[string]ResourceCheck{}
	}
	t.run.resources["ASSET:"+id+":"+action] = ResourceCheck{"ASSET", id, action}
}
