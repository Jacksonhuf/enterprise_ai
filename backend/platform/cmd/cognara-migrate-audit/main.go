package main

import (
	"context"
	"encoding/json"
	"flag"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"log"
	"os"
)

func main() {
	env := flag.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "MySQL DSN environment variable")
	apply := flag.Bool("apply", false, "explicitly migrate v1 audit storage to v2")
	check := flag.Bool("check", false, "read-only audit migration preflight; no schema or data changes")
	flag.Parse()
	if *apply == *check || flag.NArg() != 0 {
		log.Fatal("choose exactly one of --check or --apply; export and verify v1 snapshot before apply")
	}
	if *check {
		report, e := app.CheckMySQLAuditMigration(context.Background(), os.Getenv(*env))
		if e != nil {
			log.Fatal(e)
		}
		if e = json.NewEncoder(os.Stdout).Encode(report); e != nil {
			log.Fatal("migration report output failed")
		}
		return
	}
	if e := app.MigrateMySQLAudit(context.Background(), os.Getenv(*env)); e != nil {
		log.Fatal(e)
	}
	log.Print("audit migration verified; state uses schema version 2")
}
