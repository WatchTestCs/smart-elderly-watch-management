// Command rekey-projection rewraps existing PG location ciphertext for the
// explicitly configured MySQL source; it never rebuilds or replaces MySQL.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/config"
	"suining-care/internal/db"
	"suining-care/internal/rekey"
	"suining-care/internal/securestore"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("rekey-projection", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, "rewrap the selected source's existing encrypted PG coordinates")
	previousFile := flags.String("previous-key-file", "", "private previous DATA_ENCRYPTION_KEY file; lookup key must stay unchanged")
	batch := flags.Int("batch-size", 100, "1..1000 projection rows per page")
	maxRows := flags.Int64("max-rows", 1000000, "stop incomplete if the selected source exceeds this bound")
	pause := flags.Duration("pause", 20*time.Millisecond, "pause between pages, 0..1m")
	timeout := flags.Duration("timeout", 15*time.Minute, "overall deadline, at most 24h")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(output)
			flags.PrintDefaults()
			return nil
		}
		return errors.New("invalid projection command arguments; use --help")
	}
	if flags.NArg() != 0 || *previousFile == "" || *timeout <= 0 || *timeout > 24*time.Hour || *batch < 1 || *batch > 1000 || *maxRows < 1 || *pause < 0 || *pause > time.Minute {
		return errors.New("provide a private previous key file and valid bounded projection options")
	}
	data, err := os.ReadFile(*previousFile)
	if err != nil {
		return errors.New("cannot read previous key file")
	}
	previous, err := securestore.New(strings.TrimSpace(string(data)))
	if err != nil {
		return errors.New("invalid previous key configuration")
	}
	current, err := securestore.New(config.Secret("DATA_ENCRYPTION_KEY"))
	if err != nil || current.ActiveKeyID() == "" || !current.SameLookupKey(previous) {
		return errors.New("current keyring must have an active ID and preserve the previous lookup key")
	}
	pgURL := config.Secret("PG_ANALYTICS_URL")
	pgConfig, err := pgxpool.ParseConfig(pgURL)
	if err != nil || pgURL == "" {
		return errors.New("explicit PG_ANALYTICS_URL is required")
	}
	pgConfig.MaxConns = 2
	pgConfig.MinConns = 0
	pgConfig.ConnConfig.ConnectTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	primary, err := db.OpenMySQL(ctx, config.Secret("MYSQL_DSN"))
	if err != nil {
		return errors.New("cannot open explicitly configured MySQL source")
	}
	var source string
	err = primary.SQL().QueryRowContext(ctx, "SELECT source_id FROM analytics_source WHERE singleton=1").Scan(&source)
	primary.Close()
	if err != nil {
		return errors.New("cannot identify the configured MySQL analytics source")
	}
	pg, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return errors.New("cannot open explicitly configured PG projection")
	}
	defer pg.Close()
	report, operationErr := rekey.RunProjection(ctx, pg, source, current, previous, rekey.Options{Apply: *apply, BatchSize: *batch, MaxRowsPerColumn: *maxRows, Pause: *pause})
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return errors.New("cannot write aggregate projection report")
	}
	return operationErr
}
