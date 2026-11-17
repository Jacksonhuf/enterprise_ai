package applications

import (
	"context"
	"errors"
	"testing"
)

func schema4GrantFixture(t *testing.T) string {
	t.Helper()
	dsn, _ := schema3GrantFixture(t)
	ctx := context.Background()
	plan, err := PlanMySQLGrants(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLGrants(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	return dsn
}

func TestMutateGrantsV4RejectsRootGrantChange(t *testing.T) {
	ctx := context.Background()
	dsn := schema4GrantFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ok, err := store.TryMutateGrantsV4(ctx, func(st *State) error {
		st.Grants = append(st.Grants, Grant{ID: "ghost", TenantID: "42", Active: true})
		return nil
	})
	if !ok || err == nil {
		t.Fatal("invalid grant insert accepted", ok, err)
	}
	if errors.Is(err, errRelationalGrantDomain) {
		t.Fatal("expected validation failure", err)
	}
}

func TestMutateGrantsV4RevokeIdempotent(t *testing.T) {
	ctx := context.Background()
	dsn := schema4GrantFixture(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var target string
	for _, g := range store.Read().Grants {
		if g.Active {
			target = g.ID
			break
		}
	}
	if target == "" {
		t.Fatal("no seed grant")
	}
	mutate := func(st *State) error {
		for i, g := range st.Grants {
			if g.ID == target {
				if !g.Active {
					return nil
				}
				st.Grants[i].Active = false
				st.Audit = append(st.Audit, Event{TenantID: g.TenantID, ID: ID("event"), OccurredAt: Now(), Action: "authorization.revoke", ResourceID: g.ID, Outcome: "success"})
				return nil
			}
		}
		return errGrantUnavailable
	}
	if _, err = store.TryMutateGrantsV4(ctx, mutate); err != nil {
		t.Fatal(err)
	}
	before := len(store.Read().Audit)
	if _, err = store.TryMutateGrantsV4(ctx, mutate); err != nil {
		t.Fatal(err)
	}
	if len(store.Read().Audit) != before {
		t.Fatal("idempotent revoke wrote audit")
	}
	var active bool
	if err = preflightDB(t, dsn).QueryRow(`SELECT active FROM cognara_fg_grants WHERE grant_id=?`, target).Scan(&active); err != nil || active {
		t.Fatal("grant still active", err)
	}
}