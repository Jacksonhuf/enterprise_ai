package api

import (
	"encoding/json"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/http"
	"strings"
	"time"
)

func (s *Server) governanceList(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	st := s.Store.Read()
	section := r.PathValue("section")
	var items any = []any{}
	switch section {
	case "users":
		items = []map[string]any{{"id": "demo-user", "name": "张敏", "source": "验证账户", "group": "业务用户组", "status": "启用"}, {"id": "demo-engineer", "name": "应用开发者", "source": "验证账户", "group": "应用构建组", "status": "启用"}, {"id": "demo-admin", "name": "企业与平台管理员", "source": "验证账户", "group": "治理管理组", "status": "启用"}}
	case "groups":
		items = []map[string]any{{"id": "all-users", "name": "业务用户组", "members": 3, "source": "本地验证配置", "status": "启用"}}
	case "clients":
		rows := []map[string]any{}
		for _, a := range st.Apps {
			if a.TenantID == p.TenantID {
				rows = append(rows, map[string]any{"id": a.ID, "name": a.Draft.Basic.Name, "type": "应用", "owner": a.OwnerID, "status": "启用"})
			}
		}
		items = rows
	case "roles":
		items = []map[string]any{{"id": "user", "name": "普通业务用户", "operations": "已授权应用使用", "scope": "当前企业与数据范围"}, {"id": "engineer", "name": "AI应用开发者", "operations": "应用配置、发布与能力引用", "scope": "当前企业"}, {"id": "admin", "name": "企业与平台管理员", "operations": "身份授权与公共配置", "scope": "管理权不等于全部数据权"}}
	case "grants":
		rows := []app.Grant{}
		for _, g := range st.Grants {
			if g.TenantID == p.TenantID {
				rows = append(rows, g)
			}
		}
		items = rows
	case "audit":
		rows := []app.Event{}
		for i := len(st.Audit) - 1; i >= 0; i-- {
			e := st.Audit[i]
			if q := strings.ToLower(r.URL.Query().Get("query")); q != "" && !strings.Contains(strings.ToLower(e.Action+e.ResourceID+e.ActorID+e.ExecutionID), q) {
				continue
			}
			rows = append(rows, e)
		}
		items = rows
	case "exceptions":
		items = st.Exceptions
	case "usage":
		items = st.Usage
	case "status":
		p.Purpose = "application_configuration"
		b, e := s.Client.Do(r.Context(), p, "GET", "/internal/v1/operations/status", nil, "")
		if e != nil {
			Write(w, map[string]any{"sampled_at": app.Now(), "modules": []map[string]any{{"module": "认知业务服务", "status": "UNKNOWN", "summary": "无法取得当前状态"}}, "mode": "validation"})
			return nil
		}
		Write(w, b)
		return nil
	case "settings":
		items = []map[string]any{{"name": "默认语言", "value": "简体中文"}, {"name": "时区", "value": "Asia/Shanghai"}, {"name": "数据标记", "value": "合成验证数据"}}
	default:
		return fail(404, "ASSET_UNAVAILABLE", "管理功能不存在")
	}
	if items == nil {
		items = []any{}
	}
	Write(w, map[string]any{"items": items, "sampled_at": time.Now().UTC().Format(time.RFC3339), "mode": "validation"})
	return nil
}
func (s *Server) createGrant(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	var req struct {
		SubjectType  string `json:"subject_type"`
		SubjectID    string `json:"subject_id"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Action       string `json:"action"`
		Scope        string `json:"scope"`
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	if req.Scope != "east" {
		return fail(400, "INVALID_REQUEST", "验证集当前只支持华东范围")
	}
	valid := false
	switch req.SubjectType {
	case "USER":
		valid = req.SubjectID == "demo-user" || req.SubjectID == "demo-engineer" || req.SubjectID == "demo-admin"
	case "GROUP":
		valid = req.SubjectID == "all-users"
	case "APPLICATION":
		_, valid = s.Store.Read().Apps[req.SubjectID]
	}
	if !valid {
		return fail(400, "INVALID_REQUEST", "授权主体不存在")
	}
	if req.SubjectType == "APPLICATION" {
		switch req.ResourceType {
		case "SERVICE":
			valid = req.ResourceID == "replenishment-gap" && req.Action == "service.invoke"
		case "ASSET":
			valid = (req.ResourceID == "chip-gap-metric" && (req.Action == "execution.invoke" || req.Action == "asset.context.read")) || (req.ResourceID == "inventory-policy" && req.Action == "asset.context.read")
		default:
			valid = false
		}
	} else {
		a, ok := s.Store.Read().Apps[req.ResourceID]
		valid = ok && a.TenantID == p.TenantID && req.ResourceType == "APPLICATION" && req.Action == "application.use"
	}
	if !valid {
		return fail(400, "INVALID_REQUEST", "资源或操作不在当前支持的授权集合内")
	}
	g := app.Grant{ID: app.ID("grant"), TenantID: p.TenantID, SubjectType: req.SubjectType, SubjectID: req.SubjectID, ResourceType: req.ResourceType, ResourceID: req.ResourceID, Action: req.Action, Scope: req.Scope, Active: true}
	if e := s.Store.Update(func(st *app.State) error { st.Grants = append(st.Grants, g); return nil }); e != nil {
		return e
	}
	s.event(p, "authorization.grant", g.ResourceID, "success", "")
	Write(w, g)
	return nil
}
func (s *Server) revokeGrant(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	e := s.Store.Update(func(st *app.State) error {
		for i, g := range st.Grants {
			if g.ID == r.PathValue("id") && g.TenantID == p.TenantID {
				st.Grants[i].Active = false
				return nil
			}
		}
		return fail(404, "ASSET_UNAVAILABLE", "授权记录不存在")
	})
	if e != nil {
		return e
	}
	s.event(p, "authorization.revoke", r.PathValue("id"), "success", "")
	Write(w, map[string]bool{"success": true})
	return nil
}

var settingSections = map[string]bool{"models": true, "model-catalog": true, "mcp": true, "skills": true, "websearch": true, "credentials": true, "storage": true, "vectorstore": true, "queues": true, "system": true, "profile": true, "tenant": true, "api-keys": true, "general": true, "public": true}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	sec := r.PathValue("section")
	if !settingSections[sec] {
		return fail(404, "ASSET_UNAVAILABLE", "配置页面不存在")
	}
	if sec != "profile" && sec != "general" {
		if e := s.requireAdmin(p); e != nil {
			return e
		}
	}
	key := sec
	if sec == "profile" || sec == "general" {
		key = p.UserID + ":" + sec
	}
	value := s.Store.Read().Settings[key]
	if value == nil {
		value = json.RawMessage(`{}`)
	}
	Write(w, map[string]any{"section": sec, "values": value, "mode": "validation", "connections_verified": false})
	return nil
}
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	sec := r.PathValue("section")
	if sec != "profile" && sec != "general" && sec != "public" {
		return fail(503, "CAPABILITY_UNAVAILABLE", "基础连接尚未接入真实提供方；当前只读，不保存秘密或伪报连接成功")
	}
	if sec == "public" {
		if e := s.requireAdmin(p); e != nil {
			return e
		}
	}
	var req struct {
		Values map[string]string `json:"values"`
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	allowed := map[string]bool{"display_name": true, "language": true, "timezone": true, "density": true, "notice": true}
	for k, v := range req.Values {
		if !allowed[k] || len(v) > 512 {
			return fail(400, "INVALID_REQUEST", "包含不允许的配置项")
		}
	}
	b, _ := json.Marshal(req.Values)
	key := sec
	if sec != "public" {
		key = p.UserID + ":" + sec
	}
	if e := s.Store.Update(func(st *app.State) error { st.Settings[key] = b; return nil }); e != nil {
		return e
	}
	s.event(p, "settings.update", sec, "success", "")
	Write(w, map[string]any{"values": req.Values})
	return nil
}
