// Package opscli consolidates the shared CLI lifecycle for bounded offline
// encryption backfill commands without duplicating flag, key, and DB wiring.
package opscli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"suining-care/internal/config"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

// BackfillSpec describes one bounded offline encryption backfill operation.
type BackfillSpec[R any] struct {
	Name             string
	ApplyUsage       string
	LimitUsage       string
	Subject          string
	RequireKeyOnScan bool
	Run              func(ctx context.Context, db *sql.DB, box *securestore.Box, limit int, apply bool) (R, error)
}

// RunBackfill parses standard --apply/--limit flags, initializes securestore and
// MySQL with a bounded timeout, executes spec.Run, and emits JSON to output.
func RunBackfill[R any](spec BackfillSpec[R], args []string, output io.Writer) error {
	flags := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, spec.ApplyUsage)
	limit := flags.Int("limit", 100, spec.LimitUsage)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(output)
			flags.PrintDefaults()
			return nil
		}
		return fmt.Errorf("invalid %s arguments; use --help", spec.Subject)
	}
	if flags.NArg() != 0 || *limit < 1 || *limit > 1000 {
		return errors.New("provide a limit between 1 and 1000 without extra arguments")
	}

	var box *securestore.Box
	if spec.RequireKeyOnScan || *apply {
		var err error
		box, err = securestore.New(config.Secret("DATA_ENCRYPTION_KEY"))
		if err != nil || box == nil {
			return fmt.Errorf("valid service encryption key required for %s", spec.Subject)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := db.OpenMySQL(ctx, config.Secret("MYSQL_DSN"))
	if err != nil {
		return errors.New("cannot open explicitly configured MySQL")
	}
	defer pool.Close()

	report, operationErr := spec.Run(ctx, pool.DB, box, *limit, *apply)
	if err := json.NewEncoder(output).Encode(report); err != nil {
		return fmt.Errorf("cannot write %s aggregate report", spec.Subject)
	}
	return operationErr
}
