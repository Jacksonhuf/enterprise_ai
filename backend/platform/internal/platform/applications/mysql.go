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
	if e = validateMySQLStateSchema(ctx, db, cfg.DBName); e != nil {
		return bad("MySQL state schema incompatible; explicit migration required")
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

// Do not silently use a pre-existing table whose storage semantics differ.
func validateMySQLStateSchema(ctx context.Context, db *sql.DB, database string) error {
	var engine string
	if e := db.QueryRowContext(ctx, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME='cognara_fg_state'`, database).Scan(&engine); e != nil {
		return e
	}
	if engine != "InnoDB" {
		return errors.New("transactional storage required")
	}
	rows, e := db.QueryContext(ctx, `SELECT COLUMN_NAME,DATA_TYPE,IS_NULLABLE,COLUMN_KEY,EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=? AND TABLE_NAME='cognara_fg_state'`, database)
	if e != nil {
		return e
	}
	type column struct{ kind, nullable, key, extra string }
	columns := map[string]column{}
	for rows.Next() {
		var name string
		var c column
		if e = rows.Scan(&name, &c.kind, &c.nullable, &c.key, &c.extra); e != nil {
			rows.Close()
			return e
		}
		columns[name] = c
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(columns) != 3 {
		return errors.New("unexpected state columns")
	}
	for name, kind := range map[string]string{"id": "int", "schema_version": "int", "body": "json"} {
		c, ok := columns[name]
		if !ok || c.kind != kind || c.nullable != "NO" || c.extra != "" {
			return errors.New("incompatible state column")
		}
	}
	if columns["id"].key != "PRI" || columns["schema_version"].key == "PRI" || columns["body"].key == "PRI" {
		return errors.New("incompatible state primary key")
	}
	var unexpected int
	if e = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_state WHERE id<>1`).Scan(&unexpected); e != nil {
		return e
	}
	if unexpected != 0 {
		return errors.New("unexpected state rows")
	}
	return nil
}
