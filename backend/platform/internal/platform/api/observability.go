package api

import (
	"context"
	"encoding/json"
	"errors"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http"
	"regexp"
	"strings"
)

type requestIDKey struct{}

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func requestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey{}).(string)
	return id
}
func (s *Server) trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := app.ID("request")
		if strings.HasPrefix(r.URL.Path, "/internal/v1/") && r.Header.Get("Authorization") == "Bearer "+s.ServiceToken && validRequestID.MatchString(r.Header.Get("X-Request-ID")) {
			id = r.Header.Get("X-Request-ID")
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}
func publicError(err error) (int, string, string) {
	var local *Error
	if errors.As(err, &local) {
		return local.Status, local.Code, local.Message
	}
	var remote *cognitive.RemoteError
	if errors.As(err, &remote) {
		status := remote.Status
		if status < 400 || status > 599 {
			status = 503
		}
		code := "CAPABILITY_UNAVAILABLE"
		switch status {
		case 400:
			code = "INVALID_REQUEST"
		case 401:
			code = "AUTHENTICATION_REQUIRED"
		case 403:
			code = "AUTHORIZATION_DENIED"
		case 404:
			code = "ASSET_UNAVAILABLE"
		case 409:
			code = "VERSION_INCOMPATIBLE"
		}
		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(remote.Body), &envelope) == nil {
			if knownPublicCodes[envelope.Error.Code] {
				code = envelope.Error.Code
			}
		}
		return status, code, "认知服务未完成请求，请核对权限、参数或关联请求"
	}
	return 503, "CAPABILITY_UNAVAILABLE", "服务暂时不可用，请稍后重试"
}

var knownPublicCodes = map[string]bool{"INVALID_REQUEST": true, "AUTHENTICATION_REQUIRED": true, "AUTHORIZATION_DENIED": true, "ASSET_UNAVAILABLE": true, "VERSION_INCOMPATIBLE": true, "REFERENCE_NOT_READY": true, "IDEMPOTENCY_CONFLICT": true, "CAPABILITY_UNAVAILABLE": true, "EXECUTION_TIMEOUT": true}

func appendAudit(st *app.State, p governance.Principal, action, resource, outcome, execution, id string) {
	if id == "" {
		id = app.ID("request")
	}
	st.Audit = append(st.Audit, app.Event{TenantID: p.TenantID, ID: app.ID("event"), OccurredAt: app.Now(), ActorID: p.UserID, ClientID: p.ApplicationID, Action: action, ResourceID: resource, Outcome: outcome, ExecutionID: execution, RequestID: id})
}
func (s *Server) recordFailure(r *http.Request, p governance.Principal, err error) {
	status, code, _ := publicError(err)
	outcome := "denied"
	if status >= 500 {
		outcome = "failed"
	}
	if r.Context().Err() != nil {
		outcome = "canceled"
	}
	events := app.State{}
	appendAudit(&events, p, r.Pattern, "", outcome, "", requestID(r))
	if status < 500 {
		// Expected request failures only append audit; do not rewrite unrelated domains.
		if handled, _ := s.Store.AppendMySQLAuditContext(context.Background(), events.Audit); handled {
			return
		}
	}
	_ = s.Store.UpdateOperational(func(st *app.State) error {
		st.Audit = append(st.Audit, events.Audit...)
		if status >= 500 && r.Context().Err() == nil {
			st.Exceptions = append(st.Exceptions, map[string]any{"id": app.ID("exception"), "tenant_id": p.TenantID, "name": "服务请求未完成", "module": "应用运行与配置", "request_id": requestID(r), "code": code, "route": r.Pattern, "http_status": status, "status": "OPEN", "revision": app.ID("exception_revision"), "occurred_at": app.Now(), "updated_at": app.Now()})
		}
		return nil
	})
}
func (s *Server) updateException(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	var in struct {
		Expected string `json:"expected_revision"`
		Status   string `json:"status"`
	}
	if e := Decode(r, &in); e != nil {
		return e
	}
	if in.Expected == "" || (in.Status != "ACKNOWLEDGED" && in.Status != "RESOLVED" && in.Status != "OPEN") {
		return fail(400, "INVALID_REQUEST", "异常状态参数无效")
	}
	var out map[string]any
	e := s.Store.UpdateOperationalContext(r.Context(), func(st *app.State) error {
		for i, item := range st.Exceptions {
			if item["id"] != r.PathValue("id") || item["tenant_id"] != p.TenantID {
				continue
			}
			if item["revision"] != in.Expected {
				return fail(409, "VERSION_INCOMPATIBLE", "异常记录已变更，请刷新")
			}
			old, _ := item["status"].(string)
			valid := old == "OPEN" && (in.Status == "ACKNOWLEDGED" || in.Status == "RESOLVED") || old == "ACKNOWLEDGED" && in.Status == "RESOLVED" || old == "RESOLVED" && in.Status == "OPEN"
			if !valid {
				return fail(409, "VERSION_INCOMPATIBLE", "不支持该异常状态迁移")
			}
			item["status"] = in.Status
			item["revision"] = app.ID("exception_revision")
			item["updated_at"] = app.Now()
			item["updated_by"] = p.UserID
			st.Exceptions[i] = item
			out = item
			appendAudit(st, p, "exception."+strings.ToLower(in.Status), r.PathValue("id"), "success", "", requestID(r))
			return nil
		}
		return fail(404, "ASSET_UNAVAILABLE", "异常记录不存在")
	})
	if e != nil {
		return e
	}
	Write(w, out)
	return nil
}
