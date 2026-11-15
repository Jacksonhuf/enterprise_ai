package api_test

import (
	"bytes"
	"encoding/json"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostgresHTTPSharedPermissionsReplayAndEventRedelivery(t *testing.T) {
	dsn := testsupport.PostgresDSN(t)
	f := setup(t)
	a, e := app.OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := app.OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	f.store = a
	f.server.Store = a
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	other := &fixture{t: t, store: b, host: host, client: f.client, server: second}
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	answer := a.Read().Sessions[id].Messages[1]
	raw := other.request("demo-user", "GET", "/api/v1/messages/"+id+"/load", nil, 200)
	if !bytes.Contains(raw, []byte("276.6")) {
		t.Fatal("second API host did not see persisted result")
	}
	other.request("demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-service", nil, 200)
	raw = f.request("demo-user", "GET", "/api/v1/messages/"+id+"/load", nil, 200)
	if bytes.Contains(raw, []byte("276.6")) {
		t.Fatal("other API host used stale grants")
	}
	replay := "/api/v1/sessions/continue-stream/" + id + "?message_id=" + answer.ID
	f.request("demo-user", "GET", replay, nil, 403)
	other.request("demo-admin", "POST", "/api/v1/cognitive/governance/grants", map[string]any{"subject_type": "APPLICATION", "subject_id": "channel-replenishment", "resource_type": "SERVICE", "resource_id": "replenishment-gap", "action": "service.invoke", "scope": "east"}, 200)
	usage := a.Read().Usage[0]
	a.Close()
	restored, e := app.OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	f.store = restored
	f.server.Store = restored
	f.request("demo-user", "GET", replay, nil, 200)
	if len(restored.Read().Usage) != 1 {
		t.Fatal("replay reran a business execution")
	}
	body, _ := json.Marshal(usage)
	req, _ := http.NewRequest("POST", f.host.URL+"/internal/v1/usage-events", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.server.ServiceToken)
	req.Header.Set("Idempotency-Key", usage["event_id"].(string))
	req.Header.Set("X-Request-ID", "pg-usage-redelivery")
	response, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	out, _ := io.ReadAll(response.Body)
	if response.StatusCode != 200 || !bytes.Contains(out, []byte("DUPLICATE")) || len(restored.Read().Usage) != 1 {
		t.Fatal("Postgres event redelivery was not deduplicated")
	}
}
func TestDatabaseReadinessIsSeparateFromProcessHealth(t *testing.T) {
	f := setup(t)
	db, e := app.OpenSQLite(t.TempDir() + "/readiness.db")
	if e != nil {
		t.Fatal(e)
	}
	f.server.Store = db
	f.store = db
	f.request("demo-user", "GET", "/ready", nil, 200)
	db.Close()
	body := f.request("demo-user", "GET", "/ready", nil, 503)
	if !bytes.Contains(body, []byte("DOWN")) {
		t.Fatal("closed database reported ready")
	}
	f.request("demo-user", "GET", "/health", nil, 200)
	f.request("demo-user", "GET", "/api/v1/cognitive/apps", nil, 503)
}
