package api

import (
	"context"
	"encoding/json"
	agentruntime "github.com/Tencent/WeKnora/internal/platform/adapters/runtime"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var qtyPattern = regexp.MustCompile(`(?:到|为|至)\s*([+-]?[0-9][0-9.eE,+-]*)`)
var orderPattern = regexp.MustCompile(`SO[0-9]+`)

func (s *Server) chat(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	session, e := s.owned(p, r.PathValue("session_id"))
	if e != nil {
		return e
	}
	var req struct {
		Query        string `json:"query"`
		DisableTitle bool   `json:"disable_title,omitempty"`
		Channel      string `json:"channel,omitempty"`
	}
	if e = Decode(r, &req); e != nil {
		return e
	}
	if strings.TrimSpace(req.Query) == "" || len([]rune(req.Query)) > 4000 {
		return fail(400, "INVALID_REQUEST", "请输入有效业务问题")
	}
	a := s.Store.ReadOperational().Apps[session.AppID]
	release, ok := app.Current(a)
	if !ok {
		return fail(409, "VERSION_INCOMPATIBLE", "应用没有有效发布")
	}
	s.mu.Lock()
	if _, busy := s.running[session.ID]; busy {
		s.mu.Unlock()
		return fail(409, "VERSION_INCOMPATIBLE", "当前会话仍有执行，请等待或停止")
	}
	ctx, cancel := context.WithCancel(r.Context())
	s.running[session.ID] = cancel
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); delete(s.running, session.ID); s.mu.Unlock() }()
	turn := app.ID("turn")
	user := app.Message{ID: app.ID("message"), SessionID: session.ID, Role: "user", Content: req.Query, CreatedAt: app.Now(), Completed: true}
	answer := app.Message{ID: app.ID("message"), SessionID: session.ID, Role: "assistant", CreatedAt: app.Now(), ReleaseID: release.ID, TurnID: turn, Completed: true, AuthorizationProfile: 1}
	p.ApplicationID = a.ID
	p.ReleaseID = release.ID
	p.TurnID = turn
	p.Purpose = "application_turn"
	key := app.ID("call")
	order := orderPattern.FindString(req.Query)
	quantity := 0
	match := qtyPattern.FindStringSubmatch(req.Query)
	if len(match) > 1 {
		quantity, _ = strconv.Atoi(match[1])
	}
	if order == "" || (quantity == 0 && len(match) == 0) {
		for i := len(session.Messages) - 1; i >= 0; i-- {
			m := session.Messages[i]
			if m.Role != "user" {
				continue
			}
			if order == "" {
				order = orderPattern.FindString(m.Content)
			}
			if quantity == 0 && len(match) == 0 {
				v := qtyPattern.FindStringSubmatch(m.Content)
				if len(v) > 1 {
					quantity, _ = strconv.Atoi(v[1])
				}
			}
			if order != "" && quantity != 0 {
				break
			}
		}
	}
	var checks []agentruntime.ResourceCheck
	var result json.RawMessage
	var remoteErr error
	if release.Config.ModelPrompt.ModelID != "validation-only" {
		engine := &agentruntime.Engine{Models: s.Models, Client: s.Client, Authorize: s.CanResource}
		run, err := engine.Run(ctx, agentruntime.Request{Principal: p, Release: release, Session: app.Session{ID: session.ID, AppID: session.AppID, TenantID: session.TenantID, OwnerID: session.OwnerID, Title: session.Title, Messages: s.visibleMessages(p, session)}, MessageID: answer.ID, Query: req.Query})
		if err != nil {

			return fail(503, "CAPABILITY_UNAVAILABLE", "模型或受控工具执行未成功，未生成业务结论")
		}
		answer.Content = run.Content
		answer.Result = run.ServiceResult
		answer.ExecutionID = run.ExecutionID
		result = run.ServiceResult
		checks = run.CheckedResources
	} else if order == "" || (quantity == 0 && len(match) == 0) {
		answer.Content = "请明确订单与目标数量，例如：SO1002需求增加到350后，有多少物料缺口？当前验证只计算需求、库存与原始缺口，不生成交期承诺。"
	} else if order != "SO1002" || quantity < 1 || quantity > 100000 {
		answer.Content = "当前验证集只支持SO1002，目标数量必须为1至100000的正整数。本次未执行计算。"
	} else {
		permitted := false
		for _, tool := range release.Config.CapabilityRefs.ToolNames {
			if tool == "cognitive_call_service" {
				permitted = true
			}
		}
		var ref app.ServiceRef
		for _, x := range release.Config.CapabilityRefs.ServiceRefs {
			if x.ServiceID == "replenishment-gap" {
				ref = x
			}
		}
		if !permitted || ref.ServiceID == "" {
			return fail(403, "AUTHORIZATION_DENIED", "当前发布未选择补货分析服务和受控调用能力")
		}
		body := map[string]any{"service_ref": ref, "channel": "INTERNAL_APPLICATION", "arguments": map[string]any{"order_id": order, "target_quantity": quantity}, "correlation_context": map[string]string{"turn_id": turn, "application_release_id": release.ID, "assistant_message_id": answer.ID, "tool_call_id": key}}
		result, remoteErr = s.Client.Do(ctx, p, "POST", "/internal/v1/service-executions", body, key)
		if remoteErr != nil {
			return remoteErr
		}
		var response struct {
			Status string `json:"status"`
			Result struct {
				Summary string `json:"summary"`
			} `json:"result"`
			Steps []struct {
				ExecutionID string `json:"execution_id"`
			} `json:"step_executions"`
		}
		if json.Unmarshal(result, &response) != nil || response.Status != "SUCCESS" {
			return fail(503, "CAPABILITY_UNAVAILABLE", "服务未完成有效计算，不能生成确定结论")
		}
		answer.Content = response.Result.Summary + "\n\n数据时点：2026-04-02。以上为合成验证数据；未纳入替代料和后续到货，不构成交期承诺。"
		answer.Result = result
		if len(response.Steps) > 0 {
			answer.ExecutionID = response.Steps[0].ExecutionID
		}
	}
	if release.Config.ModelPrompt.ModelID == "validation-only" && result != nil {
		checks = append(checks, agentruntime.ResourceCheck{Type: "SERVICE", ID: "replenishment-gap", Action: "service.invoke"})
	}
	for _, c := range checks {
		if !s.CanResource(p, c.Type, c.ID, c.Action) {
			return fail(403, "AUTHORIZATION_DENIED", "资源授权已变更，请重新核对权限")
		}
	}
	answer.Resources = append([]app.ResourceUse(nil), checks...)
	// Revocation is checked again before persisting or exposing a remote result.
	if _, e = s.owned(p, session.ID); e != nil {
		return e
	}
	if ctx.Err() != nil {
		return fail(409, "EXECUTION_TIMEOUT", "本次接收已停止；远端执行状态须另行确认")
	}
	if e = s.applySessionMutation(r.Context(), func(st *app.State) error {
		current, exists := st.Apps[session.AppID]
		if !exists || !s.canUse(*st, p, current) {
			return fail(403, "AUTHORIZATION_DENIED", "应用使用授权已撤销")
		}
		for _, c := range checks {
			if !s.canResource(*st, p, c.Type, c.ID, c.Action) {
				return fail(403, "AUTHORIZATION_DENIED", "资源授权已撤销")
			}
		}
		v := st.Sessions[session.ID]
		if len(v.Messages) == 0 && !req.DisableTitle && !v.TitleCustomized && (v.Title == "新对话" || v.Title == "新的补货分析") {
			v.Title = string([]rune(req.Query)[:min(len([]rune(req.Query)), 30)])
			v.Revision = app.ID("revision")
		}
		v.Messages = append(v.Messages, user, answer)
		v.UpdatedAt = app.Now()
		st.Sessions[v.ID] = v
		return nil
	}); e != nil {
		return e
	}
	s.event(p, "application.turn", a.ID, "success", answer.ExecutionID)
	stream := completedStream(answer, req.Query)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(stream)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}
