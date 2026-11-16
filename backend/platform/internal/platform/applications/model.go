package applications

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func ID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b)
}
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

type Basic struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type ModelPrompt struct {
	ModelID string `json:"model_id"`
	Prompt  string `json:"prompt"`
}
type AssetRef struct {
	AssetID string `json:"asset_id"`
}
type ServiceRef struct {
	ServiceID string `json:"service_id"`
	Version   string `json:"version"`
}
type CapabilityRefs struct {
	AssetRefs   []AssetRef   `json:"asset_refs"`
	ServiceRefs []ServiceRef `json:"service_refs"`
	ToolNames   []string     `json:"tool_names"`
}
type ConversationDefaults struct {
	Welcome            string   `json:"welcome"`
	SuggestedQuestions []string `json:"suggested_questions"`
}
type Config struct {
	Basic                Basic                `json:"basic"`
	ModelPrompt          ModelPrompt          `json:"model_prompt"`
	CapabilityRefs       CapabilityRefs       `json:"capability_refs"`
	ConversationDefaults ConversationDefaults `json:"conversation_defaults"`
}
type Release struct {
	ID          string `json:"release_id"`
	AppID       string `json:"app_id"`
	Config      Config `json:"config_snapshot"`
	Note        string `json:"release_note"`
	PublishedAt string `json:"published_at"`
	PublishedBy string `json:"published_by"`
}
type App struct {
	ID               string    `json:"app_id"`
	AgentID          string    `json:"agent_id"`
	TenantID         string    `json:"tenant_id"`
	OwnerID          string    `json:"owner_id"`
	Draft            Config    `json:"draft"`
	Revision         string    `json:"draft_revision"`
	CurrentReleaseID *string   `json:"current_release_id"`
	Releases         []Release `json:"releases"`
	UpdatedAt        string    `json:"updated_at"`
}
type ResourceUse struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Action string `json:"action"`
}
type Message struct {
	Resources            []ResourceUse   `json:"resource_uses,omitempty"`
	AuthorizationProfile int             `json:"authorization_profile,omitempty"`
	ContentRedacted      bool            `json:"content_redacted,omitempty"`
	ID                   string          `json:"id"`
	SessionID            string          `json:"session_id"`
	Role                 string          `json:"role"`
	Content              string          `json:"content"`
	CreatedAt            string          `json:"created_at"`
	ReleaseID            string          `json:"application_release_id,omitempty"`
	TurnID               string          `json:"turn_id,omitempty"`
	ExecutionID          string          `json:"execution_id,omitempty"`
	Result               json.RawMessage `json:"result,omitempty"`
	Completed            bool            `json:"is_completed"`
}
type Session struct {
	CreatedAt       string    `json:"created_at,omitempty"`
	UpdatedAt       string    `json:"updated_at,omitempty"`
	Revision        string    `json:"revision,omitempty"`
	TitleCustomized bool      `json:"title_customized,omitempty"`
	ID              string    `json:"session_id"`
	AppID           string    `json:"app_id"`
	TenantID        string    `json:"tenant_id"`
	OwnerID         string    `json:"owner_id"`
	Title           string    `json:"title"`
	Messages        []Message `json:"messages"`
}
type Check struct {
	ID           string    `json:"check_id"`
	AppID        string    `json:"app_id"`
	Revision     string    `json:"draft_revision"`
	UserID       string    `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	ValidationID string    `json:"validation_id"`
}
type Grant struct {
	ID           string `json:"grant_id"`
	TenantID     string `json:"tenant_id"`
	SubjectType  string `json:"subject_type"`
	SubjectID    string `json:"subject_id"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Action       string `json:"action"`
	Scope        string `json:"scope"`
	Active       bool   `json:"active"`
}
type Event struct {
	TenantID    string `json:"tenant_id,omitempty"`
	ID          string `json:"event_id"`
	OccurredAt  string `json:"occurred_at"`
	ActorID     string `json:"actor_user_id"`
	ClientID    string `json:"client_id"`
	Action      string `json:"action"`
	ResourceID  string `json:"resource_id"`
	Outcome     string `json:"outcome"`
	RequestID   string `json:"request_id"`
	ExecutionID string `json:"execution_id,omitempty"`
}
type State struct {
	Apps             map[string]App             `json:"apps"`
	Sessions         map[string]Session         `json:"sessions"`
	Checks           map[string]Check           `json:"checks"`
	Grants           []Grant                    `json:"grants"`
	Audit            []Event                    `json:"audit"`
	Usage            []map[string]any           `json:"usage"`
	Receipts         map[string]string          `json:"receipts"`
	Settings         map[string]json.RawMessage `json:"settings"`
	SettingRevisions map[string]string          `json:"setting_revisions"`
	Exceptions       []map[string]any           `json:"exceptions"`
}
type Store struct {
	dialect string
	db      *sql.DB
	mu      sync.Mutex
	path    string
	state   State
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if b, e := os.ReadFile(path); e == nil {
		if e = json.Unmarshal(b, &s.state); e != nil {
			return nil, fmt.Errorf("invalid state: %w", e)
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	s.init()
	if len(s.state.Apps) == 0 {
		s.seed()
	}
	return s, nil
}
func (s *Store) init() { initializeState(&s.state) }
func initializeState(state *State) {
	if state.Apps == nil {
		state.Apps = map[string]App{}
	}
	if state.Sessions == nil {
		state.Sessions = map[string]Session{}
	}
	if state.Checks == nil {
		state.Checks = map[string]Check{}
	}
	if state.Settings == nil {
		state.Settings = map[string]json.RawMessage{}
	}
	if state.SettingRevisions == nil {
		state.SettingRevisions = map[string]string{}
	}
	if state.Receipts == nil {
		state.Receipts = map[string]string{}
	}
}
func clone(v State) State {
	b, _ := json.Marshal(v)
	var out State
	_ = json.Unmarshal(b, &out)
	return out
}
func (s *Store) Read() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		st, err := s.readSQL()
		if err != nil {
			return State{}
		}
		return st
	}
	return clone(s.state)
}

