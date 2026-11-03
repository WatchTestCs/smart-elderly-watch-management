package userphone

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
	"suining-care/internal/securestore"
)

func TestAdministratorPhoneMySQL(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	a, b, retired := testBox(t, "A", false), testBox(t, "B", true), testBox(t, "B", false)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.SQL().ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO users(user_id,user_name,user_phone,role) VALUES(1,'first','13800000001','1'),(2,'empty',NULL,'1'),(3,'invalid','not-a-phone','1')")
	// Concurrent first writers with different lookup keys cannot overwrite one
	// another's immutable binding, even while all account rows are still legacy.
	other, err := securestore.NewKeyring("A", map[string]string{"A": testKey(1)}, testKey(4))
	if err != nil {
		t.Fatal(err)
	}
	type bindingResult struct {
		box *securestore.Box
		err error
	}
	results := make(chan bindingResult, 2)
	start := make(chan struct{})
	for _, box := range []*securestore.Box{a, other} {
		go func() {
			<-start
			tx, err := pool.Begin(ctx)
			if err != nil {
				results <- bindingResult{box, err}
				return
			}
			err = BindLookupKey(ctx, tx, box)
			if err == nil {
				err = tx.Commit(ctx)
			}
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				results <- bindingResult{box, rollbackErr}
				return
			}
			results <- bindingResult{box, err}
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		result := <-results
		_, searchErr := SearchIndex(ctx, pool, result.box, "13800000001")
		if result.err == nil {
			successes++
			if searchErr != nil {
				t.Fatal("winner's lookup binding was overwritten", searchErr)
			}
		} else if searchErr == nil {
			t.Fatal("losing key could search a differently bound account database")
		}
	}
	if successes != 1 {
		t.Fatal("concurrent lookup-key initialization did not have exactly one winner", successes)
	}
	// Reset only this test fixture's verifier before testing the A→B scenario.
	exec("DELETE FROM account_phone_key_state WHERE singleton=1")
	options := Options{Apply: true, BatchSize: 1, MaxRows: 100}
	if report, err := Run(ctx, pool.SQL(), a, options); err == nil || report.Before.Invalid != 1 || report.Applied.Rewritten != 0 {
		t.Fatal("late invalid plaintext did not block every mutation", report, err)
	}
	var plaintext int64
	if err := pool.SQL().QueryRowContext(ctx, "SELECT count(*) FROM users WHERE user_phone_protected=false").Scan(&plaintext); err != nil || plaintext != 3 {
		t.Fatal("preflight partially changed data", err)
	}
	exec("UPDATE users SET user_phone='13800000003' WHERE user_id=3")
	for _, bad := range []Options{{BatchSize: 0, MaxRows: 10}, {BatchSize: 1001, MaxRows: 10}, {BatchSize: 1, MaxRows: 0}, {BatchSize: 1, MaxRows: 10, Pause: -1}} {
		if _, err := Run(ctx, pool.SQL(), a, bad); err == nil {
			t.Fatal("invalid bound accepted")
		}
	}
	if _, err := Run(context.Background(), pool.SQL(), a, options); err == nil {
		t.Fatal("unbounded operation accepted")
	}
	budget := options
	budget.MaxRows = 2
	if report, err := Run(ctx, pool.SQL(), a, budget); err == nil || report.Applied.Rewritten != 0 || report.SelectedScopeClean {
		t.Fatal("row budget returned success or wrote before complete preflight", report, err)
	}
	scan := options
	scan.Apply = false
	if report, err := Run(ctx, pool.SQL(), a, scan); err != nil || report.Before.Legacy != 3 || report.SelectedScopeClean || report.Applied.Rewritten != 0 {
		t.Fatal("default scan mutated data or hid legacy records", report, err)
	}
	report, err := Run(ctx, pool.SQL(), a, options)
	if err != nil || report.Applied.Rewritten != 3 || report.After.Scanned != 3 || !report.SelectedScopeClean || report.RetirementAuthorized {
		t.Fatal("protect failed", report, err)
	}
	read := func(id int64) candidate {
		t.Helper()
		row := candidate{id: id}
		if err := pool.SQL().QueryRowContext(ctx, "SELECT user_phone,user_phone_cipher,user_phone_lookup,user_phone_protected FROM users WHERE user_id=?", id).Scan(&row.Legacy, &row.Cipher, &row.Lookup, &row.Protected); err != nil {
			t.Fatal(err)
		}
		return row
	}
	first := read(1)
	if first.Legacy != nil || first.Cipher == nil || strings.Contains(*first.Cipher, "13800000001") {
		t.Fatal("plaintext remained in selected storage")
	}
	if _, err := SearchIndex(ctx, pool, nil, "13800000001"); err == nil {
		t.Fatal("missing key silently hid protected accounts from duplicate checks")
	}
	if second, err := Run(ctx, pool.SQL(), a, options); err != nil || second.Applied.Rewritten != 0 || *read(1).Cipher != *first.Cipher {
		t.Fatal("protect was not idempotent", err)
	}
	wrongLookup, err := securestore.NewKeyring("A", map[string]string{"A": testKey(1)}, testKey(4))
	if err != nil {
		t.Fatal(err)
	}
	if report, err := Run(ctx, pool.SQL(), wrongLookup, options); err == nil || report.Applied.Rewritten != 0 || report.SelectedScopeClean {
		t.Fatal("lookup key switch bypassed preflight", report, err)
	}
	if report, err := Run(ctx, pool.SQL(), b, options); err != nil || report.Applied.Rewritten != 2 || *read(1).Lookup != *first.Lookup {
		t.Fatal("key rotation changed deterministic index or failed", report, err)
	}
	rotated := read(1)
	if phone, err := rotated.Open(retired, 1); err != nil || phone == nil || *phone != "13800000001" {
		t.Fatal("rotated value requires retired encryption key", err)
	}
	for _, changedCipher := range []string{*rotated.Cipher + " ", strings.ToLower(*rotated.Cipher)} {
		exec("UPDATE users SET user_phone_cipher=? WHERE user_id=1", changedCipher)
		if changed, err := compareAndSwap(ctx, pool.SQL(), rotated, Record{}); err != nil || changed || *read(1).Cipher != changedCipher {
			t.Fatal("CAS ignored binary case/trailing-space changes", err)
		}
	}
	exec("UPDATE users SET user_phone_cipher=? WHERE user_id=1", rotated.Cipher)
	phone := "13800000009"
	newer, err := Seal(b, 1, &phone)
	if err != nil {
		t.Fatal(err)
	}
	exec("UPDATE users SET user_phone_cipher=?,user_phone_lookup=? WHERE user_id=1", newer.Cipher, newer.Lookup)
	if changed, err := compareAndSwap(ctx, pool.SQL(), rotated, Record{Legacy: &phone}); err != nil || changed || *read(1).Cipher != *newer.Cipher {
		t.Fatal("CAS overwrote a concurrent writer", err)
	}
	deleted := read(3)
	exec("DELETE FROM users WHERE user_id=3")
	if changed, err := compareAndSwap(ctx, pool.SQL(), deleted, Record{Legacy: &phone}); err != nil || changed {
		t.Fatal("CAS recreated or changed a deleted row", err)
	}
	// Schema guards prevent an old plaintext writer from silently downgrading a
	// protected account, and the lookup's unique constraint survives encryption.
	if _, err := pool.SQL().ExecContext(ctx, "UPDATE users SET user_phone='13800000009' WHERE user_id=1"); err == nil {
		t.Fatal("protected row accepted an old plaintext write")
	}
	if _, err := pool.SQL().ExecContext(ctx, "INSERT INTO users(user_id,user_phone_cipher,user_phone_lookup,user_phone_protected) VALUES(4,?,?,true)", newer.Cipher, newer.Lookup); err == nil {
		t.Fatal("duplicate lookup index was accepted")
	}
	// Authentication failure, oversize and cancellation all leave plaintext
	// rollback disabled; no raw values appear in the aggregate report.
	exec("UPDATE users SET user_phone_cipher=? WHERE user_id=1", strings.Repeat("x", 5000))
	rollback := options
	rollback.Rollback = true
	if report, err := Run(ctx, pool.SQL(), b, rollback); err == nil || report.Before.Invalid != 1 || report.Applied.Rewritten != 0 {
		t.Fatal("oversized corrupt ciphertext did not block rollback", report, err)
	}
	exec("UPDATE users SET user_phone_cipher=? WHERE user_id=1", newer.Cipher)
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if report, err := Run(cancelled, pool.SQL(), b, rollback); err == nil || report.SelectedScopeClean {
		t.Fatal("cancelled operation returned success", report, err)
	}
	if report, err := Run(ctx, pool.SQL(), retired, rollback); err != nil || report.Applied.Rewritten != 2 || report.After.Protected != 0 || !report.SelectedScopeClean {
		t.Fatal("controlled rollback did not restore original storage", report, err)
	}
	if restored := read(1); restored.Protected || restored.Legacy == nil || *restored.Legacy != phone || restored.Cipher != nil || restored.Lookup != nil {
		t.Fatal("rollback changed value or retained duplicate sensitive storage")
	}
	if empty := read(2); empty.Legacy != nil || empty.Protected {
		t.Fatal("rollback changed NULL into an empty string")
	}
	if _, err := SearchIndex(ctx, pool, nil, phone); err != nil {
		t.Fatal("legacy-only fallback was not restored", err)
	}
	if report, err := Run(ctx, pool.SQL(), retired, options); err != nil || report.Applied.Rewritten != 2 {
		t.Fatal("reapply after rollback failed", report, err)
	}
	encoded, err := json.Marshal(report)
	if err != nil || strings.Contains(string(encoded), "138000000") || strings.Contains(string(encoded), *first.Cipher) || strings.Contains(string(encoded), *first.Lookup) {
		t.Fatal("aggregate report leaked values", err)
	}
}
