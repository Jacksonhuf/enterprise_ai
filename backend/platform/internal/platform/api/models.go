package api

import (
	"net/http"
)

func (s *Server) models(w http.ResponseWriter, r *http.Request) error {
	if e := s.requireBuild(Principal(r)); e != nil {
		return e
	}
	Write(w, map[string]any{"items": s.Models.List()})
	return nil
}
func (s *Server) testModel(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	if e := s.requireAdmin(p); e != nil {
		return e
	}
	if err := s.Models.Test(r.Context(), r.PathValue("id")); err != nil {
		s.event(p, "model.connection.test", r.PathValue("id"), "failed", "")
		return fail(503, "CAPABILITY_UNAVAILABLE", "连接测试失败，请检查服务端模型配置及凭据")
	}
	s.event(p, "model.connection.test", r.PathValue("id"), "success", "")
	Write(w, map[string]any{"status": "VERIFIED"})
	return nil
}
func (s *Server) runtimeStatus(w http.ResponseWriter, r *http.Request) error {
	enabled := s.configuredModels()
	Write(w, map[string]any{"mode": "validation", "agent_engine": "weknora-react", "models_configured": enabled})
	return nil
}

func (s *Server) configuredModels() bool {
	for _, m := range s.Models.List() {
		if m.ID != "validation-only" && m.Configured {
			return true
		}
	}
	return false
}
