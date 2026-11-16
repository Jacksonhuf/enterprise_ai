package applications

import (
	"context"
	"database/sql"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"strings"
	"testing"
)

func TestMySQLInvalidConfiguration(t *testing.T) {
	for _, dsn := range []string{"", "secret-password@invalid", "user:secret@tcp(localhost:3306)/test?multiStatements=true"} {
		if _, e := OpenMySQL(dsn); e == nil || strings.Contains(e.Error(), "secret") {
			t.Fatal("invalid configuration accepted or leaked")
		}
	}
}
func TestMySQLRollbackReopenAndUnknownSchema(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	before := len(s.Read().Sessions)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e = s.UpdateContext(ctx, func(st *State) error { st.Sessions["not-committed"] = Session{ID: "not-committed"}; return nil }); e == nil {
		t.Fatal("canceled update committed")
	}
	reopened, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if len(reopened.Read().Sessions) != before {
		t.Fatal("canceled update persisted")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec("UPDATE cognara_fg_state SET schema_version=99 WHERE id=1"); e != nil {
		t.Fatal(e)
	}
	if _, e = OpenMySQL(dsn); e == nil {
		t.Fatal("unknown schema accepted")
	}
	var version int
	db.QueryRow("SELECT schema_version FROM cognara_fg_state WHERE id=1").Scan(&version)
	if version != 99 {
		t.Fatal("unknown schema overwritten")
	}
}
