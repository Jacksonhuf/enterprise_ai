package main

import (
	"context"
	"encoding/json"
	"flag"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	env := flag.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "target MySQL DSN environment variable")
	input := flag.String("input", "", "verified snapshot file")
	sha := flag.String("expected-sha256", "", "expected source SHA-256")
	version := flag.Int("source-schema-version", 0, "explicit source schema version (1 or 2)")
	flag.Parse()
	if *input == "" || *sha == "" || *version == 0 || flag.NArg() != 0 {
		log.Fatal("input, expected-sha256 and source-schema-version are required")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	report, e := app.ImportMySQLSnapshot(ctx, os.Getenv(*env), *input, *sha, *version)
	if e != nil {
		log.Fatal(e)
	}
	if e = json.NewEncoder(os.Stdout).Encode(report); e != nil {
		log.Fatal("import committed but report output failed; inspect target before retry")
	}
}
