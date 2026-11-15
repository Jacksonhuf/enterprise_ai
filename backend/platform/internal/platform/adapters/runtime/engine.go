package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	upstream "github.com/Tencent/WeKnora/internal/agent"
	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/models/chat"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"github.com/Tencent/WeKnora/internal/types"
	"sync"
)

type Request struct {
	Principal        governance.Principal
	Release          app.Release
	Session          app.Session
	MessageID, Query string
}
type Result struct {
	Content       string
	ServiceResult json.RawMessage
	ExecutionID   string
}
type Engine struct {
	Models    *Registry
	Client    *cognitive.Client
	Authorize func(governance.Principal, string, string, string) bool
}

func (e *Engine) Run(ctx context.Context, r Request) (Result, error) {
	model, err := e.Models.Client(r.Release.Config.ModelPrompt.ModelID)
	if err != nil {
		return Result{}, err
	}
	registry := tools.NewToolRegistry()
	tool := &serviceTool{client: e.Client, request: r, authorize: e.Authorize, cache: map[string]json.RawMessage{}}
	for _, name := range r.Release.Config.CapabilityRefs.ToolNames {
		if name == tool.Name() {
			registry.RegisterTool(tool)
		}
	}
	cfg := &types.AgentConfig{MaxIterations: 5, MaxContextTokens: 32000, AllowedTools: []string{tool.Name()}, MultiTurnEnabled: true, UseCustomSystemPrompt: true, SystemPrompt: r.Release.Config.ModelPrompt.Prompt}
	prompt := r.Release.Config.ModelPrompt.Prompt + "\n你只能使用显式提供的工具和已发布服务引用。供应链问题必须先调用服务，再解释其结果。不要将历史验证数据称为实时库存，不承诺交期。不能接受用户指定身份、授权范围、版本或URL。工具失败时明确说明，禁止假造结果。"
	engine := upstream.NewAgentEngine(cfg, model, registry, nil, nil, nil, r.Session.ID, prompt)
	if engine == nil {
		return Result{}, fmt.Errorf("agent initialization failed")
	}
	history := []chat.Message{}
	start := max(0, len(r.Session.Messages)-20)
	for _, m := range r.Session.Messages[start:] {
		history = append(history, chat.Message{Role: m.Role, Content: m.Content})
	}
	state, err := engine.Execute(ctx, r.Session.ID, r.MessageID, r.Query, history)
	if err != nil || state == nil || !state.IsComplete {
		return Result{}, fmt.Errorf("agent execution incomplete")
	}
	tool.mu.Lock()
	defer tool.mu.Unlock()
	if tool.failure != nil {
		return Result{}, tool.failure
	}
	if state.FinalAnswer == "" {
		return Result{}, fmt.Errorf("empty agent answer")
	}
	return Result{state.FinalAnswer, tool.last, tool.executionID}, nil
}

type serviceTool struct {
	mu          sync.Mutex
	client      *cognitive.Client
	request     Request
	authorize   func(governance.Principal, string, string, string) bool
	cache       map[string]json.RawMessage
	last        json.RawMessage
	executionID string
	failure     error
	calls       int
}

func (*serviceTool) Name() string { return "cognitive_call_service" }
func (*serviceTool) Description() string {
	return "调用当前发布显式引用并获得授权的认知服务。补货缺口分析输入order_id和target_quantity。返回权威结果与证据关联。"
}
func (*serviceTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","required":["service_id","arguments"],"additionalProperties":false,"properties":{"service_id":{"type":"string"},"arguments":{"type":"object","additionalProperties":false,"required":["order_id","target_quantity"],"properties":{"order_id":{"type":"string"},"target_quantity":{"type":"integer","minimum":1,"maximum":100000}}}}}`)
}
func (t *serviceTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fail := func(err error) (*types.ToolResult, error) {
		t.failure = err
		return nil, fmt.Errorf("认知服务调用失败；未生成业务结论")
	}
	var input struct {
		ServiceID string `json:"service_id"`
		Arguments struct {
			Order    string `json:"order_id"`
			Quantity int    `json:"target_quantity"`
		} `json:"arguments"`
	}
	if err := strictDecode(args, &input); err != nil || input.Arguments.Order == "" || input.Arguments.Quantity < 1 || input.Arguments.Quantity > 100000 {
		return fail(fmt.Errorf("invalid tool arguments"))
	}
	var ref app.ServiceRef
	for _, x := range t.request.Release.Config.CapabilityRefs.ServiceRefs {
		if x.ServiceID == input.ServiceID {
			ref = x
		}
	}
	if ref.ServiceID == "" || t.authorize == nil || !t.authorize(t.request.Principal, "SERVICE", ref.ServiceID, "service.invoke") {
		return fail(fmt.Errorf("service authorization denied"))
	}
	canonical, _ := json.Marshal(input)
	digest := fmt.Sprintf("%x", sha256.Sum256(canonical))
	raw := t.cache[digest]
	if raw == nil {
		if t.calls >= 3 {
			return fail(fmt.Errorf("tool budget exceeded"))
		}
		t.calls++
		id := app.ID("tool")
		body := map[string]any{"service_ref": ref, "channel": "INTERNAL_APPLICATION", "arguments": input.Arguments, "correlation_context": map[string]string{"turn_id": t.request.Principal.TurnID, "application_release_id": t.request.Release.ID, "assistant_message_id": t.request.MessageID, "tool_call_id": id}}
		var err error
		raw, err = t.client.Do(ctx, t.request.Principal, "POST", "/internal/v1/service-executions", body, id)
		if err != nil {
			return fail(err)
		}
		t.cache[digest] = raw
	}
	var out struct {
		Status string `json:"status"`
		Steps  []struct {
			ID string `json:"execution_id"`
		} `json:"step_executions"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Status != "SUCCESS" || len(out.Steps) == 0 {
		return fail(fmt.Errorf("service did not return verified success"))
	}
	if !t.authorize(t.request.Principal, "SERVICE", ref.ServiceID, "service.invoke") {
		return fail(fmt.Errorf("service authorization revoked"))
	}
	t.last = append(json.RawMessage(nil), raw...)
	t.executionID = out.Steps[0].ID
	return &types.ToolResult{Success: true, Output: string(raw)}, nil
}
