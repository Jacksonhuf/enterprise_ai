package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalExceptionDomain = errors.New("exception changes require relational writers")

func stageMySQLExceptionRoot(ctx context.Context, tx *sql.Tx) error {
	var version int
	var original []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &original); err != nil {
		return err
	}
	if version != 6 {
		return errors.New("exception root staging requires schema 6")
	}
	layout, err := splitExceptionLayout(original)
	if err != nil {
		return err
	}
	if err = applyMySQLExceptionLayout(ctx, tx, original); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(layout.Root)); err != nil {
		return err
	}
	got, err := loadStagedMySQLExceptionState(ctx, tx, layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(original, got) {
		return errors.New("exception root handoff reconciliation failed")
	}
	return nil
}

func scanExceptionLayout(ctx context.Context, q applicationSchemaQuery) ([]exceptionLayoutRow, error) {
	rows, err := q.QueryContext(ctx, `SELECT tenant_id,exception_id,ordinal,status,request_id,raw_payload FROM cognara_fg_exceptions ORDER BY ordinal`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layout []exceptionLayoutRow
	for rows.Next() {
		var tenant, id, status, requestID, raw []byte
		var ordinal uint64
		if err = rows.Scan(&tenant, &id, &ordinal, &status, &requestID, &raw); err != nil {
			return nil, err
		}
		layout = append(layout, exceptionLayoutRow{
			TenantID: string(tenant), ExceptionID: string(id), Ordinal: ordinal,
			Status: string(status), RequestID: string(requestID), Payload: append(json.RawMessage(nil), raw...),
		})
	}
	return layout, rows.Err()
}

func loadStagedMySQLExceptionState(ctx context.Context, tx *sql.Tx, layout exceptionLayout) ([]byte, error) {
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return nil, err
	}
	if version != 6 {
		return nil, errors.New("exception root staging requires schema 6")
	}
	rows, err := scanExceptionLayout(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(layout.Exceptions) {
		return nil, errors.New("exception staging row count mismatch")
	}
	byID := map[string]exceptionLayoutRow{}
	for _, row := range rows {
		byID[row.ExceptionID] = row
	}
	for _, expected := range layout.Exceptions {
		row, ok := byID[expected.ExceptionID]
		if !ok || row.TenantID != expected.TenantID || row.Ordinal != expected.Ordinal || !equalApplicationJSON(expected.Payload, row.Payload) {
			return nil, errors.New("staged exception row mismatch")
		}
	}
	root, err := currentExceptionStagingRoot(ctx, tx)
	if err != nil {
		return nil, err
	}
	kind := layout.ExceptionsKind
	if len(rows) > 0 {
		kind = "array"
	}
	return joinExceptionLayout(exceptionLayout{Root: root, Exceptions: rows, ExceptionsKind: kind})
}

func currentExceptionStagingRoot(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return nil, errors.New("invalid exception staging root")
	}
	if _, ok := object["exceptions"]; ok {
		return nil, errors.New("exception staging root still contains exceptions")
	}
	return append([]byte(nil), root...), nil
}

func loadMySQLExceptionsFromTable(ctx context.Context, q applicationSchemaQuery) ([]map[string]any, error) {
	rows, err := scanExceptionLayout(ctx, q)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil {
			return nil, errors.New("exception table payload invalid")
		}
		items = append(items, item)
	}
	return items, nil
}

func insertMySQLExceptionRow(ctx context.Context, tx *sql.Tx, exception map[string]any) error {
	raw, err := json.Marshal(exception)
	if err != nil {
		return err
	}
	id, _ := exception["id"].(string)
	tenant, _ := exception["tenant_id"].(string)
	if !applicationPlanID(id, 512) || !applicationPlanID(tenant, 64) {
		return errors.New("invalid exception identity")
	}
	status, _ := exception["status"].(string)
	if status == "" {
		status = "OPEN"
	}
	requestID, _ := exception["request_id"].(string)
	if requestID == "" {
		requestID = "-"
	}
	var ordinal uint64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordinal),0)+1 FROM cognara_fg_exceptions`).Scan(&ordinal); err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{"status": status, "request_id": requestID, "ordinal": ordinal})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_exceptions(tenant_id,exception_id,ordinal,status,request_id,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
		tenant, id, ordinal, status, requestID, string(meta), raw)
	return err
}

func updateMySQLExceptionRow(ctx context.Context, tx *sql.Tx, tenant, id string, patch *ExceptionPatch, item map[string]any) error {
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	status := patch.Status
	requestID, _ := item["request_id"].(string)
	if requestID == "" {
		requestID = "-"
	}
	var ordinal uint64
	if err = tx.QueryRowContext(ctx, `SELECT ordinal FROM cognara_fg_exceptions WHERE tenant_id=? AND exception_id=?`, tenant, id).Scan(&ordinal); err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{"status": status, "request_id": requestID, "ordinal": ordinal, "revision": patch.Revision})
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE cognara_fg_exceptions SET status=?,request_id=?,metadata=?,raw_payload=? WHERE tenant_id=? AND exception_id=?`,
		status, requestID, string(meta), raw, tenant, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("exception row unavailable")
	}
	return nil
}