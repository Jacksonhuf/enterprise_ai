package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMySQLV3ApplicationEntryThroughStore(t *testing.T) {
	ctx := context.Background()
	s, db, dsn := stageMySQLV3Writers(t)
	base := readV3(t, s).Apps["channel-replenishment"]
	tenant, actor := base.TenantID, "demo-engineer"
	client, request := "client-entry", "request-entry"

	cfg := base.Draft
	cfg.Basic = Basic{Name: "入口应用", Description: "schema3"}
	created, ok, err := s.TryCreateApplicationV3(ctx, tenant, actor, cfg, "", nil)
	if err != nil || !ok || created.Draft.Basic.Name != "入口应用" {
		t.Fatal("create", ok, err)
	}
	if n := auditActionCount(t, db, "application.create"); n != 1 {
		t.Fatal("create audit count", n)
	}

	cfg.Basic.Name = "入口应用v2"
	saved, ok, err := s.TrySaveApplicationDraftV3(ctx, tenant, created.ID, actor, client, request, created.Revision, cfg)
	if err != nil || !ok || saved.Revision == created.Revision || saved.Draft.Basic.Name != "入口应用v2" {
		t.Fatal("save", ok, err, saved.Revision)
	}
	ev := lastAuditForResource(t, db, saved.ID)
	if ev.ClientID != client || ev.RequestID != request || ev.Action != "application.draft.save" {
		t.Fatal("draft audit metadata", ev)
	}

	if _, ok, err = s.TryCreateApplicationV3(ctx, tenant, actor, cfg, base.ID, &base.Revision); !ok || err != nil {
		t.Fatal("template copy", ok, err)
	}
	stale := "stale"
	if _, ok, err = s.TryCreateApplicationV3(ctx, tenant, actor, cfg, base.ID, &stale); !errors.Is(err, ErrRelationalDraftConflict) || !ok {
		t.Fatal("template revision guard", ok, err)
	}

	until := time.Now().Add(5 * time.Minute)
	check, ok, err := s.TryRecordReleaseCheckV3(ctx, tenant, saved.ID, actor, saved.Revision, "validation-entry", until)
	if err != nil || !ok || check.AppID != saved.ID {
		t.Fatal("check", ok, err)
	}
	if err = s.UpdateOperationalContext(ctx, func(st *State) error {
		a := st.Apps[saved.ID]
		a.Draft.Basic.Name = "blocked"
		st.Apps[saved.ID] = a
		return nil
	}); !errors.Is(err, ErrRelationalApplicationDomain) {
		t.Fatal("generic store still blocks app mutation", err)
	}

	release, ok, err := s.TryPublishReleaseV3(ctx, tenant, saved.ID, actor, client, request, check.ID, saved.Revision, nil, "入口发布")
	if err != nil || !ok || release.Note != "入口发布" {
		t.Fatal("publish", ok, err)
	}
	after := readV3(t, s)
	if after.Apps[saved.ID].CurrentReleaseID == nil || len(after.Checks) != 0 {
		t.Fatal("publish left check or missing pointer")
	}

	second := preflightDB(t, dsn)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _, e := s.TrySaveApplicationDraftV3(ctx, tenant, saved.ID, actor, "", "", after.Apps[saved.ID].Revision, cfg)
		errs <- e
	}()
	go func() {
		defer wg.Done()
		next := after.Apps[saved.ID].Draft
		next.Basic.Name = "并发入口"
		_, e := saveMySQLV3Draft(ctx, second, tenant, saved.ID, actor, "", "", after.Apps[saved.ID].Revision, next)
		errs <- e
	}()
	wg.Wait()
	close(errs)
	var sawConflict bool
	for e := range errs {
		if errors.Is(e, ErrRelationalDraftConflict) {
			sawConflict = true
		} else if e != nil {
			t.Fatal(e)
		}
	}
	if !sawConflict {
		t.Fatal("concurrent save did not CAS")
	}

	if _, err = db.Exec(`CREATE TRIGGER fail_entry_audit BEFORE INSERT ON cognara_fg_audit FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected entry failure'`); err != nil {
		t.Fatal(err)
	}
	if _, ok, err = s.TryCreateApplicationV3(ctx, tenant, actor, blankEntryConfig("应回滚"), "", nil); err == nil || !ok {
		t.Fatal("audit failure accepted", ok, err)
	}
}

func blankEntryConfig(name string) Config {
	return Config{
		Basic:                Basic{Name: name},
		ModelPrompt:          ModelPrompt{ModelID: "validation-only", Prompt: "仅使用已授权能力，明确来源与限制。"},
		CapabilityRefs:       CapabilityRefs{ServiceRefs: []ServiceRef{}},
		ConversationDefaults: ConversationDefaults{},
	}
}

func auditActionCount(t *testing.T, db *sql.DB, action string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_audit WHERE action=?`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func lastAuditForResource(t *testing.T, db *sql.DB, resource string) Event {
	t.Helper()
	var raw []byte
	if err := db.QueryRow(`SELECT payload FROM cognara_fg_audit WHERE resource_id=? ORDER BY sequence_no DESC LIMIT 1`, resource).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	return ev
}
