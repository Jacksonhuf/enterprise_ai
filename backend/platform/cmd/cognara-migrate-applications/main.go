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
	flags := flag.NewFlagSet("cognara-migrate-applications", flag.ContinueOnError)
	flags.SetOutput(stderr)
	env := flags.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "MySQL DSN environment variable; never pass a DSN on the command line")
	check := flags.Bool("check", false, "read-only application migration plan; apply is not implemented")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*check || flags.NArg() != 0 {
		io.WriteString(stderr, "choose --check; application migration apply is not implemented\n")
		return 2
	}
	report, err := app.PlanMySQLApplications(context.Background(), getenv(*env))
	if err != nil {
		io.WriteString(stderr, err.Error()+"\n")
		return 1
	}
	if json.NewEncoder(stdout).Encode(report) != nil {
		io.WriteString(stderr, "application migration plan output failed\n")
		return 1
	}
	return 0
}
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv)) }
