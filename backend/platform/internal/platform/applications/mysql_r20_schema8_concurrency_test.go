package applications

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func openSchema8DualStore(t *testing.T) (context.Context, string, *Store, *Store) {
	t.Helper()
	ctx := context.Background()
	dsn := schema8UsageSnapshotFixture(t)
	a, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("store a", err)
	}
	b, err := OpenMySQL(dsn)
	if err != nil {
		a.Close()
		t.Fatal("store b", err)
	}
	t.Cleanup(func() { a.Close(); b.Close() })
	return ctx, dsn, a, b
}

func TestR20Schema8DualStoreDraftCAS(t *testing.T) {
	ctx, _, a, b := openSchema8DualStore(t)
	base := a.readSQLContextMust(t, ctx).Apps["channel-replenishment"]
	tenant, appID, revision := base.TenantID, base.ID, base.Revision
	cfgA, cfgB := base.Draft, base.Draft
	cfgA.Basic.Name = "schema8-concurrent-A"
	cfgB.Basic.Name = "schema8-concurrent-B"
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
		t.Fatal("dual connection draft CAS", wins, sawConflict)
	}
	st := a.readSQLContextMust(t, ctx)
	name := st.Apps[base.ID].Draft.Basic.Name
	if name != "schema8-concurrent-A" && name != "schema8-concurrent-B" {
		t.Fatal("draft lost", name)
	}
}

func TestR20Schema8DualStoreConcurrentUsageAppends(t *testing.T) {
	ctx, _, a, b := openSchema8DualStore(t)
	before := len(a.readSQLContextMust(t, ctx).Usage)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs <- a.UpdateOperationalContext(ctx, func(st *State) error {
			st.Usage = append(st.Usage, map[string]any{
				"event_id": "r20-usage-a", "tenant_id": "42", "unit": "CALL", "capability": "metric", "count": 1, "occurred_at": Now(),
			})
			return nil
		})
	}()
	go func() {
		defer wg.Done()
		errs <- b.UpdateOperationalContext(ctx, func(st *State) error {
			st.Usage = append(st.Usage, map[string]any{
				"event_id": "r20-usage-b", "tenant_id": "42", "unit": "TOKEN", "capability": "metric", "count": 1, "occurred_at": Now(),
			})
			return nil
		})
	}()
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	after := len(a.readSQLContextMust(t, ctx).Usage)
	if after != before+2 {
		t.Fatal("usage count", before, after)
	}
}

func TestR20Schema8DualStoreExceptionStatusSerialize(t *testing.T) {
	ctx, _, a, b := openSchema8DualStore(t)
	st := a.readSQLContextMust(t, ctx)
	var target string
	for _, item := range st.Exceptions {
		if item["tenant_id"] == "42" {
			target, _ = item["id"].(string)
			break
		}
	}
	if target == "" {
		t.Fatal("no exception fixture")
	}
	patch := func(status string) func(map[string]any) (*ExceptionPatch, error) {
		return func(item map[string]any) (*ExceptionPatch, error) {
			rev, _ := item["revision"].(string)
			return &ExceptionPatch{
				Status: status, Revision: rev + "-r20", UpdatedAt: Now(), UpdatedBy: "demo-admin",
				Audit: []Event{{TenantID: "42", ID: fmt.Sprintf("audit-r20-%s", status), OccurredAt: Now()}},
			}, nil
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, handled, e := a.UpdateMySQLExceptionContext(ctx, "42", target, patch("ACKNOWLEDGED"))
		if e != nil || !handled {
			errs <- fmt.Errorf("a: handled=%v err=%w", handled, e)
			return
		}
		errs <- nil
	}()
	go func() {
		defer wg.Done()
		_, handled, e := b.UpdateMySQLExceptionContext(ctx, "42", target, patch("RESOLVED"))
		if e != nil || !handled {
			errs <- fmt.Errorf("b: handled=%v err=%w", handled, e)
			return
		}
		errs <- nil
	}()
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	final := a.readSQLContextMust(t, ctx)
	var status string
	for _, item := range final.Exceptions {
		if id, _ := item["id"].(string); id == target {
			status, _ = item["status"].(string)
			break
		}
	}
	if status != "ACKNOWLEDGED" && status != "RESOLVED" {
		t.Fatal("invalid final status", status)
	}
	if len(final.Exceptions) != len(st.Exceptions) {
		t.Fatal("exception rows duplicated")
	}
}