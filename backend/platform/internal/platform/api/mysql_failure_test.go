package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLHTTPFailurePreservesDomainsAndBindsExceptionAudit(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			store, e := app.OpenMySQL(dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer store.Close()
			if version == 2 {
				if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
					t.Fatal(e)
				}
			}
			f := setup(t)
			f.store = store
			f.server.Store = store
			db, e := sql.Open("mysql", dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.apps."channel-replenishment".future',CAST(? AS JSON),'$.exceptions',NULL)`, `{"n":9007199254740993}`); e != nil {
				t.Fatal(e)
			}
			req, _ := http.NewRequest("POST", f.host.URL+"/api/v1/cognitive/models/missing/test", bytes.NewBufferString(`{}`))
			req.Header.Set("Authorization", "Bearer demo-admin")
			req.Header.Set("X-Request-ID", "forged-browser-id")
			response, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := io.ReadAll(response.Body)
			response.Body.Close()
			out := decode(t, raw)
			id := out["request_id"].(string)
			if response.StatusCode != 503 || id == "forged-browser-id" || id != response.Header.Get("X-Request-ID") {
				t.Fatal("failure trace changed")
			}
			st := store.Read()
			if len(st.Exceptions) != 1 || len(st.Audit) != 2 {
				t.Fatal("failure not persisted atomically")
			}
			exception := st.Exceptions[0]
			var event app.Event
			for _, candidate := range st.Audit {
				if candidate.RequestID == id {
					event = candidate
				}
			}
			if exception["request_id"] != id || event.RequestID != id || exception["status"] != "OPEN" || event.Outcome != "failed" || exception["tenant_id"] != "42" || strings.Contains(exception["route"].(string), "missing") {
				t.Fatal("failure metadata changed")
			}
			// Only v2 has the targeted preservation guarantee; v1 deliberately retains its old path.
			if version == 2 {
				var future string
				if e = db.QueryRow(`SELECT JSON_EXTRACT(body,'$.apps."channel-replenishment".future') FROM cognara_fg_state WHERE id=1`).Scan(&future); e != nil || !strings.Contains(future, "9007199254740993") {
					t.Fatal("5xx recording rewrote other domains", e)
				}
			}
			list := decode(t, f.request("demo-admin", "GET", "/api/v1/cognitive/governance/exceptions", nil, 200))["items"].([]any)
			if len(list) != 1 {
				t.Fatal("record not visible")
			}
			if len(exception["revision"].(string)) == 0 {
				t.Fatal("exception revision missing")
			}
			// Persisted state keeps the exception/audit correlation after reopening the database.
			reopened, e := app.OpenMySQL(dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer reopened.Close()
			state := reopened.Read()
			encoded, _ := json.Marshal(state.Exceptions)
			if len(state.Audit) != 2 || !bytes.Contains(encoded, []byte(id)) {
				t.Fatal("failure association lost on reopen")
			}
		})
	}
}
