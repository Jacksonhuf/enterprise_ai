package applications

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"net/url"
	"os"
	"path/filepath"
)

// OpenSQLite persists the existing aggregate contract in a transactional local database.
// Each update acquires an IMMEDIATE SQLite transaction and reloads authoritative state.
// This adapter is local/shared-file persistence, not a distributed Postgres deployment.
func OpenSQLite(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path required")
	}
	abs, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(filepath.Dir(abs), 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(abs, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	f.Close()
	u := url.URL{Scheme: "file", Path: abs}
	db, e := sql.Open("sqlite3", u.String()+"?_busy_timeout=5000&_txlock=immediate&_journal_mode=WAL")
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	bad := func(err error) (*Store, error) { db.Close(); return nil, err }
	if _, e = db.Exec(`CREATE TABLE IF NOT EXISTS cognara_fg_state (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL, body TEXT NOT NULL)`); e != nil {
		return bad(e)
	}
	s := &Store{db: db}
	s.init()
	s.seed()
	b, _ := json.Marshal(s.state)
	if _, e = db.Exec(`INSERT OR IGNORE INTO cognara_fg_state(id,schema_version,body) VALUES(1,1,?)`, string(b)); e != nil {
		return bad(e)
	}
	if _, e = s.readSQL(); e != nil {
		return bad(e)
	}
	return s, nil
}
func (s *Store) readSQL() (State, error) {
	var raw []byte
	var version int
	if e := s.db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); e != nil {
		return State{}, e
	}
	if version != 1 {
		return State{}, fmt.Errorf("unsupported state schema")
	}
	var st State
	e := json.Unmarshal(raw, &st)
	return st, e
}
func (s *Store) updateSQL(fn func(*State) error) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var raw []byte
	var version int
	if e = tx.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); e != nil {
		return e
	}
	if version != 1 {
		return fmt.Errorf("unsupported state schema")
	}
	var next State
	if e = json.Unmarshal(raw, &next); e != nil {
		return e
	}
	if e = fn(&next); e != nil {
		return e
	}
	b, e := json.Marshal(next)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, string(b)); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	s.state = next
	return nil
}
func (s *Store) Check() error {
	if s.db == nil {
		return nil
	}
	return s.db.Ping()
}
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
