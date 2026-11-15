// Package mockae is a contract fixture for unavailable A-E services. It is never a production data adapter.
package mockae

import (
	"bytes"
	"encoding/json"
	"fmt"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	CallbackURL, ServiceToken string
	mu                        sync.Mutex
	executions                map[string]map[string]any
	services                  map[string]map[string]any
	requests                  map[string]record
	principals                map[string]string
	contexts                  map[string]contextReceipt
}
type record struct {
	Digest, Operation, ExecutionID, ServiceID string
	Result                                    map[string]any
}

func New(callback, token string) *Server {
	return &Server{CallbackURL: callback, ServiceToken: token, executions: map[string]map[string]any{}, services: map[string]map[string]any{}, requests: map[string]record{}, principals: map[string]string{}, contexts: map[string]contextReceipt{}}
}
func (s *Server) callback(r *http.Request, path string, body any) (map[string]any, error) {
	b, _ := json.Marshal(body)
	req, e := http.NewRequestWithContext(r.Context(), "POST", s.CallbackURL+path, bytes.NewReader(b))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ServiceToken)
	req.Header.Set("X-Delegation-Token", r.Header.Get("X-Delegation-Token"))
	req.Header.Set("X-Request-ID", app.ID("request"))
	if m, ok := body.(map[string]any); ok {
		if id, ok := m["event_id"].(string); ok {
			req.Header.Set("Idempotency-Key", id)
		}
	}
	resp, e := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("go callback %d", resp.StatusCode)
	}
	var out map[string]any
	e = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out)
	return out, e
}
func (s *Server) identity(r *http.Request) (map[string]any, error) {
	purpose := "application_configuration"
	if strings.Contains(r.URL.Path, "/executions") || strings.Contains(r.URL.Path, "service-executions") {
		purpose = "application_turn"
	}
	if strings.HasSuffix(r.URL.Path, "/evidence") {
		purpose = "evidence_read"
	}
	if strings.HasSuffix(r.URL.Path, "/lookup") {
		purpose = "execution_recovery"
	}
	if strings.HasSuffix(r.URL.Path, "/cancel") {
		purpose = "execution_cancel"
	}
	if r.URL.Path == "/internal/v1/assets/discover" || r.URL.Path == "/internal/v1/assets/context" || strings.HasSuffix(r.URL.Path, "/content-projection") {
		if hint := r.Header.Get("X-Invocation-Purpose"); hint == "application_configuration" || hint == "application_turn" {
			purpose = hint
		}
	}
	out, e := s.callback(r, "/internal/v1/identity/introspect", map[string]any{"audience": "cognitive-business", "purpose": purpose})
	if e != nil || out["active"] != true {
		return nil, fmt.Errorf("invalid delegation")
	}
	ctx, ok := out["principal_context"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing principal")
	}
	return ctx, nil
}
func (s *Server) authorize(r *http.Request, kind, id, action string) error {
	v, e := s.callback(r, "/internal/v1/authorization/check", map[string]any{"action": action, "resource": map[string]string{"type": kind, "id": id}, "context": map[string]string{"entrypoint": "APPLICATION_RUNTIME"}})
	if e != nil || v["decision"] != "ALLOW" {
		return fmt.Errorf("denied")
	}
	scopes, ok := v["scope_constraints"].([]any)
	if !ok || len(scopes) == 0 {
		return fmt.Errorf("scope missing")
	}
	first, _ := scopes[0].(map[string]any)
	if first["scope_ref"] != "scope-east-42" {
		return fmt.Errorf("unsupported data scope")
	}
	return nil
}
func errResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	api.Write(w, map[string]any{"request_id": app.ID("request"), "error": map[string]any{"code": code, "message": message, "retryable": false}})
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	add := func(pattern string, h func(http.ResponseWriter, *http.Request, map[string]any)) {
		m.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+s.ServiceToken {
				errResponse(w, 401, "AUTHENTICATION_REQUIRED", "服务身份无效")
				return
			}
			p, e := s.identity(r)
			if e != nil {
				errResponse(w, 401, "AUTHENTICATION_REQUIRED", "委托无效或用途不匹配")
				return
			}
			if p["tenant_id"] != "42" {
				errResponse(w, 403, "AUTHORIZATION_DENIED", "当前企业不在验证数据范围内")
				return
			}
			h(w, r, p)
		})
	}
	m.HandleFunc("GET /internal/v1/operations/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.ServiceToken {
			errResponse(w, 401, "AUTHENTICATION_REQUIRED", "服务身份无效")
			return
		}
		mods := []map[string]any{}
		for _, name := range []string{"A", "B", "C", "D", "E"} {
			mods = append(mods, map[string]any{"module": name, "status": "UP", "summary": "本地合成验证适配器；不是实际生产服务"})
		}
		api.Write(w, map[string]any{"sampled_at": app.Now(), "modules": mods, "integration_capabilities": []map[string]any{{"name": "REFERENCE_VALIDATION", "enabled": true}, {"name": "ATOMIC_IDEMPOTENCY_LOOKUP", "enabled": false, "reason": "验证进程仅有内存执行记录，不具备跨重启原子幂等保证"}, {"name": "EXECUTION_CANCEL", "enabled": false}, {"name": "SERVICE_EXECUTION_CANCEL", "enabled": false}, {"name": "RESOURCE_PROJECTION", "enabled": true}}})
	})
	add("POST /internal/v1/assets/discover", s.assets)
	add("POST /internal/v1/assets/context", s.assetContext)
	add("POST /internal/v1/services/discover", s.discover)
	add("POST /internal/v1/references/validate", s.validate)
	add("POST /internal/v1/executions", s.execute)
	add("POST /internal/v1/service-executions", s.executeService)
	add("GET /internal/v1/executions/{id}/evidence", s.evidence)
	add("GET /internal/v1/service-executions/{id}", s.serviceDetail)
	add("POST /internal/v1/execution-requests/lookup", s.lookup)
	for _, path := range []string{"POST /internal/v1/executions/{id}/cancel", "POST /internal/v1/service-executions/{id}/cancel"} {
		add(path, func(w http.ResponseWriter, r *http.Request, p map[string]any) {
			api.Write(w, map[string]any{"cancel_request_id": app.ID("cancel"), "target_id": r.PathValue("id"), "status": "NOT_SUPPORTED", "message": "当前验证计算同步完成，不支持远端取消；不能声称已经中止"})
		})
	}
	add("GET /internal/v1/resources/{id}/content-projection", func(w http.ResponseWriter, r *http.Request, p map[string]any) {
		if r.PathValue("id") != "inventory-policy" || s.authorize(r, "ASSET", "inventory-policy", "asset.context.read") != nil {
			errResponse(w, 403, "AUTHORIZATION_DENIED", "无权读取文档")
			return
		}
		api.Write(w, map[string]any{"resource_id": "inventory-policy", "resolved_version": "v1", "display_name": "库存管理制度", "projection": map[string]any{"media_type": "text/plain", "text": "库存可用量口径需核对在途与占用；分析不等于交期承诺。", "truncated": true}, "source_references": []map[string]any{{"source_id": "inventory-policy", "locator": "section:3"}}, "redactions": []string{}})
	})
	return m
}

