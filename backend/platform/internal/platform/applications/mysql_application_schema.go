package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

//go:embed migrations/002_applications.sql
var mysqlApplicationDDL string

// Internal migration building block. No runtime/CLI calls this yet. MySQL DDL
// commits independently; failures may leave partial tables, never auto-drop them.
func createMySQLApplicationSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(mysqlApplicationDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

type applicationSchemaQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type applicationSchemaSpec struct {
	columns  map[string]string
	nullable string
	indexes  map[string]string
	foreign  map[string]string
	checks   map[string]string
}

func applicationSchemaSpecs() map[string]applicationSchemaSpec {
	ids := func(names ...string) map[string]string {
		m := map[string]string{"tenant_id": "varbinary(64)"}
		for _, name := range names {
			m[name] = "varbinary(512)"
		}
		return m
	}
	apps := ids("app_id", "agent_id", "owner_id", "current_release_id")
	apps["updated_at"] = "varchar(64)"
	apps["metadata"] = "json"
	apps["releases_is_null"] = "tinyint(1)"
	drafts := ids("app_id", "revision")
	drafts["config"] = "json"
	releases := ids("app_id", "release_id", "published_by")
	releases["ordinal"] = "bigint unsigned"
	releases["snapshot"] = "json"
	releases["published_at"] = "varchar(64)"
	releases["metadata"] = "json"
	checks := ids("check_id", "app_id", "draft_revision", "user_id", "validation_id")
	checks["expires_at"] = "varchar(64)"
	checks["metadata"] = "json"
	parent := "cognara_fg_applications"
	fk := func(name string, columns, reference []string, table string) map[string]string {
		m := map[string]string{}
		for i, c := range columns {
			m[fmt.Sprintf("%s:%d", name, i+1)] = c + ":" + table + ":" + reference[i] + ":RESTRICT:RESTRICT"
		}
		return m
	}
	return map[string]applicationSchemaSpec{
		parent:                            {apps, "current_release_id", schemaIndexes(map[string][]string{"PRIMARY": {"tenant_id", "app_id"}, "application_id": {"app_id"}}, map[string][]string{"application_owner": {"tenant_id", "owner_id", "app_id"}, "application_current": {"tenant_id", "app_id", "current_release_id"}}), fk("application_current_fk", []string{"tenant_id", "app_id", "current_release_id"}, []string{"tenant_id", "app_id", "release_id"}, "cognara_fg_application_releases"), map[string]string{"application_metadata_object": "json_type(metadata)='OBJECT'", "application_release_shape": "releases_is_null in (0,1)"}},
		"cognara_fg_application_drafts":   {drafts, "", schemaIndexes(map[string][]string{"PRIMARY": {"tenant_id", "app_id"}}, nil), fk("draft_application_fk", []string{"tenant_id", "app_id"}, []string{"tenant_id", "app_id"}, parent), map[string]string{"draft_config_object": "json_type(config)='OBJECT'"}},
		"cognara_fg_application_releases": {releases, "", schemaIndexes(map[string][]string{"PRIMARY": {"tenant_id", "app_id", "release_id"}, "release_ordinal": {"tenant_id", "app_id", "ordinal"}}, nil), fk("release_application_fk", []string{"tenant_id", "app_id"}, []string{"tenant_id", "app_id"}, parent), map[string]string{"release_snapshot_object": "json_type(snapshot)='OBJECT'", "release_metadata_object": "json_type(metadata)='OBJECT'"}},
		"cognara_fg_application_checks":   {checks, "", schemaIndexes(map[string][]string{"PRIMARY": {"tenant_id", "check_id"}, "check_id": {"check_id"}}, map[string][]string{"check_application": {"tenant_id", "app_id", "check_id"}}), fk("check_application_fk", []string{"tenant_id", "app_id"}, []string{"tenant_id", "app_id"}, parent), map[string]string{"check_metadata_object": "json_type(metadata)='OBJECT'"}},
	}
}
func schemaIndexes(unique, ordinary map[string][]string) map[string]string {
	result := map[string]string{}
	for name, columns := range unique {
		for i, c := range columns {
			result[fmt.Sprintf("%s:%d", name, i+1)] = c + ":0"
		}
	}
	for name, columns := range ordinary {
		for i, c := range columns {
			result[fmt.Sprintf("%s:%d", name, i+1)] = c + ":1"
		}
	}
	return result
}
func validateMySQLApplicationSchema(ctx context.Context, q applicationSchemaQuery) error {
	for table, spec := range applicationSchemaSpecs() {
		if err := validateApplicationTable(ctx, q, table, spec); err != nil {
			return fmt.Errorf("application schema incompatible (%s): %w", table, err)
		}
	}
	return nil
}
func validateApplicationTable(ctx context.Context, q applicationSchemaQuery, table string, spec applicationSchemaSpec) error {
	var engine, collation, database string
	if err := q.QueryRowContext(ctx, `SELECT ENGINE,TABLE_COLLATION,TABLE_SCHEMA FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`, table).Scan(&engine, &collation, &database); err != nil {
		return err
	}
	if engine != "InnoDB" || collation != "utf8mb4_bin" {
		return errors.New("engine or collation differs")
	}
	rows, err := q.QueryContext(ctx, `SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,EXTRA,CHARACTER_SET_NAME,COLLATION_NAME,COLUMN_DEFAULT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`, table)
	if err != nil {
		return err
	}
	count := 0
	for rows.Next() {
		var name, kind, nullable, extra string
		var charset, collation, defaultValue sql.NullString
		if err = rows.Scan(&name, &kind, &nullable, &extra, &charset, &collation, &defaultValue); err != nil {
			rows.Close()
			return err
		}
		wantNullable := "NO"
		if name == spec.nullable {
			wantNullable = "YES"
		}
		text := strings.HasPrefix(kind, "varchar")
		if spec.columns[name] != kind || nullable != wantNullable || extra != "" || defaultValue.Valid || (text && (charset.String != "utf8mb4" || collation.String != "utf8mb4_bin")) {
			rows.Close()
			return errors.New("column differs")
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != len(spec.columns) {
		return errors.New("column count differs")
	}
	rows, err = q.QueryContext(ctx, `SELECT INDEX_NAME,SEQ_IN_INDEX,COLUMN_NAME,NON_UNIQUE,SUB_PART,INDEX_TYPE,IS_VISIBLE,EXPRESSION FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?`, table)
	if err != nil {
		return err
	}
	count = 0
	for rows.Next() {
		var name, column, kind, visible string
		var ordinal, nonunique int
		var prefix sql.NullInt64
		var expression sql.NullString
		if err = rows.Scan(&name, &ordinal, &column, &nonunique, &prefix, &kind, &visible, &expression); err != nil {
			rows.Close()
			return err
		}
		if spec.indexes[fmt.Sprintf("%s:%d", name, ordinal)] != fmt.Sprintf("%s:%d", column, nonunique) || prefix.Valid || expression.Valid || kind != "BTREE" || visible != "YES" {
			rows.Close()
			return errors.New("index differs")
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != len(spec.indexes) {
		return errors.New("index count differs")
	}
	rows, err = q.QueryContext(ctx, `SELECT k.CONSTRAINT_NAME,k.ORDINAL_POSITION,k.COLUMN_NAME,k.REFERENCED_TABLE_NAME,k.REFERENCED_COLUMN_NAME,r.UPDATE_RULE,r.DELETE_RULE,k.REFERENCED_TABLE_SCHEMA FROM information_schema.KEY_COLUMN_USAGE k JOIN information_schema.REFERENTIAL_CONSTRAINTS r ON r.CONSTRAINT_SCHEMA=k.CONSTRAINT_SCHEMA AND r.CONSTRAINT_NAME=k.CONSTRAINT_NAME AND r.TABLE_NAME=k.TABLE_NAME WHERE k.TABLE_SCHEMA=DATABASE() AND k.TABLE_NAME=?`, table)
	if err != nil {
		return err
	}
	count = 0
	for rows.Next() {
		var name, column, parent, parentColumn, updateRule, deleteRule, schema string
		var ordinal int
		if err = rows.Scan(&name, &ordinal, &column, &parent, &parentColumn, &updateRule, &deleteRule, &schema); err != nil {
			rows.Close()
			return err
		}
		actual := strings.Join([]string{column, parent, parentColumn, updateRule, deleteRule}, ":")
		if schema != database || spec.foreign[fmt.Sprintf("%s:%d", name, ordinal)] != actual {
			rows.Close()
			return errors.New("foreign key differs")
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != len(spec.foreign) {
		return errors.New("foreign key count differs")
	}
	rows, err = q.QueryContext(ctx, `SELECT t.CONSTRAINT_NAME,c.CHECK_CLAUSE,t.ENFORCED FROM information_schema.TABLE_CONSTRAINTS t JOIN information_schema.CHECK_CONSTRAINTS c ON c.CONSTRAINT_SCHEMA=t.CONSTRAINT_SCHEMA AND c.CONSTRAINT_NAME=t.CONSTRAINT_NAME WHERE t.TABLE_SCHEMA=DATABASE() AND t.TABLE_NAME=? AND t.CONSTRAINT_TYPE='CHECK'`, table)
	if err != nil {
		return err
	}
	count = 0
	for rows.Next() {
		var name, clause, enforced string
		if err = rows.Scan(&name, &clause, &enforced); err != nil {
			rows.Close()
			return err
		}
		expected, ok := spec.checks[name]
		if !ok || enforced != "YES" || normalizeApplicationCheck(clause) != normalizeApplicationCheck(expected) {
			rows.Close()
			return fmt.Errorf("check differs: %s", name)
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if count != len(spec.checks) {
		return errors.New("check count differs")
	}
	return nil
}

// Only the simple equality/IN checks above are supported; string literal case is
// retained. MySQL's harmless identifier quoting, introducer and grouping vary.
func normalizeApplicationCheck(value string) string {
	var out strings.Builder
	quoted := false
	for i := 0; i < len(value); i++ {
		c := value[i]
		// information_schema escapes the quotes of this exact known literal.
		// Do not generally unescape SQL or alter arbitrary literal contents.
		escapedObject := `_utf8mb4\'OBJECT\'`
		if !quoted && strings.HasPrefix(value[i:], escapedObject) {
			out.WriteString("'OBJECT'")
			i += len(escapedObject) - 1
			continue
		}
		if !quoted && strings.HasPrefix(value[i:], "_utf8mb4'") {
			i += len("_utf8mb4") - 1
			continue
		}
		if c == '\'' {
			quoted = !quoted
			out.WriteByte(c)
			continue
		}
		if !quoted && (unicode.IsSpace(rune(c)) || c == '`') {
			continue
		}
		if !quoted && c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out.WriteByte(c)
	}
	result := out.String()
	// Strip only grouping enclosing the entire expression, never function/IN
	// parentheses or operand structure.
	for len(result) > 1 && result[0] == '(' && result[len(result)-1] == ')' {
		depth := 0
		literal := false
		encloses := true
		for i := 0; i < len(result); i++ {
			c := result[i]
			if c == '\'' {
				literal = !literal
			}
			if literal {
				continue
			}
			if c == '(' {
				depth++
			}
			if c == ')' {
				depth--
			}
			if depth == 0 && i < len(result)-1 {
				encloses = false
				break
			}
		}
		if !encloses || depth != 0 {
			break
		}
		result = result[1 : len(result)-1]
	}
	return result
}
