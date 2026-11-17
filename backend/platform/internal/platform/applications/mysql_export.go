package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/go-sql-driver/mysql"
	"os"
	"path/filepath"
	"time"
)

type SnapshotReport struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	AuditEvents         int    `json:"audit_events"`
	SHA256              string `json:"sha256"`
	Bytes               int    `json:"bytes"`
	Applications        int    `json:"applications"`
	Sessions            int    `json:"sessions"`
	Messages            int    `json:"messages"`
}

// ExportMySQLSnapshot never initializes or seeds a database. The destination must not exist.
// The raw aggregate is a migration checkpoint, not a normalized-table backup or restore guarantee.
func ExportMySQLSnapshot(parent context.Context, dsn, destination string) (SnapshotReport, error) {
	var report SnapshotReport
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil || cfg.DBName == "" || cfg.MultiStatements || destination == "" {
		return report, errors.New("invalid MySQL snapshot configuration")
	}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return report, errors.New("MySQL snapshot connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if e = validateMySQLStateSchema(ctx, db, cfg.DBName); e != nil {
		return report, errors.New("MySQL snapshot schema incompatible or unavailable")
	}
	tx, e := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return report, errors.New("MySQL snapshot transaction unavailable")
	}
	defer tx.Rollback()
	var raw []byte
	var version int
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); e != nil {
		return report, errors.New("MySQL snapshot state unavailable")
	}
	if (version != 1 && version != 2 && version != 3 && version != 4 && version != 5 && version != 6) || len(raw) == 0 || len(raw) > 64<<20 {
		return report, errors.New("MySQL snapshot version or size unsupported")
	}
	report.SourceSchemaVersion = version
	if version == 6 {
		if e = validateMySQLApplicationSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v6 snapshot application schema incompatible")
		}
		if e = validateMySQLGrantSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v6 snapshot grant schema incompatible")
		}
		if e = validateMySQLSessionSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v6 snapshot session schema incompatible")
		}
		if e = validateMySQLSettingsSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v6 snapshot settings schema incompatible")
		}
		st, _, e := readMySQLApplicationV6(ctx, tx, true)
		if e != nil {
			return report, errors.New("MySQL v6 snapshot state invalid")
		}
		raw, report.AuditEvents, e = exportMySQLV6(ctx, tx)
		if e != nil {
			return SnapshotReport{}, errors.New("MySQL v6 snapshot invalid or unavailable")
		}
		report.Applications = len(st.Apps)
		report.Sessions = len(st.Sessions)
		for _, s := range st.Sessions {
			report.Messages += len(s.Messages)
		}
	} else if version == 5 {
		if e = validateMySQLApplicationSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v5 snapshot application schema incompatible")
		}
		if e = validateMySQLGrantSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v5 snapshot grant schema incompatible")
		}
		if e = validateMySQLSessionSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v5 snapshot session schema incompatible")
		}
		st, _, e := readMySQLApplicationV5(ctx, tx, true)
		if e != nil {
			return report, errors.New("MySQL v5 snapshot state invalid")
		}
		raw, report.AuditEvents, e = exportMySQLV5(ctx, tx)
		if e != nil {
			return SnapshotReport{}, errors.New("MySQL v5 snapshot invalid or unavailable")
		}
		report.Applications = len(st.Apps)
		report.Sessions = len(st.Sessions)
		for _, s := range st.Sessions {
			report.Messages += len(s.Messages)
		}
	} else if version == 4 {
		if e = validateMySQLApplicationSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v4 snapshot application schema incompatible")
		}
		if e = validateMySQLGrantSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v4 snapshot grant schema incompatible")
		}
		st, _, e := readMySQLApplicationV4(ctx, tx, true)
		if e != nil {
			return report, errors.New("MySQL v4 snapshot state invalid")
		}
		raw, report.AuditEvents, e = exportMySQLV4(ctx, tx)
		if e != nil {
			return SnapshotReport{}, errors.New("MySQL v4 snapshot invalid or unavailable")
		}
		report.Applications = len(st.Apps)
		report.Sessions = len(st.Sessions)
		for _, s := range st.Sessions {
			report.Messages += len(s.Messages)
		}
	} else if version == 3 {
		if e = validateMySQLApplicationSchema(ctx, db); e != nil {
			return report, errors.New("MySQL v3 snapshot schema incompatible")
		}
		st, _, e := readMySQLApplicationV3(ctx, tx, true)
		if e != nil {
			return report, errors.New("MySQL v3 snapshot state invalid")
		}
		raw, report.AuditEvents, e = exportMySQLV3(ctx, tx)
		if e != nil {
			return SnapshotReport{}, errors.New("MySQL v3 snapshot invalid or unavailable")
		}
		report.Applications = len(st.Apps)
		report.Sessions = len(st.Sessions)
		for _, s := range st.Sessions {
			report.Messages += len(s.Messages)
		}
	} else {
		var object map[string]json.RawMessage
		var state State
		if json.Unmarshal(raw, &object) != nil || object == nil || json.Unmarshal(raw, &state) != nil {
			return report, errors.New("MySQL snapshot state invalid")
		}
		report.AuditEvents = len(state.Audit)
		if version == 2 {
			raw, report.AuditEvents, e = exportMySQLV2(ctx, tx, raw)
			if e != nil {
				return SnapshotReport{}, errors.New("MySQL v2 snapshot invalid or unavailable")
			}
		}
		report.Applications = len(state.Apps)
		report.Sessions = len(state.Sessions)
		for _, s := range state.Sessions {
			report.Messages += len(s.Messages)
		}
	}
	if e = tx.Commit(); e != nil {
		return report, errors.New("MySQL snapshot transaction failed")
	}
	sum := sha256.Sum256(raw)
	report.SHA256 = hex.EncodeToString(sum[:])
	report.Bytes = len(raw)
	if ctx.Err() != nil {
		return SnapshotReport{}, errors.New("MySQL snapshot canceled")
	}
	if e = writeNewSnapshot(destination, raw); e != nil {
		return SnapshotReport{}, errors.New("snapshot destination unavailable; existing files are never overwritten")
	}
	return report, nil
}
func writeNewSnapshot(destination string, raw []byte) error {
	temp, e := os.CreateTemp(filepath.Dir(destination), ".cognara-snapshot-*")
	if e != nil {
		return e
	}
	name := temp.Name()
	defer os.Remove(name)
	defer temp.Close()
	if _, e = temp.Write(raw); e != nil {
		return e
	}
	if e = temp.Sync(); e != nil {
		return e
	}
	if e = temp.Close(); e != nil {
		return e
	}
	// Link publishes a complete file exclusively, including when the destination is a symlink.
	return os.Link(name, destination)
}