var assetItems = []map[string]any{{"asset_id": "inventory-policy", "asset_type": "KNOWLEDGE", "name": "渠道库存管理制度", "summary": "库存口径、来源与业务边界", "effective_version": "v1"}, {"asset_id": "chip-gap-metric", "asset_type": "METRIC", "name": "物料原始缺口", "summary": "需求减去当前可用量，不纳入替代料及后续到货", "effective_version": "v1"}}

func (s *Server) assets(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var req struct {
		Query  string   `json:"query"`
		Types  []string `json:"asset_types"`
		Domain string   `json:"domain_ref"`
		Limit  *int     `json:"limit"`
		Cursor *string  `json:"cursor"`
	}
	if e := api.Decode(r, &req); e != nil {
		api.ErrorResponse(w, e)
		return
	}
	rows := []map[string]any{}
	for _, a := range assetItems {
		if req.Query != "" && !strings.Contains(a["name"].(string), req.Query) {
			continue
		}
		if len(req.Types) > 0 {
			ok := false
			for _, t := range req.Types {
				if t == a["asset_type"] {
					ok = true
				}
			}
			if !ok {
				continue
			}
		}
		if s.authorize(r, "ASSET", a["asset_id"].(string), "asset.discover") != nil {
			continue
		}
		rows = append(rows, a)
	}
	writeDiscoveryPage(w, rows, req.Limit, req.Cursor)
}
func (s *Server) assetContext(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var req struct {
		Refs  []app.AssetRef `json:"asset_refs"`
		Level string         `json:"level"`
	}
	if e := api.Decode(r, &req); e != nil {
		api.ErrorResponse(w, e)
		return
	}
	if len(req.Refs) < 1 || len(req.Refs) > 20 || (req.Level != "SUMMARY" && req.Level != "STANDARD" && req.Level != "EXECUTION_CONTEXT") {
		errResponse(w, 400, "INVALID_REQUEST", "资产引用数量无效")
		return
	}
	items := []map[string]any{}
	for _, a := range req.Refs {
		if s.authorize(r, "ASSET", a.AssetID, "asset.context.read") != nil {
			errResponse(w, 403, "AUTHORIZATION_DENIED", "无权读取资产内容")
			return
		}
		if a.AssetID != "inventory-policy" && a.AssetID != "chip-gap-metric" {
			errResponse(w, 404, "ASSET_UNAVAILABLE", "资产不存在")
			return
		}
		ready := "READY"
		definition := "原始缺口=max(生产需求×单位用量×损耗系数−当前可用量,0)"
		if a.AssetID == "inventory-policy" {
			ready = "NOT_APPLICABLE"
			definition = "库存可用量口径需核对在途与占用；分析不等于交期承诺。"
		}
		items = append(items, map[string]any{"asset_id": a.AssetID, "resolved_version": "v1", "professional_content": map[string]any{"definition": definition, "as_of": "2026-04-02"}, "execution_readiness": ready, "usage_constraints": []string{"不承诺交期；不纳入替代料与后续到货"}, "related_refs": []any{}})
	}
	receipt := app.ID("context")
	versions := map[string]string{}
	for _, a := range req.Refs {
		versions[a.AssetID] = "v1"
	}
	s.mu.Lock()
	for key, old := range s.contexts {
		if time.Now().After(old.Expires) {
			delete(s.contexts, key)
		}
	}
	if len(s.contexts) >= 10000 {
		s.mu.Unlock()
		errResponse(w, 503, "CAPABILITY_UNAVAILABLE", "上下文凭据容量已满")
		return
	}
	s.contexts[receipt] = contextReceipt{Binding: contextBinding(p), Versions: versions, Expires: time.Now().Add(5 * time.Minute), Level: req.Level}
	s.mu.Unlock()
	api.Write(w, map[string]any{"items": items, "context_receipt": receipt})
}
func (s *Server) discover(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var req struct {
		Channel string  `json:"channel"`
		Query   string  `json:"query"`
		Limit   *int    `json:"limit"`
		Cursor  *string `json:"cursor"`
	}
	if e := api.Decode(r, &req); e != nil {
		api.ErrorResponse(w, e)
		return
	}
	if req.Channel != "INTERNAL_APPLICATION" {
		errResponse(w, 400, "INVALID_REQUEST", "只允许内部应用渠道")
		return
	}
	rows := []any{}
	if s.authorize(r, "SERVICE", "replenishment-gap", "service.discover") == nil && (req.Query == "" || strings.Contains("补货缺口分析", req.Query)) {
		rows = append(rows, map[string]any{"service_id": "replenishment-gap", "service_version": "v1.1", "name": "补货缺口分析", "service_kind": "SCENARIO", "channel": "INTERNAL_APPLICATION", "input_schema": map[string]any{"type": "object", "required": []string{"order_id", "target_quantity"}, "additionalProperties": false, "properties": map[string]any{"order_id": map[string]any{"type": "string", "enum": []string{"SO1002"}}, "target_quantity": map[string]any{"type": "integer", "minimum": 1, "maximum": 100000}}}, "output_schema": map[string]any{"type": "object"}, "contract_revision": "gap-contract-v1"})
	}
	writeDiscoveryPage(w, rows, req.Limit, req.Cursor)
}
func (s *Server) validate(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var req struct {
		Refs []struct {
			Kind     string `json:"kind"`
			ID       string `json:"id"`
			Version  string `json:"version"`
			Contract string `json:"contract_revision"`
		} `json:"references"`
		Revision string `json:"draft_revision"`
		Purpose  string `json:"purpose"`
	}
	if e := api.Decode(r, &req); e != nil {
		api.ErrorResponse(w, e)
		return
	}
	if len(req.Refs) == 0 || req.Purpose != "APPLICATION_RELEASE" || req.Revision == "" {
		errResponse(w, 400, "INVALID_REQUEST", "引用检查参数无效")
		return
	}
	status := "PASS"
	items := []map[string]any{}
	for _, x := range req.Refs {
		valid := (x.Kind == "ASSET" && (x.ID == "inventory-policy" || x.ID == "chip-gap-metric") && x.Version == "v1") || (x.Kind == "SERVICE" && x.ID == "replenishment-gap" && x.Version == "v1.1")
		st := "VALID"
		if !valid {
			st = "VERSION_INCOMPATIBLE"
			status = "BLOCKED"
		}
		if s.authorize(r, x.Kind, x.ID, "reference.validate") != nil {
			st = "DENIED"
			status = "BLOCKED"
		}
		ref := map[string]any{"kind": x.Kind, "id": x.ID, "version": x.Version}
		if x.Contract != "" {
			ref["contract_revision"] = x.Contract
		}
		items = append(items, map[string]any{"reference": ref, "status": st})
	}
	api.Write(w, map[string]any{"validation_id": app.ID("validation"), "draft_revision": req.Revision, "valid_until": time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339), "status": status, "items": items})
}
func decimal(v int) string {
	if v%100 == 0 {
		return strconv.Itoa(v / 100)
	}
	return strings.TrimRight(fmt.Sprintf("%d.%02d", v/100, v%100), "0")
}
func Calculate(quantity int) (map[string]any, error) {
	if quantity < 1 || quantity > 100000 {
		return nil, fmt.Errorf("quantity outside range")
	}
	production := max(quantity-20, 0)
	rows := []any{}
	spec := []struct {
		Name              string
		Factor, Available int
	}{{"芯片", 102, 60}, {"电路板", 101, 320}, {"外壳", 100, 260}, {"测试套件", 100, 220}}
	for _, x := range spec {
		required := production * x.Factor
		gap := max(required-x.Available*100, 0)
		rows = append(rows, []any{x.Name, decimal(required), strconv.Itoa(x.Available), decimal(gap)})
	}
	return map[string]any{"type": "TABLE", "columns": []map[string]any{{"name": "物料", "data_type": "STRING"}, {"name": "需求量", "data_type": "DECIMAL"}, {"name": "当前可用", "data_type": "DECIMAL"}, {"name": "原始缺口", "data_type": "DECIMAL"}}, "rows": rows, "summary": fmt.Sprintf("SO1002目标数量%d，成品可用20，待产%d。芯片需求%s，当前可用60，原始缺口%s。", quantity, production, decimal(production*102), decimal(max(production*102-6000, 0)))}, nil
}
func (s *Server) executeService(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var req struct {
		Ref       app.ServiceRef `json:"service_ref"`
		Channel   string         `json:"channel"`
		Arguments struct {
			Order    string `json:"order_id"`
			Quantity int    `json:"target_quantity"`
		} `json:"arguments"`
		Correlation map[string]string `json:"correlation_context"`
	}
	digest, decodeErr := decodeRequest(r, &req)
	if e := decodeErr; e != nil {
		api.ErrorResponse(w, e)
		return
	}
	if req.Ref.ServiceID != "replenishment-gap" || req.Ref.Version != "v1.1" || req.Channel != "INTERNAL_APPLICATION" {
		errResponse(w, 409, "VERSION_INCOMPATIBLE", "指定服务版本不可用")
		return
	}
	if req.Arguments.Order != "SO1002" {
		errResponse(w, 400, "INVALID_REQUEST", "验证集未包含该订单")
		return
	}
	if s.authorize(r, "SERVICE", req.Ref.ServiceID, "service.invoke") != nil {
		errResponse(w, 403, "AUTHORIZATION_DENIED", "应用未获服务调用权或范围不允许")
		return
	}
	for _, k := range []string{"turn_id", "application_release_id", "assistant_message_id", "tool_call_id"} {
		if req.Correlation[k] == "" {
			errResponse(w, 400, "INVALID_REQUEST", "关联上下文不完整")
			return
		}
	}
	if req.Correlation["turn_id"] != p["turn_id"] || req.Correlation["application_release_id"] != p["application_release_id"] {
		errResponse(w, 403, "AUTHORIZATION_DENIED", "执行关联与委托不一致")
		return
	}
	result, e := Calculate(req.Arguments.Quantity)
	if e != nil {
		errResponse(w, 400, "INVALID_REQUEST", "数量必须为1至100000的正整数")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		errResponse(w, 400, "INVALID_REQUEST", "缺少幂等键")
		return
	}

	owner := fmt.Sprint(p["tenant_id"]) + ":" + fmt.Sprint(p["application_id"]) + ":" + fmt.Sprint(p["actor_user_id"])
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := owner + ":" + key
	if old, ok := s.requests[idx]; ok {
		if old.Digest != digest || old.Operation != "SERVICE_EXECUTION" {
			errResponse(w, 409, "IDEMPOTENCY_CONFLICT", "同键载荷不同")
			return
		}
		api.Write(w, old.Result)
		return
	}
	eid := app.ID("execution")
	sid := app.ID("service_execution")
	dependencies := []map[string]any{{"type": "ASSET", "id": "chip-gap-metric", "version": "v1"}, {"type": "RESOURCE", "id": "supplychain-seed-20260402", "version": "seed-v1"}}
	exec := map[string]any{"execution_id": eid, "status": "SUCCESS", "result": result, "actual_dependencies": dependencies, "evidence_ref": map[string]string{"execution_id": eid}, "error": nil}
	service := map[string]any{"service_execution_id": sid, "status": "SUCCESS", "service_ref": req.Ref, "dependency_snapshot_ref": "validation-snapshot-v1", "result": result, "step_executions": []map[string]string{{"step_id": "gap-calculation", "execution_id": eid, "status": "SUCCESS"}}, "error": nil}
	s.executions[eid] = exec
	s.services[sid] = service
	s.principals[eid] = owner
	s.principals[sid] = owner
	s.requests[idx] = record{digest, "SERVICE_EXECUTION", eid, sid, service}
	// Fixture callbacks exercise the actual G ingress. Stable IDs make a replay deduplicable.
	subject := map[string]any{"tenant_id": p["tenant_id"], "actor_user_id": p["actor_user_id"], "client_id": p["application_id"]}
	_, auditErr := s.callback(r, "/internal/v1/audit-events", map[string]any{"event_id": "audit-" + sid, "event_type": "service.execution.completed", "occurred_at": app.Now(), "producer": "E", "subject": subject, "resource": map[string]string{"type": "SERVICE", "id": req.Ref.ServiceID}, "action": "service.invoke", "outcome": "success", "correlation": map[string]any{"turn_id": p["turn_id"], "execution_id": eid, "service_execution_id": sid}, "evidence_ref": map[string]string{"execution_id": eid}})
	_, usageErr := s.callback(r, "/internal/v1/usage-events", map[string]any{"event_id": "usage-" + sid, "occurred_at": app.Now(), "producer": "E", "tenant_id": p["tenant_id"], "application_id": p["application_id"], "capability": "replenishment-gap", "count": 1, "unit": "CALL", "execution_id": eid})
	if auditErr != nil || usageErr != nil {
		fmt.Printf("validation event delivery incomplete: audit=%v usage=%v\n", auditErr, usageErr)
	}
	api.Write(w, service)
}

