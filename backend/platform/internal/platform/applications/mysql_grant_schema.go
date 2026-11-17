package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed migrations/003_grants.sql
var mysqlGrantDDL string

// Internal migration building block for R06. No runtime/CLI apply yet.
func createMySQLGrantSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(mysqlGrantDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func grantSchemaSpecs() map[string]applicationSchemaSpec {
	ids := map[string]string{
		"tenant_id": "varbinary(64)", "grant_id": "varbinary(512)",
		"subject_type": "varbinary(64)", "subject_id": "varbinary(512)",
		"resource_type": "varbinary(64)", "resource_id": "varbinary(512)",
		"action": "varbinary(128)", "scope": "varbinary(128)",
		"active": "tinyint(1)", "metadata": "json", "raw_payload": "longblob",
	}
	return map[string]applicationSchemaSpec{
		"cognara_fg_grants": {
			columns: ids,
			indexes: schemaIndexes(
				map[string][]string{
					"PRIMARY":         {"tenant_id", "grant_id"},
					"grant_global_id": {"grant_id"},
				},
				map[string][]string{
					"grant_subject":  {"tenant_id", "subject_type", "subject_id", "resource_type", "resource_id", "action", "scope"},
					"grant_resource": {"tenant_id", "resource_type", "resource_id", "action", "scope"},
				},
			),
			checks: map[string]string{"grant_metadata_object": "json_type(metadata)='OBJECT'"},
		},
	}
}

func validateMySQLGrantSchema(ctx context.Context, q applicationSchemaQuery) error {
	for table, spec := range grantSchemaSpecs() {
		if err := validateApplicationTable(ctx, q, table, spec); err != nil {
			return err
		}
	}
	return nil
}

func scanGrantLayout(ctx context.Context, q applicationSchemaQuery) ([]grantLayoutRow, error) {
	rows, err := q.QueryContext(ctx, `SELECT tenant_id,grant_id,raw_payload FROM cognara_fg_grants ORDER BY tenant_id,grant_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layout []grantLayoutRow
	for rows.Next() {
		var tenant, grantID, payload []byte
		if err = rows.Scan(&tenant, &grantID, &payload); err != nil {
			return nil, err
		}
		layout = append(layout, grantLayoutRow{TenantID: string(tenant), GrantID: string(grantID), Payload: append(json.RawMessage(nil), payload...)})
	}
	return layout, rows.Err()
}