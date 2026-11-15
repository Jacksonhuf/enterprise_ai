// Package runtime adapts pinned WeKnora model clients and its ReAct engine to the product contract.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	wire "github.com/Tencent/WeKnora/internal/models/api"
	"github.com/Tencent/WeKnora/internal/models/api/openaicompletions"
	chat "github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ModelConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	APIKeyEnv string `json:"api_key_env"`
	// HTTP is dependency injection for local protocol tests; never loaded from JSON.
	HTTP *http.Client `json:"-"`
}
type ModelView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Configured bool   `json:"credential_configured"`
	Status     string `json:"status"`
}
type Registry struct {
	mu      sync.Mutex
	configs []ModelConfig
	status  map[string]string
}

var modelID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var envID = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,127}$`)

func Load(path string) (*Registry, error) {
	if path == "" {
		return New(nil)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var configs []ModelConfig
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e = d.Decode(&configs); e != nil {
		return nil, fmt.Errorf("invalid model registry JSON")
	}
	return New(configs)
}
func New(configs []ModelConfig) (*Registry, error) {
	seen := map[string]bool{}
	for _, c := range configs {
		u, e := url.Parse(c.BaseURL)
		if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || !modelID.MatchString(c.ID) || c.ID == "validation-only" || seen[c.ID] || c.Name == "" || c.Model == "" || !envID.MatchString(c.APIKeyEnv) {
			return nil, fmt.Errorf("invalid or duplicate model configuration")
		}
		seen[c.ID] = true
	}
	return &Registry{configs: append([]ModelConfig(nil), configs...), status: map[string]string{}}, nil
}
func (r *Registry) List() []ModelView {
	out := []ModelView{{"validation-only", "确定性验证 · 不调用模型", false, "VALIDATION"}}
	if r == nil {
		return out
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.configs {
		status := r.status[c.ID]
		if status == "" {
			status = "NOT_TESTED"
		}
		configured := os.Getenv(c.APIKeyEnv) != ""
		if !configured {
			status = "MISSING_CREDENTIAL"
		}
		out = append(out, ModelView{c.ID, c.Name, configured, status})
	}
	return out
}
func (r *Registry) Client(id string) (chat.Chat, error) {
	if r == nil {
		return nil, fmt.Errorf("model unavailable")
	}
	for _, c := range r.configs {
		if c.ID != id {
			continue
		}
		key := os.Getenv(c.APIKeyEnv)
		if key == "" {
			return nil, fmt.Errorf("model credential unavailable")
		}
		return openaicompletions.New(openaicompletions.Config{Endpoint: wire.Endpoint{BaseURL: c.BaseURL, Model: c.Model, ModelID: c.ID, Auth: wire.BearerAuth(key), Client: c.HTTP}}), nil
	}
	return nil, fmt.Errorf("model unavailable")
}
func (r *Registry) Test(ctx context.Context, id string) error {
	client, e := r.Client(id)
	if e == nil {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		var response *types.ChatResponse
		response, e = client.Chat(ctx, []chat.Message{{Role: "user", Content: "Reply with OK."}}, &chat.ChatOptions{})
		if e == nil && (response == nil || strings.TrimSpace(response.Content) == "") {
			e = fmt.Errorf("empty model response")
		}
	}
	if r != nil {
		r.mu.Lock()
		if e == nil {
			r.status[id] = "VERIFIED"
		} else {
			r.status[id] = "FAILED"
		}
		r.mu.Unlock()
	}
	if e != nil {
		return fmt.Errorf("model connection test failed")
	}
	return nil
}