func (s *Server) owned(p map[string]any, id string) bool {
	owner := fmt.Sprint(p["tenant_id"]) + ":" + fmt.Sprint(p["application_id"]) + ":" + fmt.Sprint(p["actor_user_id"])
	return s.principals[id] == owner
}
func (s *Server) evidence(w http.ResponseWriter, r *http.Request, p map[string]any) {
	if s.authorize(r, "EXECUTION", r.PathValue("id"), "evidence.read") != nil {
		errResponse(w, 403, "AUTHORIZATION_DENIED", "证据访问被拒绝")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.executions[r.PathValue("id")]
	if !ok || !s.owned(p, r.PathValue("id")) {
		errResponse(w, 404, "ASSET_UNAVAILABLE", "证据不可访问")
		return
	}
	api.Write(w, map[string]any{"execution_id": x["execution_id"], "status": "SUCCESS", "actual_dependencies": x["actual_dependencies"], "source_references": []map[string]any{{"source": "Onto-SupplyChain synthetic seed", "commit": "9599214ca97152a2c7d1cfa2b7fabb114298aad8", "order": "SO1002", "as_of": "2026-04-02"}}, "calculation_basis": map[string]any{"formula": "生产量=max(目标数量−20,0)；芯片需求=生产量×1.02；缺口=max(芯片需求−60,0)", "result": x["result"], "limitations": []string{"未纳入替代料及后续到货", "不构成交期承诺", "当前可用量沿用来源种子口径，不等同当日实物库存"}}, "redactions": []string{}})
}
func (s *Server) serviceDetail(w http.ResponseWriter, r *http.Request, p map[string]any) {
	if s.authorize(r, "SERVICE", "replenishment-gap", "service.invoke") != nil {
		errResponse(w, 403, "AUTHORIZATION_DENIED", "服务授权已撤销")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.services[r.PathValue("id")]
	if !ok || !s.owned(p, r.PathValue("id")) {
		errResponse(w, 404, "ASSET_UNAVAILABLE", "服务执行不可访问")
		return
	}
	api.Write(w, x)
}
func (s *Server) lookup(w http.ResponseWriter, r *http.Request, p map[string]any) {
	var req struct {
		Operation string `json:"operation"`
		Key       string `json:"idempotency_key"`
		Digest    string `json:"request_digest"`
	}
	if e := api.Decode(r, &req); e != nil {
		api.ErrorResponse(w, e)
		return
	}
	if (req.Operation != "DIRECT_EXECUTION" && req.Operation != "SERVICE_EXECUTION") || req.Key == "" || !validDigest(req.Digest) {
		errResponse(w, 400, "INVALID_REQUEST", "恢复参数无效")
		return
	}
	owner := fmt.Sprint(p["tenant_id"]) + ":" + fmt.Sprint(p["application_id"]) + ":" + fmt.Sprint(p["actor_user_id"])
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.requests[owner+":"+req.Key]
	if !ok {
		api.Write(w, map[string]any{"operation": req.Operation, "state": "UNKNOWN", "safe_to_resubmit_same_key": false})
		return
	}
	if x.Digest != req.Digest || x.Operation != req.Operation {
		errResponse(w, 409, "IDEMPOTENCY_CONFLICT", "原请求摘要不一致")
		return
	}
	if x.Operation == "DIRECT_EXECUTION" {
		api.Write(w, map[string]any{"operation": x.Operation, "state": "COMPLETED", "execution_id": x.ExecutionID, "execution_result": x.Result, "safe_to_resubmit_same_key": false})
		return
	}
	api.Write(w, map[string]any{"operation": x.Operation, "state": "COMPLETED", "service_execution_id": x.ServiceID, "service_result": x.Result, "safe_to_resubmit_same_key": false})
}

// The fixture uses numeric offsets; the platform proxy treats real service cursors as opaque.
func writeDiscoveryPage[T any](w http.ResponseWriter, rows []T, requested *int, cursor *string) {
	limit := 20
	if requested != nil {
		limit = *requested
	}
	if limit < 1 || limit > 100 {
		errResponse(w, 400, "INVALID_REQUEST", "每页数量应为1到100")
		return
	}
	start := 0
	if cursor != nil {
		n, err := strconv.Atoi(*cursor)
		if err != nil || n < 0 {
			errResponse(w, 400, "INVALID_REQUEST", "分页游标无效")
			return
		}
		start = n
	}
	if start > len(rows) {
		start = len(rows)
	}
	end := min(start+limit, len(rows))
	var next any
	if end < len(rows) {
		next = strconv.Itoa(end)
	}
	api.Write(w, map[string]any{"items": rows[start:end], "next_cursor": next})
}
