package applications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func mustApplySchema3FromSchema2(t *testing.T, dsn string) ApplicationMigrationPlan {
	t.Helper()
	ctx := context.Background()
	if err := MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMySQLApplications(ctx, dsn)
	if err != nil || !plan.RoundTripVerified {
		t.Fatal("plan", plan, err)
	}
	applied, err := ApplyMySQLApplications(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" {
		t.Fatal("apply", applied, err)
	}
	return applied
}

func TestMySQLApplicationApplyRestartPreservesState(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Sessions["apply-restart"] = Session{
			ID: "apply-restart", AppID: "channel-replenishment", TenantID: "42",
			OwnerID: "demo-user", Title: "before-apply", Messages: nil,
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, err := store.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	mustApplySchema3FromSchema2(t, dsn)
	restarted, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("restart open", err)
	}
	defer restarted.Close()
	after, err := restarted.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	slice := func(st State) []byte {
		raw, _ := json.Marshal(map[string]any{"apps": st.Apps, "sessions": st.Sessions, "checks": st.Checks})
		return raw
	}
	if !equalApplicationJSON(slice(before), slice(after)) {
		t.Fatal("apply changed application/session state")
	}
	var version int
	if err = preflightDB(t, dsn).QueryRow(`SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil || version != 3 {
		t.Fatal("schema version", version, err)
	}
}

func TestMySQLApplicationApplyDualStoreDraftCAS(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	first, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	mustApplySchema3FromSchema2(t, dsn)
	a, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	base := a.readSQLContextMust(t, ctx).Apps["channel-replenishment"]
	tenant, appID, revision := base.TenantID, base.ID, base.Revision
	cfgA, cfgB := base.Draft, base.Draft
	cfgA.Basic.Name = "并发草稿A"
	cfgB.Basic.Name = "并发草稿B"
	var wg sync.WaitGroup
	type outcome struct {
		ok  bool
		err error
	}
	out := make(chan outcome, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, ok, e := a.TrySaveApplicationDraftV3(ctx, tenant, appID, "demo-engineer", "", "", revision, cfgA)
		out <- outcome{ok, e}
	}()
	go func() {
		defer wg.Done()
		_, ok, e := b.TrySaveApplicationDraftV3(ctx, tenant, appID, "demo-engineer", "", "", revision, cfgB)
		out <- outcome{ok, e}
	}()
	wg.Wait()
	close(out)
	var wins int
	var sawConflict bool
	for o := range out {
		if errors.Is(o.err, ErrRelationalDraftConflict) {
			sawConflict = true
			continue
		}
		if o.err != nil {
			t.Fatal(o.err)
		}
		if o.ok {
			wins++
		}
	}
	if wins != 1 || !sawConflict {
		t.Fatal("dual connection CAS", wins, sawConflict)
	}
	st := a.readSQLContextMust(t, ctx)
	if st.Apps[base.ID].Draft.Basic.Name != "并发草稿A" && st.Apps[base.ID].Draft.Basic.Name != "并发草稿B" {
		t.Fatal("draft name lost")
	}
}

func (s *Store) readSQLContextMust(t *testing.T, ctx context.Context) State {
	t.Helper()
	st, err := s.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestCompiledMySQLSchema3ApplyPreservesRuntimeAPI(t *testing.T) {
	runtime := os.Getenv("COGNARA_TEST_RUNTIME_BINARY")
	if runtime == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("compiled runtime binary required")
		}
		t.Skip("compiled runtime not configured")
	}
	dsn := testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	ctx := context.Background()
	if err := MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	base, stop := recoveryHost(t, runtime, dsn)
	appPath := "/api/v1/cognitive/apps/channel-replenishment"
	beforeApp := recoveryObject(t, recoveryRequest(t, base, "demo-engineer", "GET", appPath, nil, 200))
	releasesPath := "/api/v1/cognitive/apps/channel-replenishment/releases"
	beforeReleases := recoveryRequest(t, base, "demo-engineer", "GET", releasesPath, nil, 200)
	stop()
	plan, err := PlanMySQLApplications(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLApplications(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal("apply", err)
	}
	restarted, stopRestart := recoveryHost(t, runtime, dsn)
	defer stopRestart()
	afterApp := recoveryObject(t, recoveryRequest(t, restarted, "demo-engineer", "GET", appPath, nil, 200))
	beforeAppJSON, _ := json.Marshal(beforeApp)
	afterAppJSON, _ := json.Marshal(afterApp)
	if !bytes.Equal(beforeAppJSON, afterAppJSON) {
		t.Fatal("application API drifted after apply restart")
	}
	afterReleases := recoveryRequest(t, restarted, "demo-engineer", "GET", releasesPath, nil, 200)
	if !bytes.Equal(beforeReleases, afterReleases) {
		t.Fatal("release list API drifted after apply restart")
	}
	opened, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("open after runtime", err)
	}
	opened.Close()
}