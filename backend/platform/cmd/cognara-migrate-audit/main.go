package main

import (
	"context"
	"flag"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"log"
	"os"
)

func main() {
	env := flag.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "MySQL DSN environment variable")
	apply := flag.Bool("apply", false, "explicitly migrate v1 audit storage to v2")
	flag.Parse()
	if !*apply || flag.NArg() != 0 {
		log.Fatal("explicit --apply required; export and verify v1 snapshot first")
	}
	if e := app.MigrateMySQLAudit(context.Background(), os.Getenv(*env)); e != nil {
		log.Fatal(e)
	}
	log.Print("audit migration verified; state uses schema version 2")
}
