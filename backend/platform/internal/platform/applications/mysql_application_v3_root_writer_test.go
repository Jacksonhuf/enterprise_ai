package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func rootBody(t *testing.T, db *sql.DB) map[string]json.RawMessage {
	t.Helper()
	var raw []byte
	if err := db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func TestMySQLV3RootWriterOwnsAggregateDomainsOnly(t *testing.T) {
	ctx := context.Background()
	s, db, dsn := stageMySQLV3Writers(t)
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_root',CAST('{"n":9007199254740993}' AS JSON)) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&raw)
	var extension map[string]json.RawMessage
	json.Unmarshal(raw, &extension)
	before := readV3(t, s)
	a := before.Apps["channel-replenishment"]

	session := Session{ID: "session-root", AppID: a.ID, TenantID: a.TenantID, OwnerID: "demo-user", Title: "根域会话", Messages: []Message{}}
	if err := updateMySQLV3Root(ctx, db, true, func(st *State) error {
		if len(st.Audit) != 0 {
			return errors.New("operational callback received audit history")
		}
		st.Sessions[session.ID] = session
		st.Settings["demo-user:profile"] = json.RawMessage(`{"display_name":"根域"}`)
		st.Audit = append(st.Audit, Event{ID: "event-root", TenantID: a.TenantID, ActorID: "demo-user", Action: "session.create", ResourceID: session.ID, Outcome: "success", OccurredAt: Now()})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after := readV3(t, s)
	if after.Sessions[session.ID].Title != "根域会话" || string(after.Settings["demo-user:profile"]) != `{"display_name":"根域"}` || len(after.Audit) != len(before.Audit)+1 || after.Audit[len(after.Audit)-1].ID != "event-root" {
		t.Fatal("root domains or audit not committed together")
	}
	body := rootBody(t, db)
	if _, ok := body["apps"]; ok {
		t.Fatal("root regained application authority")
	}
	if _, ok := body["checks"]; ok {
		t.Fatal("root regained check authority")
	}
	if !equalApplicationJSON(body["future_root"], extension["future_root"]) {
		t.Fatal("root extension lost")
	}
	x, _ := json.Marshal(before.Apps)
	y, _ := json.Marshal(after.Apps)
	if !equalApplicationJSON(x, y) {
		t.Fatal("root write changed applications")
	}

	if err := updateMySQLV3Root(ctx, db, false, func(st *State) error {
		st.Audit[0].Outcome = "rewritten"
		return nil
	}); err == nil {
		t.Fatal("audit history rewritten")
	}
	if err := updateMySQLV3Root(ctx, db, false, func(st *State) error {
		if len(st.Audit) != len(after.Audit) {
			return errors.New("full callback missing audit history")
		}
		st.Audit = append(st.Audit, Event{ID: "event-full", TenantID: a.TenantID, ActorID: "demo-admin", Action: "grant.review", Outcome: "success", OccurredAt: Now()})
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	current := readV3(t, s)
	for name, mutate := range map[string]func(*State) error{
		"app": func(st *State) error {
			x := st.Apps[a.ID]
			x.Draft.Basic.Name = "bypass"
			st.Apps[a.ID] = x
			return nil
		},
		"new-app":  func(st *State) error { st.Apps["injected"] = App{ID: "injected", TenantID: a.TenantID}; return nil },
		"check":    func(st *State) error { st.Checks["injected"] = Check{ID: "injected", AppID: a.ID}; return nil },
		"callback": func(st *State) error { st.Grants = nil; return errors.New("callback failed") },
	} {
		if err := updateMySQLV3Root(ctx, db, true, mutate); err == nil {
			t.Fatal(name, "accepted")
		} else if name != "callback" && !errors.Is(err, errRelationalApplicationDomain) {
			t.Fatal(name, err)
		}
	}
	p, _ := json.Marshal(current)
	q, _ := json.Marshal(readV3(t, s))
	if !equalApplicationJSON(p, q) {
		t.Fatal("rejected root write persisted")
	}

	second := preflightDB(t, dsn)
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i, conn := range []*sql.DB{db, second} {
		wg.Add(1)
		go func(i int, conn *sql.DB) {
			defer wg.Done()
			errs <- updateMySQLV3Root(ctx, conn, true, func(st *State) error {
				st.Grants = append(st.Grants, Grant{ID: fmt.Sprintf("grant-concurrent-%d", i), TenantID: a.TenantID, SubjectType: "USER", SubjectID: "demo-user", ResourceType: "APPLICATION", ResourceID: a.ID, Action: "application.use", Scope: "east", Active: true})
				return nil
			})
		}(i, conn)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		next := a.Draft
		next.Basic.Name = "并发草稿"
		_, e := saveMySQLV3Draft(ctx, second, a.TenantID, a.ID, "demo-engineer", "", "", a.Revision, next)
		errs <- e
	}()
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	merged := readV3(t, s)
	found := 0
	for _, g := range merged.Grants {
		if g.ID == "grant-concurrent-0" || g.ID == "grant-concurrent-1" {
			found++
		}
	}
	if found != 2 || merged.Apps[a.ID].Draft.Basic.Name != "并发草稿" {
		t.Fatal("concurrent root/dedicated writes lost an update", found)
	}

	if _, err := db.Exec(`CREATE TRIGGER fail_root_audit BEFORE INSERT ON cognara_fg_audit FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected root failure'`); err != nil {
		t.Fatal(err)
	}
	if err := updateMySQLV3Root(ctx, db, true, func(st *State) error {
		delete(st.Sessions, session.ID)
		st.Audit = append(st.Audit, Event{ID: "event-fail", TenantID: a.TenantID, Action: "session.delete", OccurredAt: Now()})
		return nil
	}); err == nil {
		t.Fatal("audit failure accepted")
	}
	m, _ := json.Marshal(merged)
	n, _ := json.Marshal(readV3(t, s))
	if !equalApplicationJSON(m, n) {
		t.Fatal("partial root write persisted")
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := preflightDB(t, dsn)
	tx, err := reopened.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	state, _, err := readMySQLApplicationV3(ctx, tx, true)
	if err != nil || state.Sessions[session.ID].Title != "根域会话" || len(state.Grants) != len(merged.Grants) {
		t.Fatal("reopen lost root writes", err)
	}
}
