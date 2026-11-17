package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func schema7UsageRoot(t *testing.T) (dsn string, root []byte) {
	t.Helper()
	ctx := context.Background()
	dsn = schema7ExceptionSnapshotFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Usage = append(st.Usage,
			map[string]any{"event_id": "usage-plan-1", "tenant_id": "42", "unit": "CALL", "capability": "metric", "count": 1, "occurred_at": Now()},
			map[string]any{"event_id": "usage-plan-2", "tenant_id": "42", "unit": "TOKEN", "capability": "test", "count": 2, "occurred_at": Now()},
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

func TestUsageLayoutRoundTrip(t *testing.T) {
	_, root := schema7UsageRoot(t)
	if err := verifyUsageLayout(root); err != nil {
		t.Fatal(err)
	}
	layout, err := splitUsageLayout(root)
	if err != nil || len(layout.Usage) != 2 {
		t.Fatal(err, len(layout.Usage))
	}
	rebuilt, err := joinUsageLayout(layout)
	if err != nil || !equalApplicationJSON(root, rebuilt) {
		t.Fatal("round trip", err)
	}
}

func TestUsageLayoutRejectsRelationalAuthorityInRoot(t *testing.T) {
	_, root := schema7UsageRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	object["exceptions"] = json.RawMessage(`[]`)
	bad, _ := json.Marshal(object)
	if _, err := splitUsageLayout(bad); err == nil {
		t.Fatal("exceptions in root accepted")
	}
}