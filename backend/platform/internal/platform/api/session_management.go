package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Server) applySessionMutation(ctx context.Context, fn func(*app.State) error) error {
	if ok, e := s.Store.TryMutateSessionsV5(ctx, fn); ok {
		if e != nil {
			if errors.Is(e, app.ErrSessionUnavailable) {
				return fail(404, "ASSET_UNAVAILABLE", "会话不存在或不可访问")
			}
			return e
		}
		return nil
	}
	return s.Store.UpdateOperationalContext(ctx, fn)
}

// Legacy records retain unknown creation dates. Only valid saved timestamps are used.
func sessionMetadata(v app.Session) app.Session {
	latest := time.Time{}
	for _, stamp := range []string{v.CreatedAt, v.UpdatedAt} {
		if t, e := time.Parse(time.RFC3339Nano, strings.ToUpper(stamp)); e == nil && t.After(latest) {
			latest = t
		}
	}
	for _, m := range v.Messages {
		if t, e := time.Parse(time.RFC3339Nano, strings.ToUpper(m.CreatedAt)); e == nil && t.After(latest) {
			latest = t
		}
	}
	v.UpdatedAt = ""
	if !latest.IsZero() {
		v.UpdatedAt = latest.UTC().Format(time.RFC3339Nano)
	}
	if v.Revision == "" {
		raw, _ := json.Marshal([]string{v.TenantID, v.OwnerID, v.ID, v.Title})
		sum := sha256.Sum256(raw)
		v.Revision = "legacy-" + hex.EncodeToString(sum[:])
	}
	return v
}
func (s *Server) renameSession(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	var req struct {
		Title    string `json:"title"`
		Revision string `json:"expected_revision"`
	}
	if e := Decode(r, &req); e != nil {
		return e
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || len([]rune(req.Title)) > 80 || req.Revision == "" {
		return fail(400, "INVALID_REQUEST", "请提供1到80个字符的标题和当前版本")
	}
	var result app.Session
	if e := s.applySessionMutation(r.Context(), func(st *app.State) error {
		v, ok := st.Sessions[r.PathValue("id")]
		if !ok || v.TenantID != p.TenantID || v.OwnerID != p.UserID {
			return fail(404, "ASSET_UNAVAILABLE", "会话不存在或不可访问")
		}
		a, ok := st.Apps[v.AppID]
		if !ok || !s.canUse(*st, p, a) {
			return fail(403, "AUTHORIZATION_DENIED", "应用使用授权已撤销")
		}
		v = sessionMetadata(v)
		if req.Revision != v.Revision {
			return fail(409, "REVISION_CONFLICT", "标题已变化，请重新加载历史后重试")
		}
		if v.Title != req.Title || !v.TitleCustomized {
			v.Title = req.Title
			v.TitleCustomized = true
			v.Revision = app.ID("revision")
			v.UpdatedAt = app.Now()
			st.Sessions[v.ID] = v
			st.Audit = append(st.Audit, app.Event{TenantID: p.TenantID, ID: app.ID("event"), OccurredAt: app.Now(), ActorID: p.UserID, Action: "session.rename", ResourceID: v.ID, Outcome: "success", RequestID: w.Header().Get("X-Request-ID")})
		}
		result = v
		result.Messages = nil
		return nil
	}); e != nil {
		return e
	}
	Write(w, result)
	return nil
}

type sessionCursor struct {
	Scope    string `json:"scope"`
	Snapshot string `json:"snapshot"`
	After    string `json:"after"`
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	q := r.URL.Query()
	appID := q.Get("app_id")
	for _, field := range []string{"app_id", "query", "limit", "cursor"} {
		if len(q[field]) > 1 {
			return fail(400, "INVALID_REQUEST", "查询参数不能重复")
		}
	}
	if appID == "" {
		return fail(400, "INVALID_REQUEST", "请选择应用")
	}
	st := s.Store.ReadOperational()
	a, ok := st.Apps[appID]
	if !ok || a.TenantID != p.TenantID {
		return fail(404, "ASSET_UNAVAILABLE", "应用不存在或不可访问")
	}
	if !s.canUse(st, p, a) {
		return fail(403, "AUTHORIZATION_DENIED", "当前账户没有应用使用权")
	}
	query := strings.ToLower(strings.TrimSpace(q.Get("query")))
	if len([]rune(query)) > 256 {
		return fail(400, "INVALID_REQUEST", "搜索内容不能超过256个字符")
	}
	limit := 20
	if raw := q.Get("limit"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 || n > 100 {
			return fail(400, "INVALID_REQUEST", "每页数量应为1到100")
		}
		limit = n
	}
	items := []app.Session{}
	for _, v := range st.Sessions {
		if v.TenantID == p.TenantID && v.OwnerID == p.UserID && v.AppID == appID && strings.Contains(strings.ToLower(v.Title), query) {
			v = sessionMetadata(v)
			v.Messages = nil
			items = append(items, v)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].ID > items[j].ID
		}
		ti, _ := time.Parse(time.RFC3339Nano, items[i].UpdatedAt)
		tj, _ := time.Parse(time.RFC3339Nano, items[j].UpdatedAt)
		return ti.After(tj)
	})
	raw, _ := json.Marshal([]string{p.TenantID, p.UserID, appID, query})
	sum := sha256.Sum256(raw)
	scope := hex.EncodeToString(sum[:])
	raw, _ = json.Marshal(items)
	sum = sha256.Sum256(raw)
	snapshot := hex.EncodeToString(sum[:])
	start := 0
	if raw := q.Get("cursor"); raw != "" {
		b, e := base64.RawURLEncoding.DecodeString(raw)
		var c sessionCursor
		if e != nil || json.Unmarshal(b, &c) != nil || c.Scope != scope || c.Snapshot != snapshot || c.After == "" {
			return fail(400, "INVALID_REQUEST", "历史列表已变化或游标无效，请重新加载")
		}
		start = -1
		for i, v := range items {
			if v.ID == c.After {
				start = i + 1
				break
			}
		}
		if start < 0 {
			return fail(400, "INVALID_REQUEST", "历史列表已变化，请重新加载")
		}
	}
	end := min(start+limit, len(items))
	var next any
	if end < len(items) {
		b, _ := json.Marshal(sessionCursor{scope, snapshot, items[end-1].ID})
		next = base64.RawURLEncoding.EncodeToString(b)
	}
	Write(w, map[string]any{"items": items[start:end], "next_cursor": next})
	return nil
}
