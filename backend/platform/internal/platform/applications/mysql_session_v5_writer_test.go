package applications

import (
	"context"
	"testing"
)

func schema5SessionFixture(t *testing.T) string {
	t.Helper()
	dsn, _ := schema4SessionRoot(t)
	plan, err := PlanMySQLSessions(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLSessions(context.Background(), dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	return dsn
}

func TestSchema5OperationalWriterAppendsMessages(t *testing.T) {
	ctx := context.Background()
	dsn := schema5SessionFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	before, err := store.readSQLContext(ctx)
	if err != nil || len(before.Sessions) != 1 {
		t.Fatal(err, len(before.Sessions))
	}
	var id string
	for k := range before.Sessions {
		id = k
	}
	msg := Message{ID: "msg-writer-2", SessionID: id, Role: "assistant", Content: "回复", CreatedAt: Now()}
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		s := st.Sessions[id]
		s.Messages = append(s.Messages, msg)
		s.UpdatedAt = Now()
		s.Revision = "rev-writer-2"
		st.Sessions[id] = s
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after, err := store.readSQLContext(ctx)
	if err != nil || len(after.Sessions[id].Messages) != 2 {
		t.Fatal(err, len(after.Sessions[id].Messages))
	}
	var count int
	if err = preflightDB(t, dsn).QueryRow(`SELECT COUNT(*) FROM cognara_fg_session_messages WHERE session_id=?`, id).Scan(&count); err != nil || count != 2 {
		t.Fatal("relational messages", count, err)
	}
}

func TestSchema5BackToBackSessionCreates(t *testing.T) {
	ctx := context.Background()
	dsn := schema5SessionFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for i := 0; i < 3; i++ {
		id := ID("session")
		ok, err := store.TryMutateSessionsV5(ctx, func(st *State) error {
			st.Sessions[id] = Session{
				ID: id, AppID: "channel-replenishment", TenantID: "42", OwnerID: "demo-user",
				Title: "extra", CreatedAt: Now(), UpdatedAt: Now(), Revision: ID("revision"),
			}
			return nil
		})
		if !ok || err != nil {
			t.Fatal(i, ok, err)
		}
	}
}

func TestSchema5DedicatedSessionWriterCreate(t *testing.T) {
	ctx := context.Background()
	dsn := schema5SessionFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	session := Session{
		ID: "session-writer-new", AppID: "channel-replenishment", TenantID: "42", OwnerID: "demo-user",
		Title: "新会话", CreatedAt: Now(), UpdatedAt: Now(), Revision: "rev-new",
		Messages: []Message{{ID: "msg-writer-new", SessionID: "session-writer-new", Role: "user", Content: "hi", CreatedAt: Now()}},
	}
	ok, err := store.TryMutateSessionsV5(ctx, func(st *State) error {
		st.Sessions[session.ID] = session
		return nil
	})
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	st, err := store.readSQLContext(ctx)
	if err != nil || st.Sessions[session.ID].Title != "新会话" {
		t.Fatal(err, st.Sessions[session.ID])
	}
}

func TestSchema5GrantWriterStillWorks(t *testing.T) {
	ctx := context.Background()
	dsn := schema5SessionFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ok, err := store.TryMutateGrantsV4(ctx, func(st *State) error {
		st.Grants = append(st.Grants, Grant{
			ID: "grant-schema5", TenantID: "42", SubjectType: "user", SubjectID: "demo-user",
			ResourceType: "app", ResourceID: "channel-replenishment", Action: "use", Scope: "tenant", Active: true,
		})
		return nil
	})
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	st, err := store.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range st.Grants {
		if g.ID == "grant-schema5" && g.Active {
			found = true
		}
	}
	if !found {
		t.Fatal("grant missing on schema5")
	}
}