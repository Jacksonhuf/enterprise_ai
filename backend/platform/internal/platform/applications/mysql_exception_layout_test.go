package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func schema6ExceptionRoot(t *testing.T) (dsn string, root []byte) {
	t.Helper()
	ctx := context.Background()
	dsn = schema6SettingsSnapshotFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Exceptions = append(st.Exceptions,
			map[string]any{"id": "exc-plan-1", "tenant_id": "42", "status": "OPEN", "revision": "r1", "request_id": "req-1", "occurred_at": Now()},
			map[string]any{"id": "exc-plan-2", "tenant_id": "42", "status": "ACKNOWLEDGED", "revision": "r2", "request_id": "req-2", "occurred_at": Now()},
		)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err = preflightDB(t, dsn).QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		t.Fatal(err)
	}
	return dsn, root
}

func TestExceptionLayoutRoundTrip(t *testing.T) {
	_, root := schema6ExceptionRoot(t)
	if err := verifyExceptionLayout(root); err != nil {
		t.Fatal(err)
	}
	layout, err := splitExceptionLayout(root)
	if err != nil || len(layout.Exceptions) != 2 {
		t.Fatal(err, len(layout.Exceptions))
	}
	rebuilt, err := joinExceptionLayout(layout)
	if err != nil || !equalApplicationJSON(root, rebuilt) {
		t.Fatal("round trip", err)
	}
}

func TestExceptionLayoutRejectsRelationalAuthorityInRoot(t *testing.T) {
	_, root := schema6ExceptionRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	object["settings"] = json.RawMessage(`{}`)
	bad, _ := json.Marshal(object)
	if _, err := splitExceptionLayout(bad); err == nil {
		t.Fatal("settings in root accepted")
	}
}