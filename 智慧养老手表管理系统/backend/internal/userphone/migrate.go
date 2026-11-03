package userphone

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/securestore"
)

type Options struct {
	Apply, Rollback bool
	BatchSize       int
	MaxRows         int64
	Pause           time.Duration
}

type Counts struct {
	Scanned   int64 `json:"scanned"`
	Legacy    int64 `json:"legacy"`
	Protected int64 `json:"protected"`
	Empty     int64 `json:"empty"`
	Invalid   int64 `json:"invalid"`
	Rewritten int64 `json:"rewritten"`
	Conflicts int64 `json:"conflicts"`
}

type Report struct {
	Scope                string `json:"scope"`
	Mode                 string `json:"mode"`
	Stage                string `json:"stage"`
	Before               Counts `json:"before"`
	Applied              Counts `json:"applied"`
	After                Counts `json:"after"`
	SelectedScopeClean   bool   `json:"selected_scope_clean"`
	RetirementAuthorized bool   `json:"retirement_authorized"`
}

type candidate struct {
	id int64
	Record
	oversized bool
}

// Run never prints row IDs, values, indexes, SQL driver errors or secrets. It
// authenticates the entire selected scope before the first CAS. Live scans do
// not constitute a writer barrier, asset/backup verification or key retirement.
func Run(ctx context.Context, pool *sql.DB, box *securestore.Box, options Options) (report Report, err error) {
	report = Report{Scope: "mysql_users_phone", Mode: "scan", Stage: "validation"}
	if pool == nil || box == nil || options.BatchSize < 1 || options.BatchSize > 1000 || options.MaxRows < 1 || options.Pause < 0 || options.Pause > time.Minute {
		return report, errors.New("invalid administrator phone migration configuration or bounds")
	}
	if _, ok := ctx.Deadline(); !ok {
		return report, errors.New("administrator phone migration requires an overall deadline")
	}
	if options.Apply {
		report.Mode = "protect"
		if options.Rollback {
			report.Mode = "rollback"
		}
	}
	var verifier *string
	var protected bool
	if err := pool.QueryRowContext(ctx, `SELECT (SELECT lookup_verifier FROM account_phone_key_state WHERE singleton=1),
 EXISTS(SELECT 1 FROM users WHERE user_phone_protected=true)`).Scan(&verifier, &protected); err != nil {
		return report, errors.New("cannot check administrator phone lookup binding")
	}
	if err := validateBinding(box, verifier, protected); err != nil {
		return report, err
	}
	report.Stage = "preflight"
	report.Before, err = walk(ctx, pool, box, options, false)
	if err != nil {
		return report, err
	}
	if report.Before.Invalid != 0 {
		return report, errors.New("administrator phone preflight failed; no rows were changed")
	}
	if !options.Apply {
		report.After = report.Before
		report.SelectedScopeClean = report.Before.Legacy == 0
		report.Stage = "complete"
		return report, nil
	}
	report.Stage = "apply"
	if !options.Rollback {
		// This key-specific no-op insert serializes concurrent initial binding;
		// it never overwrites another instance's verifier.
		if _, err := pool.ExecContext(ctx, `INSERT INTO account_phone_key_state(singleton,lookup_verifier) VALUES(1,?) ON DUPLICATE KEY UPDATE singleton=singleton`, lookupVerifier(box)); err != nil {
			return report, errors.New("cannot register administrator phone lookup binding")
		}
		var registered string
		if err := pool.QueryRowContext(ctx, "SELECT lookup_verifier FROM account_phone_key_state WHERE singleton=1").Scan(&registered); err != nil {
			return report, errors.New("cannot verify administrator phone lookup binding")
		}
		if err := validateBinding(box, &registered, true); err != nil {
			return report, err
		}
	}
	report.Applied, err = walk(ctx, pool, box, options, true)
	if err != nil {
		return report, err
	}
	report.Stage = "residual_scan"
	report.After, err = walk(ctx, pool, box, options, false)
	if err != nil {
		return report, err
	}
	if report.Applied.Invalid != 0 || report.After.Invalid != 0 || report.Applied.Conflicts != 0 || report.Before.Scanned != report.After.Scanned || report.Before.Scanned != report.Applied.Scanned {
		return report, errors.New("administrator phone conflicts or authentication failures remain; investigate and rescan")
	}
	report.SelectedScopeClean = report.After.Legacy == 0
	if options.Rollback {
		report.SelectedScopeClean = report.After.Protected == 0
	}
	if !report.SelectedScopeClean {
		return report, errors.New("administrator phone residuals remain; a fresh scan is required")
	}
	report.Stage = "complete"
	return report, nil
}

