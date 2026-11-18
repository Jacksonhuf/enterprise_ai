package applications

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

// TestR20Schema8GovernanceRelationalExplainSmoke records that primary tenant-scoped
// governance paging queries on schema8 can use indexes (slow-query prep evidence).
func TestR20Schema8GovernanceRelationalExplainSmoke(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	if err := PrepareSchema8IntegrationDB(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queries := []struct {
		sql  string
		args []any
	}{
		{`EXPLAIN SELECT ordinal,raw_payload FROM cognara_fg_usage WHERE tenant_id=? ORDER BY ordinal DESC LIMIT 200`, []any{"42"}},
		{`EXPLAIN SELECT ordinal,raw_payload FROM cognara_fg_exceptions WHERE tenant_id=? ORDER BY ordinal DESC LIMIT 200`, []any{"42"}},
	}
	for _, item := range queries {
		rows, err := db.QueryContext(ctx, item.sql, item.args...)
		if err != nil {
			t.Fatal(item.sql, err)
		}
		var sawPlan bool
		for rows.Next() {
			sawPlan = true
			cols, err := rows.Columns()
			if err != nil {
				rows.Close()
				t.Fatal(err)
			}
			dest := make([]sql.NullString, len(cols))
			scan := make([]any, len(cols))
			for i := range dest {
				scan[i] = &dest[i]
			}
			if err = rows.Scan(scan...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			idx := -1
			typ := ""
			for i, name := range cols {
				switch strings.ToLower(name) {
				case "key":
					idx = i
				case "type":
					typ = strings.ToUpper(dest[i].String)
				}
			}
			if typ == "ALL" {
				t.Fatalf("full table scan: %s", item.sql)
			}
			if idx >= 0 && dest[idx].Valid && dest[idx].String != "" {
				continue
			}
		}
		rows.Close()
		if !sawPlan {
			t.Fatal("no explain plan", item.sql)
		}
	}
}