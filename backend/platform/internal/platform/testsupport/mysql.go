package testsupport

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"github.com/go-sql-driver/mysql"
	"os"
	"testing"
)

func MySQLDSN(t *testing.T) string {
	t.Helper()
	raw := os.Getenv("COGNARA_TEST_MYSQL_DSN")
	if raw == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("MySQL integration DSN required")
		}
		t.Skip("set COGNARA_TEST_MYSQL_DSN for real MySQL integration")
	}
	cfg, e := mysql.ParseDSN(raw)
	if e != nil {
		t.Fatal("invalid MySQL test DSN")
	}
	cfg.DBName = ""
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		t.Fatal("MySQL test driver unavailable")
	}
	suffix := make([]byte, 8)
	if _, e = rand.Read(suffix); e != nil {
		t.Fatal(e)
	}
	name := "cognara_test_" + hex.EncodeToString(suffix)
	if _, e = db.Exec("CREATE DATABASE `" + name + "`"); e != nil {
		db.Close()
		t.Fatal("cannot create isolated MySQL test database")
	}
	t.Cleanup(func() {
		if _, e := db.Exec("DROP DATABASE `" + name + "`"); e != nil {
			t.Error("cannot remove MySQL test database")
		}
		db.Close()
	})
	cfg.DBName = name
	return cfg.FormatDSN()
}