// A mutation commits an isolated copy only after durable atomic replacement succeeds.
func (s *Store) Update(fn func(*State) error) error { return s.UpdateContext(context.Background(), fn) }
func (s *Store) UpdateContext(ctx context.Context, fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.db != nil {
		return s.updateSQLContext(ctx, fn)
	}
	next := clone(s.state)
	if err := fn(&next); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.path != "" {
		if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
			return err
		}
		b, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		f, err := os.CreateTemp(filepath.Dir(s.path), ".state-*")
		if err != nil {
			return err
		}
		tmp := f.Name()
		defer os.Remove(tmp)
		if err = f.Chmod(0600); err == nil {
			_, err = f.Write(b)
		}
		if err == nil {
			err = f.Sync()
		}
		ce := f.Close()
		if err == nil {
			err = ce
		}
		if err != nil {
			return err
		}
		if err = os.Rename(tmp, s.path); err != nil {
			return err
		}
	}
	s.state = next
	return nil
}
func (s *Store) seed() {
	cfg := Config{Basic: Basic{"渠道补货助手", "核对订单需求变化、库存及物料原始缺口，并提供可核查依据。"}, ModelPrompt: ModelPrompt{"validation-only", "先核对业务时点，仅使用授权服务。数据不足明确说明，所有结论附依据，不承诺交期。"}, CapabilityRefs: CapabilityRefs{[]AssetRef{{"inventory-policy"}, {"chip-gap-metric"}}, []ServiceRef{{"replenishment-gap", "v1.1"}}, []string{"cognitive_call_service", "cognitive_get_asset_context"}}, ConversationDefaults: ConversationDefaults{"我可以协助核对订单需求变化和补货缺口。", []string{"SO1002需求增加到350后，有多少物料缺口？", "查看本次计算的来源与限制"}}}
	rel := Release{"release-seed-1", "channel-replenishment", cfg, "初始验证场景", Now(), "demo-admin"}
	rid := rel.ID
	s.state.Apps["channel-replenishment"] = App{"channel-replenishment", "agent-validation", "42", "demo-engineer", cfg, "draft-1", &rid, []Release{rel}, Now()}
	s.state.Grants = []Grant{{"grant-metric-context", "42", "APPLICATION", "channel-replenishment", "ASSET", "chip-gap-metric", "asset.context.read", "east", true}, {"grant-app", "42", "GROUP", "all-users", "APPLICATION", "channel-replenishment", "application.use", "east", true}, {"grant-service", "42", "APPLICATION", "channel-replenishment", "SERVICE", "replenishment-gap", "service.invoke", "east", true}, {"grant-asset-1", "42", "APPLICATION", "channel-replenishment", "ASSET", "inventory-policy", "asset.context.read", "east", true}, {"grant-asset-2", "42", "APPLICATION", "channel-replenishment", "ASSET", "chip-gap-metric", "execution.invoke", "east", true}}
}
func Current(a App) (Release, bool) {
	if a.CurrentReleaseID == nil {
		return Release{}, false
	}
	for _, r := range a.Releases {
		if r.ID == *a.CurrentReleaseID {
			return r, true
		}
	}
	return Release{}, false
}
