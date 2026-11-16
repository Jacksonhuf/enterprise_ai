package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
)

func (s *Server) governanceArrayDatabasePage(w http.ResponseWriter, r *http.Request, section string) (bool, error) {
	p := Principal(r)
	cfg, e := parseGovernanceQuery(r, p.TenantID, section)
	if e != nil {
		return true, e
	}
	after := ""
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		if len(raw) > 2048 {
			return true, fail(400, "INVALID_REQUEST", "分页游标无效")
		}
		var cursor governanceCursor
		b, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || json.Unmarshal(b, &cursor) != nil || cursor.Version != 1 || cursor.After == "" || cursor.Filter != cfg.digest {
			return true, fail(400, "INVALID_REQUEST", "分页筛选已变化，请刷新列表")
		}
		after = cursor.After
	}
	query := s.Store.QueryMySQLExceptions
	if section == "usage" {
		query = s.Store.QueryMySQLUsage
	} else if section != "exceptions" {
		return true, errors.New("invalid governance array")
	}
	items, more, handled, e := query(r.Context(), p.TenantID, after, cfg.limit, cfg.match, governanceRowID)
	if !handled {
		return false, e
	}
	if errors.Is(e, app.ErrGovernancePageCursor) {
		return true, fail(400, "INVALID_REQUEST", "分页记录已变化，请刷新列表")
	}
	if e != nil {
		return true, e
	}
	var next any
	if more {
		var last map[string]any
		if json.Unmarshal(items[len(items)-1], &last) != nil {
			return true, errors.New("invalid governance page")
		}
		id := governanceRowID(last)
		if id == "" {
			return true, fail(503, "CAPABILITY_UNAVAILABLE", "记录缺少稳定标识，无法继续分页")
		}
		raw, _ := json.Marshal(governanceCursor{1, id, cfg.digest})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	Write(w, map[string]any{"items": items, "next_cursor": next, "sampled_at": time.Now().UTC().Format(time.RFC3339), "mode": "validation"})
	return true, nil
}