func (s *Server) stop(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	v, e := s.owned(p, r.PathValue("session_id"))
	if e != nil {
		return e
	}
	var req struct {
		MessageID string `json:"message_id"`
	}
	if e = Decode(r, &req); e != nil {
		return e
	}
	if req.MessageID == "" {
		return fail(400, "INVALID_REQUEST", "message_id为必填项")
	}
	found := false
	for _, m := range v.Messages {
		if m.ID == req.MessageID && m.Role == "assistant" {
			found = true
		}
	}
	if !found {
		return fail(404, "ASSET_UNAVAILABLE", "消息不存在或不属于当前会话")
	}
	s.mu.Lock()
	cancel := s.running[v.ID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	Write(w, map[string]any{"success": true, "message": "停止接收已处理；本地验证执行通常已结束，不能据此声称远端已取消"})
	return nil
}
func (s *Server) continueStream(w http.ResponseWriter, r *http.Request) error {
	v, e := s.owned(Principal(r), r.PathValue("session_id"))
	if e != nil {
		return e
	}
	id := r.URL.Query().Get("message_id")
	if id == "" {
		return fail(400, "INVALID_REQUEST", "message_id为必填项")
	}
	var found *app.Message
	query := ""
	for i, m := range v.Messages {
		if m.ID == id && m.Role == "assistant" {
			copy := m
			found = &copy
			if i > 0 && v.Messages[i-1].Role == "user" {
				query = v.Messages[i-1].Content
			}
			break
		}
	}
	if found == nil {
		return fail(404, "ASSET_UNAVAILABLE", "消息不存在")
	}
	if !found.Completed {
		return fail(409, "REFERENCE_NOT_READY", "未完成执行不能作为完成消息重放")
	}
	if !s.canReadMessage(s.Store.ReadOperational(), Principal(r), v, *found) {
		return fail(403, "AUTHORIZATION_DENIED", "消息对应资源授权已变更")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(completedStream(*found, query))
	return nil
}
