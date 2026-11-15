package mockae

import (
	"encoding/json"
	"fmt"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/http"
	"time"
)

type contextReceipt struct {
	Binding  string
	Versions map[string]string
	Expires  time.Time
	Level    string
}

func contextBinding(p map[string]any) string {
	b, _ := json.Marshal([]any{p["tenant_id"], p["actor_user_id"], p["application_id"], p["application_release_id"], p["turn_id"]})
	return string(b)
}
func (s *Server) execute(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var in struct {
		Capability string       `json:"capability_type"`
		Target     app.AssetRef `json:"target"`
		Payload    struct {
			Parameters struct {
				Order    string `json:"order_id"`
				Quantity int    `json:"target_quantity"`
			} `json:"parameters"`
		} `json:"payload"`
		Preconditions struct {
			Receipt string `json:"context_receipt"`
		} `json:"preconditions"`
		Options struct {
			MaxRows *int `json:"max_rows"`
		} `json:"execution_options"`
		Correlation struct {
			Turn    string `json:"turn_id"`
			Release string `json:"application_release_id"`
			Message string `json:"assistant_message_id"`
			Tool    string `json:"tool_call_id"`
		} `json:"correlation_context"`
	}
	digest, decodeErr := decodeRequest(r, &in)
	if e := decodeErr; e != nil {
		api.ErrorResponse(w, e)
		return
	}
	if in.Capability != "BASIC_METRIC" || in.Target.AssetID != "chip-gap-metric" {
		errResponse(w, 503, "CAPABILITY_UNAVAILABLE", "当前只实现验证集的正式缺口指标，不支持任意规则或其他能力")
		return
	}
	if in.Payload.Parameters.Order != "SO1002" || (in.Options.MaxRows != nil && (*in.Options.MaxRows < 1 || *in.Options.MaxRows > 1000)) || in.Correlation.Turn != p["turn_id"] || in.Correlation.Release != p["application_release_id"] || in.Correlation.Message == "" || in.Correlation.Tool == "" {
		errResponse(w, 400, "INVALID_REQUEST", "执行参数或关联不匹配")
		return
	}
	if s.authorize(r, "ASSET", in.Target.AssetID, "execution.invoke") != nil || s.authorize(r, "ASSET", in.Target.AssetID, "asset.context.read") != nil {
		errResponse(w, 403, "AUTHORIZATION_DENIED", "缺少指标执行或上下文授权")
		return
	}
	result, e := Calculate(in.Payload.Parameters.Quantity)
	if e != nil {
		errResponse(w, 400, "INVALID_REQUEST", "数量无效")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		errResponse(w, 400, "INVALID_REQUEST", "缺少幂等标识")
		return
	}

	owner := fmt.Sprint(p["tenant_id"]) + ":" + fmt.Sprint(p["application_id"]) + ":" + fmt.Sprint(p["actor_user_id"])
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := owner + ":" + key
	if prior, ok := s.requests[idx]; ok {
		if prior.Digest != digest || prior.Operation != "DIRECT_EXECUTION" {
			errResponse(w, 409, "IDEMPOTENCY_CONFLICT", "相同幂等标识对应不同调用")
			return
		}
		api.Write(w, prior.Result)
		return
	}
	receipt, ok := s.contexts[in.Preconditions.Receipt]
	if !ok || !receipt.valid(p, in.Target.AssetID, time.Now()) {
		errResponse(w, 409, "REFERENCE_NOT_READY", "上下文凭据无效、过期或与当前轮次不一致")
		return
	}
	id := app.ID("execution")
	out := map[string]any{"execution_id": id, "status": "SUCCESS", "result": result, "actual_dependencies": []map[string]string{{"type": "ASSET", "id": in.Target.AssetID, "version": "v1"}, {"type": "RESOURCE", "id": "supplychain-seed-20260402", "version": "seed-v1"}}, "evidence_ref": map[string]string{"execution_id": id}, "error": nil}
	s.executions[id] = out
	s.principals[id] = owner
	s.requests[idx] = record{Digest: digest, Operation: "DIRECT_EXECUTION", ExecutionID: id, Result: out}
	_, _ = s.callback(r, "/internal/v1/audit-events", map[string]any{"event_id": "metric-audit-" + id, "event_type": "metric.execution.completed", "occurred_at": app.Now(), "producer": "E", "subject": map[string]any{"tenant_id": p["tenant_id"], "actor_user_id": p["actor_user_id"], "client_id": p["application_id"]}, "resource": map[string]string{"type": "ASSET", "id": in.Target.AssetID}, "action": "execution.invoke", "outcome": "success", "correlation": map[string]any{"turn_id": p["turn_id"], "execution_id": id}})
	_, _ = s.callback(r, "/internal/v1/usage-events", map[string]any{"event_id": "metric-usage-" + id, "occurred_at": app.Now(), "producer": "E", "tenant_id": p["tenant_id"], "application_id": p["application_id"], "capability": "chip-gap-metric", "count": 1, "unit": "CALL", "execution_id": id})
	api.Write(w, out)
}

func (c contextReceipt) valid(p map[string]any, asset string, now time.Time) bool {
	return c.Binding == contextBinding(p) && now.Before(c.Expires) && c.Level == "EXECUTION_CONTEXT" && c.Versions[asset] == "v1"
}
