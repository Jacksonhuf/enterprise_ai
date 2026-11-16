package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/go-sql-driver/mysql"
	"time"
)

// OpenMySQL is an explicit validation adapter; production normalized tables remain separate work.
func OpenMySQL(dsn string) (*Store, error) { return OpenMySQLContext(context.Background(), dsn) }
func OpenMySQLContext(parent context.Context, dsn string) (*Store, error) {
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil || cfg.DBName == "" || cfg.MultiStatements {
		return nil, errors.New("invalid MySQL database configuration")
	}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return nil, errors.New("MySQL driver initialization failed")
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	bad := func(message string) (*Store, error) { db.Close(); return nil, errors.New(message) }
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		return bad("MySQL connection unavailable")
	}
	// MySQL DDL commits independently. Never imply transactional migration rollback.
	if _, e = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS cognara_fg_state (id INT PRIMARY KEY CHECK(id=1), schema_version INT NOT NULL, body JSON NOT NULL CHECK(JSON_TYPE(body)='OBJECT')) ENGINE=InnoDB`); e != nil {
		return bad("MySQL schema initialization failed")
	}
	s := &Store{db: db, dialect: "mysql"}
	s.init()
	s.seed()
	raw, e := json.Marshal(s.state)
	if e != nil {
		return bad("MySQL initial state invalid")
	}
	// Concurrent initialization preserves the existing record, including unknown versions.
	if _, e = db.ExecContext(ctx, `INSERT INTO cognara_fg_state(id,schema_version,body) VALUES(1,1,?) ON DUPLICATE KEY UPDATE id=id`, string(raw)); e != nil {
		return bad("MySQL initial state unavailable")
	}
	if _, e = s.readSQLContext(ctx); e != nil {
		return bad("MySQL stored state or schema version unsupported")
	}
	return s, nil
}
