// Command protect-user-phones scans, protects, rotates or restores users phones.
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
	"suining-care/internal/securestore"
	"suining-care/internal/userphone"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("protect-user-phones", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, "protect/rewrap after authentication preflight; default is aggregate scan only")
	rollback := flags.Bool("rollback", false, "with --apply, restore original phones for a controlled application rollback")
	batch := flags.Int("batch-size", 100, "1..1000 rows per page")
	maxRows := flags.Int64("max-rows", 1000000, "fail incomplete if this row bound is exceeded")
	pause := flags.Duration("pause", 20*time.Millisecond, "pause between pages, 0..1m")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall deadline, at most 24h")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(output)
			flags.PrintDefaults()
			return nil
		}
		return errors.New("invalid administrator phone command arguments")
	}
	if flags.NArg() != 0 || *timeout <= 0 || *timeout > 24*time.Hour || (*rollback && !*apply) || *batch < 1 || *batch > 1000 || *maxRows < 1 || *pause < 0 || *pause > time.Minute {
		return errors.New("invalid arguments; --rollback requires --apply and a bounded deadline")
	}
	box, err := securestore.New(config.Secret("DATA_ENCRYPTION_KEY"))
	if err != nil {
		return errors.New("invalid administrator phone key configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	pool, err := db.OpenMySQL(ctx, config.Secret("MYSQL_DSN"))
	if err != nil {
		return errors.New("cannot open explicitly configured MySQL")
	}
	defer pool.Close()
	report, operationErr := userphone.Run(ctx, pool.SQL(), box, userphone.Options{Apply: *apply, Rollback: *rollback, BatchSize: *batch, MaxRows: *maxRows, Pause: *pause})
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return errors.New("cannot write administrator phone aggregate report")
	}
	return operationErr
}
