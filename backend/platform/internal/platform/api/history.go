package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
)

// completedStream rebuilds a terminal stream from a durable message. It never
// invokes a model, retries a tool, or claims that an unfinished execution completed.
func completedStream(message app.Message, query string) []byte {
	var b bytes.Buffer
	emit := func(kind string, value any) {
		raw, _ := json.Marshal(value)
		fmt.Fprintf(&b, "event: %s\ndata: %s\n\n", kind, raw)
	}
	emit("agent_query", map[string]any{"message_id": message.ID, "query": query, "turn_id": message.TurnID, "application_release_id": message.ReleaseID})
	emit("answer", message)
	emit("done", map[string]any{"message_id": message.ID, "done": true})
	return b.Bytes()
}
func (s *Server) canReadMessage(st app.State, p governance.Principal, session app.Session, message app.Message) bool {
	if message.Role != "assistant" {
		return true
	}
	if !message.Completed {
		return false
	}
	a, ok := st.Apps[session.AppID]
	if !ok || !s.canUse(st, p, a) {
		return false
	}
	p.ApplicationID = session.AppID
	p.ReleaseID = message.ReleaseID
	p.TurnID = message.TurnID
	if message.AuthorizationProfile == 1 {
		for _, resource := range message.Resources {
			if !s.canResource(st, p, resource.Type, resource.ID, resource.Action) {
				return false
			}
		}
		return true
	}
	if message.AuthorizationProfile != 0 {
		return false
	}
	// Legacy completed messages did not retain their consumed-resource set. Prefer
	// their execution provenance, otherwise conservatively use the saved release.
	if len(message.Result) > 0 {
		var meta struct {
			Asset string `json:"target_asset_id"`
		}
		if json.Unmarshal(message.Result, &meta) != nil {
			return false
		}
		if meta.Asset != "" {
			return s.canResource(st, p, "ASSET", meta.Asset, "execution.invoke") && s.canResource(st, p, "ASSET", meta.Asset, "asset.context.read")
		}
		return s.canResource(st, p, "SERVICE", "replenishment-gap", "service.invoke")
	}
	for _, release := range a.Releases {
		if release.ID != message.ReleaseID {
			continue
		}
		for _, name := range release.Config.CapabilityRefs.ToolNames {
			if name == "cognitive_get_asset_context" || name == "cognitive_execute" {
				for _, ref := range release.Config.CapabilityRefs.AssetRefs {
					if !s.canResource(st, p, "ASSET", ref.AssetID, "asset.context.read") {
						return false
					}
				}
			}
		}
		return true
	}
	return false
}
func (s *Server) visibleMessages(p governance.Principal, session app.Session) []app.Message {
	st := s.Store.ReadOperational()
	out := make([]app.Message, 0, len(session.Messages))
	for _, m := range session.Messages {
		if !s.canReadMessage(st, p, session, m) {
			m.Content = "该消息包含当前无权访问的业务内容。"
			m.Result = nil
			m.ExecutionID = ""
			m.Resources = nil
			m.ContentRedacted = true
		}
		out = append(out, m)
	}
	return out
}
