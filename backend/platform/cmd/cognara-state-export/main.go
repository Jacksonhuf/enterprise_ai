// cognara-state-export reads a transitional MySQL state without initializing it.
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
	env := flag.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "environment variable containing MySQL DSN")
	output := flag.String("output", "", "new snapshot file path; must not exist")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		log.Fatal("--output is required; credentials must use an environment variable")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	report, e := app.ExportMySQLSnapshot(ctx, os.Getenv(*env), *output)
	if e != nil {
		log.Fatal(e)
	}
	if e = json.NewEncoder(os.Stdout).Encode(report); e != nil {
		log.Fatal("snapshot created but report output failed")
	}
}
