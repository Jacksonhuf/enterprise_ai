package api_test

import (
	"context"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
)

func TestMySQLSchema4GrantHTTPMatchesAggregateSemantics(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	seed, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	seed.Close()
	if err = app.MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	appPlan, err := app.PlanMySQLApplications(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = app.ApplyMySQLApplications(ctx, dsn, appPlan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	grantPlan, err := app.PlanMySQLGrants(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = app.ApplyMySQLGrants(ctx, dsn, grantPlan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	f := setup(t)
	f.store = store
	f.server.Store = store
	TestGrantRejectsForeignSubjectsAndResources(t)
	TestExistingActiveGrantReusedWithoutAudit(t)
	TestPublishedCopyVisibilityAndSessionAccessFollowUserGrantLifecycle(t)
}