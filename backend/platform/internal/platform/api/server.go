package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	agentruntime "github.com/Tencent/WeKnora/internal/platform/adapters/runtime"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type principalKey struct{}
type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string                { return e.Message }
func fail(status int, code, msg string) error { return &Error{status, code, msg} }
func Write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func ErrorResponse(w http.ResponseWriter, err error) {
	id := w.Header().Get("X-Request-ID")
	if id == "" {
		id = app.ID("request")
		w.Header().Set("X-Request-ID", id)
	}
	status, code, message := publicError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	Write(w, map[string]any{"request_id": id, "error": map[string]any{"code": code, "message": message, "retryable": false}})
}
func Decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fail(400, "INVALID_REQUEST", "请求参数格式不正确")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fail(400, "INVALID_REQUEST", "请求只能包含一个JSON对象")
	}
	return nil
}
func Principal(r *http.Request) governance.Principal {
	return r.Context().Value(principalKey{}).(governance.Principal)
}

type Handler func(http.ResponseWriter, *http.Request) error
type Server struct {
	Models       *agentruntime.Registry
	Store        *app.Store
	Client       *cognitive.Client
	ServiceToken string
	SigningKey   []byte
	mu           sync.Mutex
	running      map[string]context.CancelFunc
}

func (s *Server) event(p governance.Principal, action, resource, outcome, execution string) {
	_ = s.Store.Update(func(st *app.State) error {
		st.Audit = append(st.Audit, app.Event{TenantID: p.TenantID, ID: app.ID("event"), OccurredAt: app.Now(), ActorID: p.UserID, ClientID: p.ApplicationID, Action: action, ResourceID: resource, Outcome: outcome, RequestID: app.ID("request"), ExecutionID: execution})
		return nil
	})
}
func (s *Server) authenticated(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := governance.DemoIdentity(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			ErrorResponse(w, fail(401, "AUTHENTICATION_REQUIRED", "请先选择验证账户登录"))
			return
		}
		if err = s.Store.Check(); err != nil {
			ErrorResponse(w, fail(503, "CAPABILITY_UNAVAILABLE", "持久化服务不可用"))
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), principalKey{}, p))
		if err = h(w, r); err != nil {
			s.recordFailure(r, p, err)
			ErrorResponse(w, err)
		}
	}
}
func (s *Server) requireBuild(p governance.Principal) error {
	if !p.CanBuild() {
		return fail(403, "AUTHORIZATION_DENIED", "当前账户没有应用构建权限")
	}
	return nil
}
func (s *Server) requireAdmin(p governance.Principal) error {
	if p.Role != "admin" {
		return fail(403, "AUTHORIZATION_DENIED", "当前账户没有治理管理权限")
	}
	return nil
}
func (s *Server) canUse(st app.State, p governance.Principal, a app.App) bool {
	if a.TenantID != p.TenantID || a.CurrentReleaseID == nil {
		return false
	}
	for _, g := range st.Grants {
		if g.Active && g.TenantID == p.TenantID && g.ResourceID == a.ID && g.Action == "application.use" && g.Scope == p.Scope && ((g.SubjectType == "GROUP" && g.SubjectID == "all-users") || (g.SubjectType == "USER" && g.SubjectID == p.UserID)) {
			return true
		}
	}
	return false
}
func (s *Server) CanResource(p governance.Principal, kind, id, action string) bool {
	return s.canResource(s.Store.Read(), p, kind, id, action)
}
func (s *Server) canResource(st app.State, p governance.Principal, kind, id, action string) bool {
	if p.TenantID != "42" || p.Scope != "east" {
		return false
	}
	if p.ApplicationID == "" {
		if !p.CanBuild() {
			return false
		}
		switch kind {
		case "ASSET":
			return (id == "inventory-policy" || id == "chip-gap-metric") && (action == "asset.discover" || action == "asset.context.read" || action == "reference.validate")
		case "SERVICE":
			return id == "replenishment-gap" && (action == "service.discover" || action == "reference.validate")
		}
		return false
	}
	a, ok := st.Apps[p.ApplicationID]
	if !ok || !s.canUse(st, p, a) {
		return false
	}
	if kind == "EXECUTION" {
		reqExecutionID := id
		if action != "evidence.read" {
			return false
		}
		kind = "SERVICE"
		id = "replenishment-gap"
		action = "service.invoke"
		for _, session := range st.Sessions {
			if session.AppID != p.ApplicationID || session.OwnerID != p.UserID || session.TenantID != p.TenantID {
				continue
			}
			for _, m := range session.Messages {
				if m.ExecutionID == reqExecutionID && m.ReleaseID == p.ReleaseID && m.TurnID == p.TurnID {
					var meta struct {
						Asset string `json:"target_asset_id"`
					}
					if json.Unmarshal(m.Result, &meta) == nil && meta.Asset != "" {
						kind = "ASSET"
						id = meta.Asset
						action = "execution.invoke"
						if !s.canResource(st, p, "ASSET", meta.Asset, "asset.context.read") {
							return false
						}
					}
				}
			}
		}
	}
	for _, g := range st.Grants {
		if g.Active && g.TenantID == p.TenantID && g.SubjectType == "APPLICATION" && g.SubjectID == p.ApplicationID && g.ResourceType == kind && g.ResourceID == id && (g.Action == action || (kind == "ASSET" && action == "asset.discover" && (g.Action == "asset.context.read" || g.Action == "execution.invoke"))) && g.Scope == p.Scope {
			return true
		}
	}
	return false
}
func (s *Server) getApp(p governance.Principal, id string, build bool) (app.App, error) {
	a, ok := s.Store.Read().Apps[id]
	if !ok || a.TenantID != p.TenantID {
		return a, fail(404, "ASSET_UNAVAILABLE", "应用不存在或不可访问")
	}
	if build {
		if err := s.requireBuild(p); err != nil {
			return a, err
		}
	} else if !s.canUse(s.Store.Read(), p, a) {
		return a, fail(403, "AUTHORIZATION_DENIED", "当前账户没有应用使用权")
	}
	return a, nil
}
func (s *Server) Handler() http.Handler {
	s.running = map[string]context.CancelFunc{}
	m := http.NewServeMux()
	add := func(p string, h Handler) { m.HandleFunc(p, s.authenticated(h)) }
	m.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if s.Store.Check() != nil {
			w.WriteHeader(503)
			Write(w, map[string]any{"status": "DOWN", "persistence": "DOWN", "mode": "validation"})
			return
		}
		Write(w, map[string]any{"status": "UP", "persistence": "UP", "mode": "validation"})
	})
	m.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		Write(w, map[string]any{"status": "UP", "mode": "validation", "model_execution": s.configuredModels()})
	})
	add("GET /api/v1/cognitive/models", s.models)
	add("POST /api/v1/cognitive/models/{id}/test", s.testModel)
	add("GET /api/v1/cognitive/runtime-status", s.runtimeStatus)
	add("GET /api/v1/cognitive/me", func(w http.ResponseWriter, r *http.Request) error { Write(w, Principal(r)); return nil })
	add("GET /api/v1/cognitive/apps", s.listApps)
	add("POST /api/v1/cognitive/apps", s.createApp)
	add("GET /api/v1/cognitive/apps/{app_id}", s.detail)
	add("PUT /api/v1/cognitive/apps/{app_id}", s.saveDraft)
	add("POST /api/v1/cognitive/apps/{app_id}/release-checks", s.checkRelease)
	add("POST /api/v1/cognitive/apps/{app_id}/releases", s.release)
	add("GET /api/v1/cognitive/apps/{app_id}/releases", s.releases)
	add("POST /api/v1/cognitive/apps/{app_id}/sessions", s.createSession)
	add("GET /api/v1/cognitive/selectors/assets", s.assets)
	add("GET /api/v1/cognitive/selectors/services", s.services)
	add("GET /api/v1/cognitive/evidence/{execution_id}", s.evidence)
	add("GET /api/v1/sessions", s.sessions)
	add("GET /api/v1/sessions/{id}", s.session)
	add("GET /api/v1/messages/{session_id}/load", s.messages)
	add("POST /api/v1/agent-chat/{session_id}", s.chat)
	add("POST /api/v1/sessions/{session_id}/stop", s.stop)
	add("GET /api/v1/sessions/continue-stream/{session_id}", s.continueStream)
	add("GET /api/v1/cognitive/governance/{section}", s.governanceList)
	add("POST /api/v1/cognitive/governance/exceptions/{id}/status", s.updateException)
	add("POST /api/v1/cognitive/governance/grants", s.createGrant)
	add("DELETE /api/v1/cognitive/governance/grants/{id}", s.revokeGrant)
	add("GET /api/v1/cognitive/settings/{section}", s.settings)
	add("PUT /api/v1/cognitive/settings/{section}", s.saveSettings)
	m.Handle("/internal/v1/", s.InternalHandler())
	return s.trace(m)
}
func page[T any](items []T, r *http.Request) (map[string]any, error) {
	limit := 20
	if q := r.URL.Query().Get("limit"); q != "" {
		v, e := strconv.Atoi(q)
		if e != nil || v < 1 || v > 100 {
			return nil, fail(400, "INVALID_REQUEST", "每页数量应为1到100")
		}
		limit = v
	}
	start := 0
	if q := r.URL.Query().Get("cursor"); q != "" {
		v, e := strconv.Atoi(q)
		if e != nil || v < 0 {
			return nil, fail(400, "INVALID_REQUEST", "分页游标无效")
		}
		start = v
	}
	if start > len(items) {
		start = len(items)
	}
	end := min(start+limit, len(items))
	var cursor any
	if end < len(items) {
		cursor = strconv.Itoa(end)
	}
	return map[string]any{"items": items[start:end], "next_cursor": cursor}, nil
}
func (s *Server) listApps(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	manage := r.URL.Query().Get("purpose") == "MANAGE"
	if manage {
		if e := s.requireBuild(p); e != nil {
			return e
		}
	}
	st := s.Store.Read()
	items := []app.App{}
	for _, a := range st.Apps {
		if a.TenantID != p.TenantID {
			continue
		}
		if !manage && !s.canUse(st, p, a) {
			continue
		}
		if !manage {
			rel, _ := app.Current(a)
			a.Draft = app.Config{Basic: rel.Config.Basic, ConversationDefaults: rel.Config.ConversationDefaults}
			a.Revision = ""
			a.Releases = nil
		}
		if q := strings.ToLower(r.URL.Query().Get("query")); q != "" && !strings.Contains(strings.ToLower(a.Draft.Basic.Name+a.Draft.Basic.Description), q) {
			continue
		}
		items = append(items, a)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	out, e := page(items, r)
	if e != nil {
		return e
	}
	Write(w, out)
	return nil
}
func (s *Server) createApp(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireBuild(p); e != nil {
		return e
	}
	var req struct {
		Mode        string `json:"creation_mode"`
		Name        string `json:"name"`
		Description string `json:"description"`
		TemplateID  string `json:"template_id"`
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	if len([]rune(strings.TrimSpace(req.Name))) < 1 || len([]rune(req.Name)) > 80 {
		return fail(400, "INVALID_REQUEST", "请输入1到80字的应用名称")
	}
	cfg := app.Config{Basic: app.Basic{req.Name, req.Description}, ModelPrompt: app.ModelPrompt{"validation-only", "仅使用已授权能力，明确来源与限制。"}, CapabilityRefs: app.CapabilityRefs{AssetRefs: []app.AssetRef{}, ServiceRefs: []app.ServiceRef{}, ToolNames: []string{}}, ConversationDefaults: app.ConversationDefaults{SuggestedQuestions: []string{}}}
	if req.Mode == "TEMPLATE" {
		source, e := s.getApp(p, req.TemplateID, true)
		if e != nil {
			return e
		}
		cfg = source.Draft
		cfg.Basic = app.Basic{req.Name, req.Description}
	} else if req.Mode != "BLANK" {
		return fail(400, "INVALID_REQUEST", "创建方式无效")
	}
	a := app.App{ID: app.ID("app"), AgentID: app.ID("agent"), TenantID: p.TenantID, OwnerID: p.UserID, Draft: cfg, Revision: app.ID("draft"), Releases: []app.Release{}, UpdatedAt: app.Now()}
	if e := s.Store.UpdateContext(r.Context(), func(st *app.State) error { st.Apps[a.ID] = a; return nil }); e != nil {
		return e
	}
	s.event(p, "application.create", a.ID, "success", "")
	Write(w, a)
	return nil
}
func (s *Server) detail(w http.ResponseWriter, r *http.Request) error {
	a, e := s.getApp(Principal(r), r.PathValue("app_id"), true)
	if e != nil {
		return e
	}
	Write(w, a)
	return nil
}
func validateConfig(c app.Config) error {
	if strings.TrimSpace(c.Basic.Name) == "" || len([]rune(c.Basic.Name)) > 80 || strings.TrimSpace(c.ModelPrompt.ModelID) == "" || strings.TrimSpace(c.ModelPrompt.Prompt) == "" || len(c.ModelPrompt.Prompt) > 16000 {
		return fail(400, "INVALID_REQUEST", "请填写名称、一个主模型及有效提示词")
	}
	allowed := map[string]bool{"cognitive_discover_assets": true, "cognitive_get_asset_context": true, "cognitive_execute": true, "cognitive_call_service": true}
	for _, n := range c.CapabilityRefs.ToolNames {
		if !allowed[n] {
			return fail(400, "INVALID_REQUEST", "应用只能使用已登记的受控工具")
		}
	}
	return nil
}
func (s *Server) saveDraft(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if _, e := s.getApp(p, r.PathValue("app_id"), true); e != nil {
		return e
	}
	var req struct {
		Expected string `json:"expected_draft_revision"`
		app.Config
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	if e := validateConfig(req.Config); e != nil {
		return e
	}
	var out app.App
	e := s.Store.UpdateContext(r.Context(), func(st *app.State) error {
		a := st.Apps[r.PathValue("app_id")]
		if a.Revision != req.Expected {
			return fail(409, "VERSION_INCOMPATIBLE", "草稿已被更新，请刷新后比较")
		}
		a.Draft = req.Config
		a.Revision = app.ID("draft")
		a.UpdatedAt = app.Now()
		st.Apps[a.ID] = a
		out = a
		return nil
	})
	if e != nil {
		return e
	}
	s.event(p, "application.draft.save", out.ID, "success", "")
	Write(w, out)
	return nil
}
func (s *Server) checkRelease(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	a, e := s.getApp(p, r.PathValue("app_id"), true)
	if e != nil {
		return e
	}
	var req struct {
		Revision string `json:"expected_draft_revision"`
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	if req.Revision != a.Revision {
		return fail(409, "VERSION_INCOMPATIBLE", "草稿已更新")
	}
	if e = validateConfig(a.Draft); e != nil {
		return e
	}
	if a.Draft.ModelPrompt.ModelID != "validation-only" {
		if _, err := s.Models.Client(a.Draft.ModelPrompt.ModelID); err != nil {
			return fail(409, "REFERENCE_NOT_READY", "模型配置或凭据不可用")
		}
	}
	refs := []map[string]any{}
	p.Purpose = "application_configuration"
	if len(a.Draft.CapabilityRefs.AssetRefs) > 0 {
		raw, err := s.Client.Do(r.Context(), p, "POST", "/internal/v1/assets/context", map[string]any{"asset_refs": a.Draft.CapabilityRefs.AssetRefs, "level": "SUMMARY"}, "")
		if err != nil {
			return err
		}
		var context struct {
			Items []struct {
				ID      string `json:"asset_id"`
				Version string `json:"resolved_version"`
			} `json:"items"`
		}
		if json.Unmarshal(raw, &context) != nil {
			return fail(503, "CAPABILITY_UNAVAILABLE", "资产上下文无效")
		}
		for _, item := range context.Items {
			if item.Version == "" {
				return fail(409, "REFERENCE_NOT_READY", "资产版本未确定")
			}
			refs = append(refs, map[string]any{"kind": "ASSET", "id": item.ID, "version": item.Version})
		}
		if len(context.Items) != len(a.Draft.CapabilityRefs.AssetRefs) {
			return fail(409, "REFERENCE_NOT_READY", "资产引用未全部解析")
		}
	}
	for _, x := range a.Draft.CapabilityRefs.ServiceRefs {
		refs = append(refs, map[string]any{"kind": "SERVICE", "id": x.ServiceID, "version": x.Version})
	}
	if len(refs) == 0 {
		return fail(400, "REFERENCE_NOT_READY", "至少选择一个资产或服务")
	}
	p.Purpose = "application_configuration"
	b, e := s.Client.Do(r.Context(), p, "POST", "/internal/v1/references/validate", map[string]any{"references": refs, "draft_revision": a.Revision, "purpose": "APPLICATION_RELEASE"}, "")
	if e != nil {
		return e
	}
	var v struct {
		Status string `json:"status"`
		ID     string `json:"validation_id"`
	}
	if json.Unmarshal(b, &v) != nil || v.Status != "PASS" {
		Write(w, map[string]any{"status": "BLOCKED", "issues": []string{"引用版本或权限检查未通过"}})
		return nil
	}
	check := app.Check{ID: app.ID("check"), AppID: a.ID, Revision: a.Revision, UserID: p.UserID, ExpiresAt: time.Now().Add(5 * time.Minute), ValidationID: v.ID}
	if e = s.Store.UpdateContext(r.Context(), func(st *app.State) error { st.Checks[check.ID] = check; return nil }); e != nil {
		return e
	}
	Write(w, map[string]any{"check_id": check.ID, "draft_revision": check.Revision, "valid_until": check.ExpiresAt, "status": "PASS", "issues": []string{}})
	return nil
}
func (s *Server) release(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if _, e := s.getApp(p, r.PathValue("app_id"), true); e != nil {
		return e
	}
	var req struct {
		CheckID  string  `json:"check_id"`
		Revision string  `json:"expected_draft_revision"`
		Current  *string `json:"expected_current_release_id"`
		Note     string  `json:"release_note"`
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	var out app.Release
	e := s.Store.UpdateContext(r.Context(), func(st *app.State) error {
		a := st.Apps[r.PathValue("app_id")]
		c, ok := st.Checks[req.CheckID]
		if !ok || c.AppID != a.ID || c.UserID != p.UserID || c.Revision != a.Revision || req.Revision != a.Revision || time.Now().After(c.ExpiresAt) {
			return fail(409, "VERSION_INCOMPATIBLE", "检查已失效，请重新检查当前草稿")
		}
		if (a.CurrentReleaseID == nil) != (req.Current == nil) || (a.CurrentReleaseID != nil && *a.CurrentReleaseID != *req.Current) {
			return fail(409, "VERSION_INCOMPATIBLE", "当前发布已变化，请刷新")
		}
		out = app.Release{ID: app.ID("release"), AppID: a.ID, Config: a.Draft, Note: req.Note, PublishedAt: app.Now(), PublishedBy: p.UserID}
		a.Releases = append(a.Releases, out)
		a.CurrentReleaseID = &out.ID
		st.Apps[a.ID] = a
		delete(st.Checks, req.CheckID)
		appendAudit(st, p, "application.publish", out.AppID, "success", "", requestID(r))
		return nil
	})
	if e != nil {
		return e
	}
	Write(w, out)
	return nil
}
func (s *Server) releases(w http.ResponseWriter, r *http.Request) error {
	a, e := s.getApp(Principal(r), r.PathValue("app_id"), true)
	if e != nil {
		return e
	}
	items := append([]app.Release{}, a.Releases...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].PublishedAt == items[j].PublishedAt {
			return items[i].ID > items[j].ID
		}
		return items[i].PublishedAt > items[j].PublishedAt
	})
	out, e := page(items, r)
	if e != nil {
		return e
	}
	Write(w, out)
	return nil
}
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	a, e := s.getApp(p, r.PathValue("app_id"), false)
	if e != nil {
		return e
	}
	var req struct {
		Title string `json:"title"`
	}
	if e = Decode(r, &req); e != nil {
		return e
	}
	if req.Title == "" {
		req.Title = "新的补货分析"
	}
	sess := app.Session{ID: app.ID("session"), AppID: a.ID, TenantID: p.TenantID, OwnerID: p.UserID, Title: req.Title, Messages: []app.Message{}}
	if e = s.Store.UpdateContext(r.Context(), func(st *app.State) error { st.Sessions[sess.ID] = sess; return nil }); e != nil {
		return e
	}
	Write(w, sess)
	return nil
}
func (s *Server) owned(p governance.Principal, id string) (app.Session, error) {
	v, ok := s.Store.Read().Sessions[id]
	if !ok || v.TenantID != p.TenantID || v.OwnerID != p.UserID {
		return v, fail(404, "ASSET_UNAVAILABLE", "会话不存在或不可访问")
	}
	if _, e := s.getApp(p, v.AppID, false); e != nil {
		return v, e
	}
	return v, nil
}
func (s *Server) sessions(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	items := []app.Session{}
	for _, v := range s.Store.Read().Sessions {
		if v.OwnerID == p.UserID && v.TenantID == p.TenantID && v.AppID == r.URL.Query().Get("app_id") {
			v.Messages = nil
			items = append(items, v)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	Write(w, map[string]any{"items": items})
	return nil
}
func (s *Server) session(w http.ResponseWriter, r *http.Request) error {
	v, e := s.owned(Principal(r), r.PathValue("id"))
	if e != nil {
		return e
	}
	a := s.Store.Read().Apps[v.AppID]
	rel, _ := app.Current(a)
	v.Messages = s.visibleMessages(Principal(r), v)
	Write(w, map[string]any{"session": v, "application": map[string]any{"app_id": a.ID, "name": rel.Config.Basic.Name, "welcome": rel.Config.ConversationDefaults.Welcome, "suggested_questions": rel.Config.ConversationDefaults.SuggestedQuestions}})
	return nil
}
func (s *Server) messages(w http.ResponseWriter, r *http.Request) error {
	v, e := s.owned(Principal(r), r.PathValue("session_id"))
	if e != nil {
		return e
	}
	n := 20
	if q := r.URL.Query().Get("limit"); q != "" {
		if x, e := strconv.Atoi(q); e == nil && x > 0 {
			n = min(x, 200)
		}
	}
	items := []app.Message{}
	before := r.URL.Query().Get("before_time")
	if before != "" {
		if _, e := time.Parse(time.RFC3339Nano, before); e != nil {
			return fail(400, "INVALID_REQUEST", "时间格式无效")
		}
	}
	for _, m := range s.visibleMessages(Principal(r), v) {
		if before == "" || m.CreatedAt < before {
			items = append(items, m)
		}
	}
	if len(items) > n {
		items = items[len(items)-n:]
	}
	Write(w, map[string]any{"success": true, "data": items})
	return nil
}
func (s *Server) assets(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireBuild(p); e != nil {
		return e
	}
	body := map[string]any{}
	for _, k := range []string{"query", "cursor", "domain_ref"} {
		if v := r.URL.Query().Get(k); v != "" {
			body[k] = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		n, e := strconv.Atoi(l)
		if e != nil {
			return fail(400, "INVALID_REQUEST", "分页数量无效")
		}
		body["limit"] = n
	}
	if t := r.URL.Query()["asset_types"]; len(t) > 0 {
		body["asset_types"] = t
	}
	p.Purpose = "application_configuration"
	b, e := s.Client.Do(r.Context(), p, "POST", "/internal/v1/assets/discover", body, "")
	if e != nil {
		return e
	}
	Write(w, b)
	return nil
}
func (s *Server) services(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireBuild(p); e != nil {
		return e
	}
	p.Purpose = "application_configuration"
	body := map[string]any{"channel": "INTERNAL_APPLICATION", "query": r.URL.Query().Get("query")}
	b, e := s.Client.Do(r.Context(), p, "POST", "/internal/v1/services/discover", body, "")
	if e != nil {
		return e
	}
	Write(w, b)
	return nil
}
func (s *Server) evidence(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	v, e := s.owned(p, r.URL.Query().Get("session_id"))
	if e != nil {
		return e
	}
	var found *app.Message
	for _, m := range v.Messages {
		if m.ID == r.URL.Query().Get("message_id") && m.ExecutionID == r.PathValue("execution_id") {
			x := m
			found = &x
		}
	}
	if found == nil {
		return fail(404, "ASSET_UNAVAILABLE", "证据与消息关联不存在")
	}
	p.ApplicationID = v.AppID
	p.ReleaseID = found.ReleaseID
	p.TurnID = found.TurnID
	p.Purpose = "evidence_read"
	b, e := s.Client.Do(r.Context(), p, "GET", "/internal/v1/executions/"+found.ExecutionID+"/evidence", nil, "")
	if e != nil {
		return e
	}
	Write(w, b)
	return nil
}

// Internal endpoints authenticate the calling service independently of browser identities.
func (s *Server) InternalHandler() http.Handler {
	m := http.NewServeMux()
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+s.ServiceToken {
				ErrorResponse(w, fail(401, "AUTHENTICATION_REQUIRED", "服务凭据无效"))
				return
			}
			if s.Store.Check() != nil {
				ErrorResponse(w, fail(503, "CAPABILITY_UNAVAILABLE", "持久化服务不可用"))
				return
			}
			h(w, r)
		}
	}
	m.HandleFunc("POST /internal/v1/identity/introspect", wrap(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Audience string `json:"audience"`
			Purpose  string `json:"purpose"`
		}
		if e := Decode(r, &req); e != nil {
			ErrorResponse(w, e)
			return
		}
		p, e := governance.Verify(r.Header.Get("X-Delegation-Token"), s.SigningKey)
		if e != nil || req.Audience != p.Audience || req.Purpose != p.Purpose {
			Write(w, map[string]any{"active": false})
			return
		}
		var client any
		if p.ApplicationID != "" {
			client = map[string]any{"type": "APPLICATION", "id": p.ApplicationID}
		}
		mode := "DIRECT_USER"
		if client != nil {
			mode = "ON_BEHALF_OF_USER"
		}
		ctx := map[string]any{"tenant_id": p.TenantID, "actor_user_id": p.UserID, "client": client, "mode": mode, "delegation_id": app.ID("delegation"), "audience": p.Audience, "expires_at": time.Unix(p.Expires, 0).UTC().Format(time.RFC3339)}
		if p.ApplicationID != "" {
			ctx["application_id"] = p.ApplicationID
			ctx["application_release_id"] = p.ReleaseID
			ctx["turn_id"] = p.TurnID
		}
		Write(w, map[string]any{"active": true, "principal_context": ctx})
	}))
	m.HandleFunc("POST /internal/v1/authorization/check", wrap(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action   string                    `json:"action"`
			Resource struct{ Type, ID string } `json:"resource"`
			Context  map[string]any            `json:"context"`
		}
		if e := Decode(r, &req); e != nil {
			ErrorResponse(w, e)
			return
		}
		p, e := governance.Verify(r.Header.Get("X-Delegation-Token"), s.SigningKey)
		if e != nil {
			ErrorResponse(w, fail(401, "AUTHENTICATION_REQUIRED", "委托无效"))
			return
		}
		allow := s.CanResource(p, req.Resource.Type, req.Resource.ID, req.Action)
		decision := "DENY"
		scopes := []map[string]string{}
		if allow {
			decision = "ALLOW"
			scopes = append(scopes, map[string]string{"resource_id": req.Resource.ID, "scope_ref": "scope-east-42"})
		}
		Write(w, map[string]any{"decision": decision, "decision_id": app.ID("decision"), "policy_revision": "demo-policy-v1", "scope_constraints": scopes})
	}))
	for _, kind := range []string{"audit-events", "usage-events"} {
		kind := kind
		m.HandleFunc("POST /internal/v1/"+kind, wrap(func(w http.ResponseWriter, r *http.Request) {
			var event map[string]any
			if e := Decode(r, &event); e != nil {
				ErrorResponse(w, e)
				return
			}
			if !validEvent(kind, event) {
				ErrorResponse(w, fail(400, "INVALID_REQUEST", "事件不符合接口契约"))
				return
			}
			id, _ := event["event_id"].(string)
			producer, _ := event["producer"].(string)
			if id == "" || producer != "E" || r.Header.Get("Idempotency-Key") != id {
				ErrorResponse(w, fail(400, "INVALID_REQUEST", "事件来源或标识无效"))
				return
			}
			b, _ := json.Marshal(event)
			sum := sha256.Sum256(b)
			digest := hex.EncodeToString(sum[:])
			receipt := "ACCEPTED"
			e := s.Store.UpdateContext(r.Context(), func(st *app.State) error {
				k := "42:" + kind + ":" + producer + ":" + id
				if old, ok := st.Receipts[k]; ok {
					if old != digest {
						return fail(409, "IDEMPOTENCY_CONFLICT", "同一事件标识对应不同内容")
					}
					receipt = "DUPLICATE"
					return nil
				}
				st.Receipts[k] = digest
				subject, _ := event["subject"].(map[string]any)
				eventTenant := subject["tenant_id"]
				if kind == "usage-events" {
					eventTenant = event["tenant_id"]
				}
				if eventTenant != "42" {
					return fail(403, "AUTHORIZATION_DENIED", "事件主体不在服务授权企业内")
				}
				if kind == "audit-events" {
					actor, _ := subject["actor_user_id"].(string)
					action, _ := event["action"].(string)
					outcome, _ := event["outcome"].(string)
					when, _ := event["occurred_at"].(string)
					client, _ := subject["client_id"].(string)
					resource, _ := event["resource"].(map[string]any)
					resourceID, _ := resource["id"].(string)
					correlation, _ := event["correlation"].(map[string]any)
					execution, _ := correlation["execution_id"].(string)
					st.Audit = append(st.Audit, app.Event{TenantID: "42", ID: id, OccurredAt: when, ActorID: actor, ClientID: client, Action: action, Outcome: outcome, ResourceID: resourceID, ExecutionID: execution, RequestID: requestID(r)})
				}
				if kind == "usage-events" {
					st.Usage = append(st.Usage, event)
				}
				return nil
			})
			if e != nil {
				ErrorResponse(w, e)
				return
			}
			Write(w, map[string]any{"event_id": id, "receipt_status": receipt})
		}))
	}
	m.HandleFunc("POST /internal/v1/config/resolve", wrap(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Keys  []string `json:"keys"`
			Known *string  `json:"known_revision"`
		}
		if e := Decode(r, &req); e != nil {
			ErrorResponse(w, e)
			return
		}
		values := map[string]any{}
		allowed := map[string]any{"default_locale": "zh-CN", "default_timezone": "Asia/Shanghai", "max_result_rows": 100}
		for _, k := range req.Keys {
			v, ok := allowed[k]
			if !ok {
				ErrorResponse(w, fail(403, "AUTHORIZATION_DENIED", "配置键未在公共白名单内"))
				return
			}
			values[k] = v
		}
		Write(w, map[string]any{"revision": "public-config-v1", "values": values})
	}))
	return m
}
