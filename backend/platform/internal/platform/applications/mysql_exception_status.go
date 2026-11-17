package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrExceptionUnavailable = errors.New("exception unavailable")

type ExceptionPatch struct {
	Status, Revision, UpdatedAt, UpdatedBy string
	Audit                                  []Event
}

// UpdateMySQLExceptionContext projects the first exact tenant/id match. The array
// is scanned in MySQL; only the selected object is transferred to Go. Writes still
// serialize on the existing state row, preserving ordinal stability and audit order.
func (s *Store) UpdateMySQLExceptionContext(parent context.Context, tenant, id string, mutate func(map[string]any) (*ExceptionPatch, error)) (json.RawMessage, bool, error) {
	if s.dialect != "mysql" {
		return nil, false, nil
	}
	if tenant == "" || id == "" || mutate == nil {
		return nil, true, errors.New("invalid exception selection")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, true, e
	}
	defer tx.Rollback()
	var version int
	var keys, audit, kind []byte
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,JSON_KEYS(body),JSON_EXTRACT(body,'$.audit'),JSON_TYPE(JSON_EXTRACT(body,'$.exceptions')) FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &keys, &audit, &kind); e != nil {
		return nil, true, e
	}
	if version == 1 {
		return nil, false, nil
	}
	if version != 2 && !mySQLRelationalRuntimeVersion(version) {
		return nil, true, errors.New("unsupported state schema")
	}
	if mySQLRelationalRuntimeVersion(version) {
		if _, _, e = readMySQLApplicationRelational(ctx, tx, false, version); e != nil {
			return nil, true, e
		}
	}
	var names []string
	if json.Unmarshal(keys, &names) != nil {
		return nil, true, errors.New("invalid state envelope")
	}
	for _, name := range names {
		for _, canonical := range []string{"audit", "exceptions"} {
			if name != canonical && strings.EqualFold(name, canonical) {
				if mySQLRelationalRuntimeVersion(version) {
					return nil, true, errors.New("ambiguous schema3 exception aliases")
				}
				return nil, false, nil
			}
		}
	}
	var embedded []json.RawMessage
	if len(audit) > 0 && (json.Unmarshal(audit, &embedded) != nil || len(embedded) > 0) {
		return nil, true, errors.New("ambiguous audit storage")
	}
	if e = validateMySQLAuditSchema(ctx, tx); e != nil {
		return nil, true, e
	}
	if len(kind) == 0 || string(kind) == "NULL" {
		return nil, true, ErrExceptionUnavailable
	}
	if string(kind) != "ARRAY" {
		return nil, true, errors.New("invalid exception storage")
	}
	var ordinal uint64
	var raw []byte
	e = tx.QueryRowContext(ctx, `SELECT j.pos,JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,']')) FROM cognara_fg_state s CROSS JOIN JSON_TABLE(s.body,'$.exceptions[*]' COLUMNS(pos FOR ORDINALITY)) j WHERE s.id=1 AND JSON_TYPE(JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,'].id')))='STRING' AND CAST(JSON_UNQUOTE(JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,'].id'))) AS BINARY)=CAST(? AS BINARY) AND JSON_TYPE(JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,'].tenant_id')))='STRING' AND CAST(JSON_UNQUOTE(JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,'].tenant_id'))) AS BINARY)=CAST(? AS BINARY) ORDER BY j.pos LIMIT 1`, id, tenant).Scan(&ordinal, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, true, ErrExceptionUnavailable
	}
	if e != nil {
		return nil, true, e
	}
	if ordinal == 0 {
		return nil, true, errors.New("invalid exception ordinal")
	}
	var item map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&item) != nil || item == nil || item["id"] != id || item["tenant_id"] != tenant {
		return nil, true, errors.New("invalid selected exception")
	}
	patch, e := mutate(item)
	if e != nil {
		return nil, true, e
	}
	if patch == nil || patch.Revision == "" || len(patch.Audit) == 0 || (patch.Status != "OPEN" && patch.Status != "ACKNOWLEDGED" && patch.Status != "RESOLVED") {
		return nil, true, errors.New("invalid exception patch")
	}
	if _, e = time.Parse(time.RFC3339Nano, patch.UpdatedAt); e != nil {
		return nil, true, errors.New("invalid exception update time")
	}
	for _, event := range patch.Audit {
		if event.TenantID != tenant {
			return nil, true, errors.New("exception audit tenant mismatch")
		}
	}
	var last uint64
	if e = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); e != nil {
		return nil, true, e
	}
	if e = insertMySQLAudit(ctx, tx, patch.Audit, last); e != nil {
		return nil, true, e
	}
	base := fmt.Sprintf("$.exceptions[%d]", ordinal-1)
	if _, e = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=JSON_SET(body,?,?,?,?,?,?,?,?) WHERE id=1`, base+".status", patch.Status, base+".revision", patch.Revision, base+".updated_at", patch.UpdatedAt, base+".updated_by", patch.UpdatedBy); e != nil {
		return nil, true, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT JSON_EXTRACT(body,?) FROM cognara_fg_state WHERE id=1`, base).Scan(&raw); e != nil {
		return nil, true, e
	}
	if mySQLRelationalRuntimeVersion(version) {
		if _, _, e = readMySQLApplicationRelational(ctx, tx, false, version); e != nil {
			return nil, true, e
		}
	}
	if e = tx.Commit(); e != nil {
		return nil, true, e
	}
	return json.RawMessage(raw), true, nil
}
