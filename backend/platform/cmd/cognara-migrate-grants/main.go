package main

import (
	"context"
	"encoding/json"
	"flag"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"io"
	"os"
)

func run(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("cognara-migrate-grants", flag.ContinueOnError)
	flags.SetOutput(stderr)
	env := flags.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "MySQL DSN environment variable; never pass a DSN on the command line")
	check := flags.Bool("check", false, "read-only grant migration plan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*check || flags.NArg() != 0 {
		io.WriteString(stderr, "choose --check; grant migration apply is not implemented\n")
		return 2
	}
	report, err := app.PlanMySQLGrants(context.Background(), getenv(*env))
	if err != nil {
		io.WriteString(stderr, err.Error()+"\n")
		return 1
	}
	if json.NewEncoder(stdout).Encode(report) != nil {
		io.WriteString(stderr, "grant migration plan output failed\n")
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv)) }