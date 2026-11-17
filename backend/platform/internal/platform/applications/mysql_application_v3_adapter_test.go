package applications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestMySQLV3StoreWritersRouteRootDomains(t *testing.T) {
	ctx := context.Background()
	s, db, dsn := stageMySQLV3Writers(t)
	a := readV3(t, s).Apps["channel-replenishment"]
	tenant := a.TenantID
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',CAST(? AS JSON),'$.future_root',CAST(? AS JSON)) WHERE id=1`, `[{"id":"target","tenant_id":"`+tenant+`","status":"OPEN","revision":"r1","future":{"n":9007199254740993}}]`, `{"n":9007199254740993}`); err != nil {
		t.Fatal(err)
	}
	base := readV3(t, s)
	apps, _ := json.Marshal(base.Apps)
	event := func(id, action string) Event {
		return Event{ID: id, TenantID: tenant, ActorID: "demo-admin", Action: action, Outcome: "success", OccurredAt: Now()}
	}

	if err := s.UpdateOperationalContext(ctx, func(st *State) error {
		st.Sessions["session-adapter"] = Session{ID: "session-adapter", AppID: a.ID, TenantID: tenant, OwnerID: "demo-user", Title: "适配会话", Messages: []Message{}}
		st.Audit = append(st.Audit, event("event-store", "session.create"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateContext(ctx, func(st *State) error {
		if len(st.Audit) != len(base.Audit)+1 {
			return errors.New("full callback missing audit history")
		}
		st.Audit = append(st.Audit, event("event-full", "grant.review"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateOperationalContext(ctx, func(st *State) error {
		x := st.Apps[a.ID]
		x.Draft.Basic.Name = "bypass"
		st.Apps[a.ID] = x
		return nil
	}); !errors.Is(err, errRelationalApplicationDomain) {
		t.Fatal("store changed applications", err)
	}

	handled, err := s.UpdateMySQLSettingsContext(ctx, "demo-user:announcement", "", func(r SettingsRecord) (*SettingsPatch, error) {
		if r.Canonical {
			return nil, errors.New("unexpected existing setting")
		}
		return &SettingsPatch{Value: json.RawMessage(`{"text":"公告"}`), Revision: "s1", Audit: []Event{event("event-settings", "settings.update")}}, nil
	})
	if err != nil || !handled {
		t.Fatal("settings writer", handled, err)
	}
	record, handled, err := s.ReadMySQLSettingsContext(ctx, "demo-user:announcement", "")
	if err != nil || !handled || record.Revision != "s1" || !equalApplicationJSON(record.Value, []byte(`{"text":"公告"}`)) {
		t.Fatal("settings not readable", err, string(record.Value))
	}

	raw, handled, err := s.UpdateMySQLExceptionContext(ctx, tenant, "target", func(item map[string]any) (*ExceptionPatch, error) {
		if item["revision"] != "r1" {
			return nil, errors.New("wrong exception")
		}
		return &ExceptionPatch{Status: "ACKNOWLEDGED", Revision: "r2", UpdatedAt: Now(), UpdatedBy: "demo-admin", Audit: []Event{event("event-exception", "exception.update")}}, nil
	})
	if err != nil || !handled || !strings.Contains(string(raw), "9007199254740993") || !strings.Contains(string(raw), "ACKNOWLEDGED") {
		t.Fatal("exception writer", err, string(raw))
	}
	if _, _, err = s.UpdateMySQLExceptionContext(ctx, "foreign", "target", func(map[string]any) (*ExceptionPatch, error) {
		return nil, errors.New("foreign tenant reached callback")
	}); !errors.Is(err, ErrExceptionUnavailable) {
		t.Fatal("foreign exception selected", err)
	}

	if handled, err = s.AppendMySQLObservabilityContext(ctx, []Event{event("event-failure", "chat.failure")}, map[string]any{"id": "failure-1", "tenant_id": tenant, "status": "OPEN", "revision": "f1"}); err != nil || !handled {
		t.Fatal("observability writer", handled, err)
	}

	after := readV3(t, s)
	got, _ := json.Marshal(after.Apps)
	if !equalApplicationJSON(apps, got) {
		t.Fatal("root writers changed applications")
	}
	if after.Sessions["session-adapter"].Title != "适配会话" || len(after.Exceptions) != 2 || after.Exceptions[0]["status"] != "ACKNOWLEDGED" || after.Exceptions[1]["id"] != "failure-1" {
		t.Fatal("root domains not committed")
	}
	want := []string{"event-store", "event-full", "event-settings", "event-exception", "event-failure"}
	if len(after.Audit) != len(base.Audit)+len(want) {
		t.Fatal("audit count", len(after.Audit))
	}
	for i, id := range want {
		if after.Audit[len(base.Audit)+i].ID != id {
			t.Fatal("audit order", i, after.Audit[len(base.Audit)+i].ID)
		}
	}
	body := rootBody(t, db)
	if _, ok := body["apps"]; ok {
		t.Fatal("root regained applications")
	}
	if !equalApplicationJSON(body["future_root"], []byte(`{"n":9007199254740993}`)) || !strings.Contains(string(body["exceptions"]), "9007199254740993") {
		t.Fatal("root extension or exception precision lost")
	}

	second := preflightDB(t, dsn)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	run := func(f func() error) {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- f() }()
	}
	run(func() error {
		return s.UpdateOperationalContext(ctx, func(st *State) error {
			st.Grants = append(st.Grants, Grant{ID: "grant-adapter", TenantID: tenant, SubjectType: "USER", SubjectID: "demo-user", ResourceType: "APPLICATION", ResourceID: a.ID, Action: "application.use", Scope: "east", Active: true})
			return nil
		})
	})
	run(func() error {
		_, e := s.UpdateMySQLSettingsContext(ctx, "demo-user:theme", "", func(SettingsRecord) (*SettingsPatch, error) {
			return &SettingsPatch{Value: json.RawMessage(`{"theme":"blue"}`), Revision: "t1"}, nil
		})
		return e
	})
	run(func() error {
		_, e := s.AppendMySQLObservabilityContext(ctx, []Event{event("event-concurrent", "chat.failure")}, map[string]any{"id": "failure-2", "tenant_id": tenant, "status": "OPEN", "revision": "f1"})
		return e
	})
	run(func() error {
		next := a.Draft
		next.Basic.Name = "适配并发草稿"
		_, e := saveMySQLV3Draft(ctx, second, tenant, a.ID, "demo-engineer", "", "", a.Revision, next)
		return e
	})
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	merged := readV3(t, s)
	hasGrant := false
	for _, g := range merged.Grants {
		hasGrant = hasGrant || g.ID == "grant-adapter"
	}
	if !hasGrant || string(merged.Settings["demo-user:theme"]) == "" || len(merged.Exceptions) != 3 || merged.Apps[a.ID].Draft.Basic.Name != "适配并发草稿" || len(merged.Audit) != len(after.Audit)+2 {
		t.Fatal("concurrent writers lost an update")
	}

	if _, err = db.Exec(`CREATE TRIGGER fail_adapter_audit BEFORE INSERT ON cognara_fg_audit FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected adapter failure'`); err != nil {
		t.Fatal(err)
	}
	failures := map[string]func() error{
		"store": func() error {
			return s.UpdateOperationalContext(ctx, func(st *State) error {
				delete(st.Sessions, "session-adapter")
				st.Audit = append(st.Audit, event("event-x1", "session.delete"))
				return nil
			})
		},
		"settings": func() error {
			_, e := s.UpdateMySQLSettingsContext(ctx, "demo-user:announcement", "", func(SettingsRecord) (*SettingsPatch, error) {
				return &SettingsPatch{Value: json.RawMessage(`{"text":"x"}`), Revision: "s2", Audit: []Event{event("event-x2", "settings.update")}}, nil
			})
			return e
		},
		"exception": func() error {
			_, _, e := s.UpdateMySQLExceptionContext(ctx, tenant, "target", func(map[string]any) (*ExceptionPatch, error) {
				return &ExceptionPatch{Status: "RESOLVED", Revision: "r3", UpdatedAt: Now(), UpdatedBy: "demo-admin", Audit: []Event{event("event-x3", "exception.update")}}, nil
			})
			return e
		},
		"observability": func() error {
			_, e := s.AppendMySQLObservabilityContext(ctx, []Event{event("event-x4", "chat.failure")}, map[string]any{"id": "failure-x", "tenant_id": tenant, "status": "OPEN", "revision": "f1"})
			return e
		},
	}
	m, _ := json.Marshal(merged)
	for name, f := range failures {
		if f() == nil {
			t.Fatal(name, "accepted audit failure")
		}
		n, _ := json.Marshal(readV3(t, s))
		if !equalApplicationJSON(m, n) {
			t.Fatal(name, "partial write persisted")
		}
	}
	if opened, e := OpenMySQL(dsn); e == nil {
		opened.Close()
		t.Fatal("runtime opened schema3")
	}
}

func TestMySQLV3RootMergeKeepsStoredTextAndUnknownFields(t *testing.T) {
	ctx := context.Background()
	s, db, _ := stageMySQLV3Writers(t)
	a := readV3(t, s).Apps["channel-replenishment"]
	session := `{"session_id":"kept","app_id":"` + a.ID + `","tenant_id":"` + a.TenantID + `","owner_id":"demo-user","title":"old","messages":[],"x_ext":{"n":9007199254740993}}`
	exceptions := `[{"id":"first","tenant_id":"` + a.TenantID + `","only_first":true},{"id":"second","tenant_id":"` + a.TenantID + `","n":9007199254740993}]`
	usage := `[{"event_id":"u1","tenant_id":"` + a.TenantID + `","count":9007199254740993}]`
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.sessions.kept',CAST(? AS JSON),'$.exceptions',CAST(? AS JSON),'$.usage',CAST(? AS JSON)) WHERE id=1`, session, exceptions, usage); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateOperationalContext(ctx, func(st *State) error {
		x := st.Sessions["kept"]
		x.Title = "new"
		st.Sessions["kept"] = x
		st.Exceptions = st.Exceptions[1:]
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	body := rootBody(t, db)
	var sessions map[string]map[string]json.RawMessage
	var gotExceptions []map[string]json.RawMessage
	if json.Unmarshal(body["sessions"], &sessions) != nil || json.Unmarshal(body["exceptions"], &gotExceptions) != nil {
		t.Fatal("invalid root")
	}
	if string(sessions["kept"]["title"]) != `"new"` || !equalApplicationJSON(sessions["kept"]["x_ext"], []byte(`{"n":9007199254740993}`)) {
		t.Fatal("session edit lost unknown field", string(body["sessions"]))
	}
	if len(gotExceptions) != 1 || string(gotExceptions[0]["id"]) != `"second"` || string(gotExceptions[0]["n"]) != "9007199254740993" || gotExceptions[0]["only_first"] != nil {
		t.Fatal("array removal grafted or rounded", string(body["exceptions"]))
	}
	if !strings.Contains(string(body["usage"]), "9007199254740993") {
		t.Fatal("untouched domain rounded", string(body["usage"]))
	}
}
