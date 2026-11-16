package applications

import (
	"database/sql"
	"errors"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"sync"
	"testing"
)

func TestMySQLRejectsIncompatibleTablesWithoutSeeding(t *testing.T) {
	for _, ddl := range []string{
		`CREATE TABLE cognara_fg_state(id INT PRIMARY KEY,schema_version INT NOT NULL,body JSON NOT NULL) ENGINE=MyISAM`,
		`CREATE TABLE cognara_fg_state(id INT PRIMARY KEY,schema_version INT NOT NULL,body TEXT NOT NULL) ENGINE=InnoDB`,
		`CREATE TABLE cognara_fg_state(id INT,schema_version INT NOT NULL,body JSON NOT NULL) ENGINE=InnoDB`,
		`CREATE TABLE cognara_fg_state(id INT PRIMARY KEY,schema_version INT NOT NULL,body JSON NOT NULL,required_extra INT NOT NULL) ENGINE=InnoDB`,
	} {
		t.Run(ddl, func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			db, e := sql.Open("mysql", dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			if _, e = db.Exec(ddl); e != nil {
				t.Fatal(e)
			}
			if s, e := OpenMySQL(dsn); e == nil {
				s.Close()
				t.Fatal("incompatible table accepted")
			}
			var count int
			if e = db.QueryRow("SELECT COUNT(*) FROM cognara_fg_state").Scan(&count); e != nil || count != 0 {
				t.Fatal("startup wrote an incompatible table", e, count)
			}
		})
	}
}
func TestMySQLUnexpectedRowsPreserved(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`CREATE TABLE cognara_fg_state(id INT PRIMARY KEY,schema_version INT NOT NULL,body JSON NOT NULL) ENGINE=InnoDB`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO cognara_fg_state VALUES(2,99,'{"marker":"keep"}')`); e != nil {
		t.Fatal(e)
	}
	if s, e := OpenMySQL(dsn); e == nil {
		s.Close()
		t.Fatal("unexpected rows accepted")
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM cognara_fg_state").Scan(&count)
	if count != 1 {
		t.Fatal("existing state changed")
	}
	var marker string
	if e = db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(body,'$.marker')) FROM cognara_fg_state WHERE id=2`).Scan(&marker); e != nil || marker != "keep" {
		t.Fatal("existing data lost", e)
	}
}
func TestMySQLConcurrentBootstrapPreservesCommittedData(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	var wg sync.WaitGroup
	results := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, e := OpenMySQL(dsn)
			if e == nil {
				e = s.Update(func(st *State) error { st.Settings["bootstrap-marker"] = []byte(`"keep"`); return nil })
				s.Close()
			}
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if string(s.Read().Settings["bootstrap-marker"]) != `"keep"` {
		t.Fatal("bootstrap reset state")
	}
	if e = s.Update(func(st *State) error {
		st.Settings["bootstrap-marker"] = []byte(`"discard"`)
		return errors.New("rollback")
	}); e == nil {
		t.Fatal("failure committed")
	}
	reopened, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if string(reopened.Read().Settings["bootstrap-marker"]) != `"keep"` {
		t.Fatal("failed transaction persisted")
	}
}
