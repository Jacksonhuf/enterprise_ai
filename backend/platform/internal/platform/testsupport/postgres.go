// Package testsupport creates isolated schemas in an explicitly supplied test database.
package testsupport

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/url"
	"os"
	"testing"
	"time"
)

func PostgresDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("COGNARA_TEST_POSTGRES_DSN")
	if dsn == "" {
		if os.Getenv("COGNARA_REQUIRE_POSTGRES_TESTS") == "1" {
			t.Fatal("PostgreSQL integration tests required but test DSN missing")
		}
		t.Skip("set COGNARA_TEST_POSTGRES_DSN to run real PostgreSQL integration tests")
	}
	u, e := url.Parse(dsn)
	if e != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("test PostgreSQL DSN must be a postgres URL")
	}
	db, e := sql.Open("pgx", dsn)
	if e != nil {
		t.Fatal("test PostgreSQL driver unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		t.Fatal("test PostgreSQL not reachable")
	}
	suffix := make([]byte, 8)
	if _, e = rand.Read(suffix); e != nil {
		db.Close()
		t.Fatal(e)
	}
	schema := "cognara_test_" + hex.EncodeToString(suffix)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, e = db.ExecContext(ctx, "CREATE SCHEMA "+quoted); e != nil {
		db.Close()
		t.Fatal("cannot create isolated test schema")
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, e := db.ExecContext(cleanup, "DROP SCHEMA "+quoted+" CASCADE"); e != nil {
			t.Error("cannot clean isolated PostgreSQL test schema")
		}
		db.Close()
	})
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}
