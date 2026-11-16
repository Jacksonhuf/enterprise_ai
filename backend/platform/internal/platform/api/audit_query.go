package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http"
	"time"
)

func (s *Server) auditDatabasePage(w http.ResponseWriter, r *http.Request) (bool, error) {
	p := Principal(r)
	cfg, e := parseGovernanceQuery(r, p.TenantID, "audit")
	if e != nil {
		return true, e
	}
	after := ""
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		if len(raw) > 2048 {
			return true, fail(400, "INVALID_REQUEST", "分页游标无效")
		}
		var c governanceCursor
		b, err := base64.RawURLEncoding.DecodeString(raw)
		if len(raw) > 2048 || err != nil || json.Unmarshal(b, &c) != nil || c.Version != 1 || c.After == "" || c.Filter != cfg.digest {
			return true, fail(400, "INVALID_REQUEST", "分页筛选已变化，请刷新列表")
		}
		after = c.After
	}
	items, more, handled, e := s.Store.QueryMySQLAudit(r.Context(), p.TenantID, after, cfg.limit, func(event app.Event) bool {
		if event.TenantID == "" {
			actor, err := governance.DemoIdentity(event.ActorID)
			if err != nil || actor.TenantID != p.TenantID {
				return false
			}
		} else if event.TenantID != p.TenantID {
			return false
		}
		raw, _ := json.Marshal(event)
		var row map[string]any
		_ = json.Unmarshal(raw, &row)
		return cfg.match(row)
	})
	if !handled {
		return false, e
	}
	if errors.Is(e, app.ErrAuditCursor) {
		return true, fail(400, "INVALID_REQUEST", "分页记录已变化，请刷新列表")
	}
	if e != nil {
		return true, e
	}
	var next any
	if more {
		raw, _ := json.Marshal(governanceCursor{1, items[len(items)-1].ID, cfg.digest})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	Write(w, map[string]any{"items": items, "next_cursor": next, "sampled_at": time.Now().UTC().Format(time.RFC3339), "mode": "validation"})
	return true, nil
}