func walk(ctx context.Context, pool *sql.DB, box *securestore.Box, options Options, apply bool) (counts Counts, err error) {
	var high sql.NullInt64
	if err := pool.QueryRowContext(ctx, "SELECT MAX(user_id) FROM users").Scan(&high); err != nil {
		return counts, errors.New("cannot read administrator phone high-water mark")
	}
	if !high.Valid {
		return counts, nil
	}
	var last int64
	first := true
	for {
		page, err := readPage(ctx, pool, first, last, high.Int64, options.BatchSize)
		if err != nil {
			return counts, err
		}
		if len(page) == 0 {
			return counts, nil
		}
		for _, row := range page {
			if counts.Scanned >= options.MaxRows {
				return counts, errors.New("administrator phone row budget exceeded; scan is incomplete")
			}
			if err := ctx.Err(); err != nil {
				return counts, errors.New("administrator phone migration cancelled or deadline exceeded")
			}
			counts.Scanned++
			if row.Protected {
				counts.Protected++
			} else {
				counts.Legacy++
			}
			phone, openErr := row.Open(box, row.id)
			if row.oversized || row.id <= 0 || openErr != nil || (phone != nil && !auth.ValidPhone(*phone)) {
				counts.Invalid++
				continue
			}
			if phone == nil {
				counts.Empty++
			}
			if !apply || (options.Rollback && !row.Protected) || (!options.Rollback && row.Protected && activeEnvelope(box, row.Cipher)) {
				continue
			}
			next := Record{Legacy: phone}
			if !options.Rollback {
				next, err = Seal(box, row.id, phone)
				if err != nil {
					return counts, err
				}
			}
			changed, err := compareAndSwap(ctx, pool, row, next)
			if err != nil {
				return counts, err
			}
			if changed {
				counts.Rewritten++
			} else {
				counts.Conflicts++
			}
		}
		last, first = page[len(page)-1].id, false
		if options.Pause > 0 {
			timer := time.NewTimer(options.Pause)
			select {
			case <-ctx.Done():
				timer.Stop()
				return counts, errors.New("administrator phone migration cancelled or deadline exceeded")
			case <-timer.C:
			}
		}
	}
}

func activeEnvelope(box *securestore.Box, cipher *string) bool {
	if cipher == nil {
		return true
	}
	if id := box.ActiveKeyID(); id != "" {
		return strings.HasPrefix(*cipher, securestore.Prefix+"v2:"+id+":")
	}
	return !strings.HasPrefix(*cipher, securestore.Prefix+"v2:")
}

func readPage(ctx context.Context, pool *sql.DB, first bool, last, high int64, batch int) (page []candidate, err error) {
	rows, err := pool.QueryContext(ctx, `SELECT user_id,user_phone,
 CASE WHEN OCTET_LENGTH(user_phone_cipher)<=4096 THEN user_phone_cipher ELSE NULL END,
 user_phone_lookup,user_phone_protected,COALESCE(OCTET_LENGTH(user_phone_cipher)>4096,false)
 FROM users WHERE (? OR user_id>?) AND user_id<=? ORDER BY user_id LIMIT ?`, first, last, high, batch)
	if err != nil {
		return nil, errors.New("cannot read administrator phone page")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			page, err = nil, errors.New("cannot close administrator phone page")
		}
	}()
	for rows.Next() {
		var row candidate
		if err := rows.Scan(&row.id, &row.Legacy, &row.Cipher, &row.Lookup, &row.Protected, &row.oversized); err != nil {
			return nil, errors.New("cannot decode administrator phone page")
		}
		page = append(page, row)
	}
	if rows.Err() != nil {
		return nil, errors.New("cannot finish administrator phone page")
	}
	return page, nil
}

func compareAndSwap(ctx context.Context, pool *sql.DB, old candidate, next Record) (bool, error) {
	result, err := pool.ExecContext(ctx, `UPDATE users SET user_phone=?,user_phone_cipher=?,user_phone_lookup=?,user_phone_protected=?
 WHERE user_id=? AND user_phone_protected=?
 AND CAST(user_phone AS BINARY)<=>CAST(? AS BINARY)
 AND CAST(user_phone_cipher AS BINARY)<=>CAST(? AS BINARY)
 AND CAST(user_phone_lookup AS BINARY)<=>CAST(? AS BINARY)`, next.Legacy, next.Cipher, next.Lookup, next.Protected, old.id, old.Protected, old.Legacy, old.Cipher, old.Lookup)
	if err != nil {
		return false, errors.New("cannot commit administrator phone CAS; investigate constraints and retry from scan")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, errors.New("cannot verify administrator phone CAS result")
	}
	return count == 1, nil
}
