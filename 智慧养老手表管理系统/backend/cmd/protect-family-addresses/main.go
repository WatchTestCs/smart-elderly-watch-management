// Command protect-family-addresses explicitly migrates existing family addresses.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"suining-care/internal/config"
	"suining-care/internal/db"
	"suining-care/internal/familyaddress"
	"suining-care/internal/securestore"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("protect-family-addresses", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, "apply with relevant writers stopped and a tested backup")
	limit := flags.Int("limit", 100, "maximum addresses per invocation, 1..1000")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(output)
			flags.PrintDefaults()
			return nil
		}
		return errors.New("invalid family address backfill arguments; use --help")
	}
	if flags.NArg() != 0 || *limit < 1 || *limit > 1000 {
		return errors.New("provide a limit between 1 and 1000 without extra arguments")
	}
	box, err := securestore.New(config.Secret("DATA_ENCRYPTION_KEY"))
	if err != nil || box == nil {
		return errors.New("valid service encryption key required for family address inventory")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := db.OpenMySQL(ctx, config.Secret("MYSQL_DSN"))
	if err != nil {
		return errors.New("cannot open explicitly configured MySQL")
	}
	defer pool.Close()
	report, operationErr := familyaddress.Backfill(ctx, pool.DB, box, *limit, *apply)
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return errors.New("cannot write family address aggregate report")
	}
	return operationErr
}
