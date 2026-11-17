package applications

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestSettingsSchemaDDLValidates(t *testing.T) {
	ctx := context.Background()
	db := preflightDB(t, testsupport.MySQLDSN(t))
	if err := createMySQLSettingsSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := validateMySQLSettingsSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
}