package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func schema5SettingsRoot(t *testing.T) (dsn string, root []byte, tenant string) {
	t.Helper()
	dsn = schema5SessionFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.UpdateOperationalContext(context.Background(), func(st *State) error {
		st.Settings["demo-user:profile"] = json.RawMessage(`{"display_name":"计划验证"}`)
		st.SettingRevisions["demo-user:profile"] = "rev-settings-1"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	st, err := store.readSQLContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tenant, err = dominantTenantID(st)
	if err != nil {
		t.Fatal(err)
	}
	if err = preflightDB(t, dsn).QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		t.Fatal(err)
	}
	return dsn, root, tenant
}

func TestSettingsLayoutRoundTrip(t *testing.T) {
	_, root, tenant := schema5SettingsRoot(t)
	if err := verifySettingsLayout(root, tenant); err != nil {
		t.Fatal(err)
	}
	layout, err := splitSettingsLayout(root)
	if err != nil || len(layout.Settings) != 1 {
		t.Fatal(err, len(layout.Settings))
	}
	layout.InferredTenant = tenant
	layout.Settings[0].TenantID = tenant
	rebuilt, err := joinSettingsLayout(layout)
	if err != nil || !equalApplicationJSON(root, rebuilt) {
		t.Fatal("round trip", err)
	}
}

func TestSettingsLayoutRejectsSessionAuthorityInRoot(t *testing.T) {
	_, root, tenant := schema5SettingsRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	object["sessions"] = json.RawMessage(`{}`)
	bad, _ := json.Marshal(object)
	if _, err := splitSettingsLayout(bad); err == nil {
		t.Fatal("sessions in root accepted")
	}
	if err := verifySettingsLayout(root, tenant); err != nil {
		t.Fatal(err)
	}
}