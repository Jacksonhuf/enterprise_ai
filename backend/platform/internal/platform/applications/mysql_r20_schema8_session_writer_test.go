package applications

import (
	"context"
	"fmt"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestR20Schema8BackToBackSessionWriterCreates(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	if err := PrepareSchema8IntegrationDB(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	before := len(store.readSQLContextMust(t, ctx).Sessions)
	added := 0
	for i := 0; i < 4; i++ {
		id := ID("session")
		ok, err := store.TryMutateSessionsV5(ctx, func(st *State) error {
			st.Sessions[id] = Session{
				ID: id, AppID: "channel-replenishment", TenantID: "42", OwnerID: "demo-user",
				Title: fmt.Sprintf("writer-%d", i), CreatedAt: Now(), UpdatedAt: Now(), Revision: ID("revision"),
			}
			return nil
		})
		if !ok || err != nil {
			t.Fatalf("create %d ok=%v err=%v", i, ok, err)
		}
		added++
	}
	after := store.readSQLContextMust(t, ctx)
	if len(after.Sessions) != before+added {
		t.Fatal("sessions", before, added, len(after.Sessions))
	}
}