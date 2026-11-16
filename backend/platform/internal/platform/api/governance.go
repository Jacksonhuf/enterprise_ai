package api

import (
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http"
	"time"
)

func (s *Server) governanceList(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	if r.PathValue("section") == "audit" {
		if handled, e := s.auditDatabasePage(w, r); handled || e != nil {
			return e
		}
	}
	if r.PathValue("section") == "exceptions" {
		if handled, e := s.exceptionDatabasePage(w, r); handled || e != nil {
			return e
		}
	}
	st := s.Store.ReadOperational()
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
			if e.TenantID != "" && e.TenantID != p.TenantID {
				continue
			}
			if e.TenantID == "" {
				actor, err := governance.DemoIdentity(e.ActorID)
				if err != nil || actor.TenantID != p.TenantID {
					continue
				}
			}
			rows = append(rows, e)
		}
		items = rows
	case "exceptions":
		rows := []map[string]any{}
		for i := len(st.Exceptions) - 1; i >= 0; i-- {
			if st.Exceptions[i]["tenant_id"] == p.TenantID {
				rows = append(rows, st.Exceptions[i])
			}
		}
		items = rows
	case "usage":
		rows := []map[string]any{}
		for _, item := range st.Usage {
			if item["tenant_id"] == p.TenantID {
				rows = append(rows, item)
			}
		}
		items = rows
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
	if pagedGovernanceSections[section] {
		result, err := governancePage(items, r, p.TenantID, section)
		if err != nil {
			return err
		}
		Write(w, result)
		return nil
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
	g := app.Grant{ID: app.ID("grant"), TenantID: p.TenantID, SubjectType: req.SubjectType, SubjectID: req.SubjectID, ResourceType: req.ResourceType, ResourceID: req.ResourceID, Action: req.Action, Scope: req.Scope, Active: true}
	if e := s.Store.UpdateOperationalContext(r.Context(), func(st *app.State) error {
		valid := false
		switch req.SubjectType {
		case "USER":
			subject, err := governance.DemoIdentity(req.SubjectID)
			valid = err == nil && subject.TenantID == p.TenantID
		case "GROUP":
			valid = req.SubjectID == "all-users" && p.TenantID == "42"
		case "APPLICATION":
			subject, ok := st.Apps[req.SubjectID]
			valid = ok && subject.TenantID == p.TenantID
		}
		if !valid {
			return fail(400, "INVALID_REQUEST", "授权主体不存在或不属于当前企业")
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
			resource, ok := st.Apps[req.ResourceID]
			valid = ok && resource.TenantID == p.TenantID && req.ResourceType == "APPLICATION" && req.Action == "application.use"
		}
		if !valid {
			return fail(400, "INVALID_REQUEST", "资源或操作不在当前支持的授权集合内")
		}
		for _, existing := range st.Grants {
			if existing.Active && existing.TenantID == g.TenantID && existing.SubjectType == g.SubjectType && existing.SubjectID == g.SubjectID && existing.ResourceType == g.ResourceType && existing.ResourceID == g.ResourceID && existing.Action == g.Action && existing.Scope == g.Scope {
				g = existing
				return nil
			}
		}
		st.Grants = append(st.Grants, g)
		appendAudit(st, p, "authorization.grant", g.ResourceID, "success", "", requestID(r))
		return nil
	}); e != nil {
		return e
	}
	Write(w, g)
	return nil
}
func (s *Server) revokeGrant(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	e := s.Store.UpdateOperationalContext(r.Context(), func(st *app.State) error {
		for i, g := range st.Grants {
			if g.ID == r.PathValue("id") && g.TenantID == p.TenantID {
				if !g.Active {
					return nil
				}
				st.Grants[i].Active = false
				appendAudit(st, p, "authorization.revoke", g.ID, "success", "", requestID(r))
				return nil
			}
		}
		return fail(404, "ASSET_UNAVAILABLE", "授权记录不存在")
	})
	if e != nil {
		return e
	}
	Write(w, map[string]bool{"success": true})
	return nil
}

var settingSections = map[string]bool{"models": true, "model-catalog": true, "mcp": true, "skills": true, "websearch": true, "credentials": true, "storage": true, "vectorstore": true, "queues": true, "system": true, "profile": true, "tenant": true, "api-keys": true, "general": true, "public": true}
