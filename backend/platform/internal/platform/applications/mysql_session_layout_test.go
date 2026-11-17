package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func schema4SessionRoot(t *testing.T) (dsn string, root []byte) {
	t.Helper()
	dsn = schema4GrantFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.UpdateOperationalContext(context.Background(), func(st *State) error {
		st.Sessions["session-plan-1"] = Session{
			ID: "session-plan-1", AppID: "channel-replenishment", TenantID: "42", OwnerID: "demo-user",
			Title: "计划验证", CreatedAt: Now(), UpdatedAt: Now(), Revision: "rev-plan-1",
			Messages: []Message{{ID: "msg-plan-1", SessionID: "session-plan-1", Role: "user", Content: "你好", CreatedAt: Now()}},
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err = preflightDB(t, dsn).QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		t.Fatal(err)
	}
	return dsn, root
}

func TestSessionLayoutRoundTrip(t *testing.T) {
	_, root := schema4SessionRoot(t)
	if err := verifySessionLayout(root); err != nil {
		t.Fatal(err)
	}
	layout, err := splitSessionLayout(root)
	if err != nil || len(layout.Sessions) != 1 || len(layout.Sessions[0].Messages) != 1 {
		t.Fatal(err, len(layout.Sessions))
	}
	rebuilt, err := joinSessionLayout(layout)
	if err != nil || !equalApplicationJSON(root, rebuilt) {
		t.Fatal("round trip", err)
	}
}

func TestSessionLayoutRejectsGrantAuthorityInRoot(t *testing.T) {
	_, root := schema4SessionRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	object["grants"] = json.RawMessage(`[]`)
	bad, _ := json.Marshal(object)
	if _, err := splitSessionLayout(bad); err == nil {
		t.Fatal("grants in root accepted")
	}
}