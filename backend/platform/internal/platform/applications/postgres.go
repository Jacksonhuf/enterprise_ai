package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"strings"
	"time"
)

// OpenPostgres preserves the current aggregate interface in JSONB. It is a
// transactional validation adapter, not a normalized multi-tenant data model.
// No DSN or driver diagnostics are returned, because they may contain secrets.
func OpenPostgres(dsn string) (*Store, error) { return OpenPostgresContext(context.Background(), dsn) }
func OpenPostgresContext(parent context.Context, dsn string) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("PostgreSQL connection configuration is required")
	}
	if _, err := pgx.ParseConfig(dsn); err != nil {
		return nil, errors.New("invalid PostgreSQL connection configuration")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, errors.New("PostgreSQL driver initialization failed")
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	bad := func(message string) (*Store, error) { db.Close(); return nil, errors.New(message) }
	if err = db.PingContext(ctx); err != nil {
		return bad("PostgreSQL connection unavailable")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return bad("PostgreSQL migration transaction unavailable")
	}
	defer tx.Rollback()
	// Serialize bootstrap DDL so concurrent hosts do not race catalog creation.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1129277262,1)`); err != nil {
		return bad("PostgreSQL migration lock unavailable")
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS cognara_fg_state (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL, body JSONB NOT NULL CHECK(jsonb_typeof(body)='object'))`); err != nil {
		return bad("PostgreSQL schema initialization failed")
	}
	s := &Store{db: db, dialect: "postgres"}
	s.init()
	s.seed()
	raw, err := json.Marshal(s.state)
	if err != nil {
		return bad("PostgreSQL initial state invalid")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_state(id,schema_version,body) VALUES(1,1,$1::jsonb) ON CONFLICT(id) DO NOTHING`, string(raw)); err != nil {
		return bad("PostgreSQL initial state unavailable")
	}
	if err = tx.Commit(); err != nil {
		return bad("PostgreSQL schema transaction failed")
	}
	if _, err = s.readSQLContext(ctx); err != nil {
		return bad("PostgreSQL stored state or schema version unsupported")
	}
	return s, nil
}
