package applications

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestPrepareSchema8IntegrationDB(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	if err := PrepareSchema8IntegrationDB(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	st, err := store.readSQLContext(context.Background())
	if err != nil || len(st.Usage) < 2 || len(st.Exceptions) < 1 {
		t.Fatal("fixture incomplete", err, len(st.Usage), len(st.Exceptions))
	}
}