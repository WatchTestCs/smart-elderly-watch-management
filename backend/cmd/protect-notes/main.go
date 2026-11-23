// Command protect-notes converts selected legacy note fields in bounded batches.
// Stop writers, back up and test restoration before --apply or --decrypt --apply.
package main

import (
    "context"
    "database/sql"
    "errors"
    "flag"
    "fmt"
    "io"
    "os"
    "strings"
    "time"

    "suining-care/internal/config"
    "suining-care/internal/db"
    "suining-care/internal/securestore"
)

func main() {
    if err := run(os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(args []string, output io.Writer) error {
    flags := flag.NewFlagSet("protect-notes", flag.ContinueOnError)
    flags.SetOutput(io.Discard)
    apply := flags.Bool("apply", false, "apply after testing a backup, with all writers stopped")
    decrypt := flags.Bool("decrypt", false, "restore plaintext notes for old binaries")
    limit := flags.Int("limit", 100, "maximum candidates per table per invocation, 1..1000")
    if err := flags.Parse(args); err != nil {
        if errors.Is(err, flag.ErrHelp) {
            flags.SetOutput(output)
            flags.PrintDefaults()
            return nil
        }
        return errors.New("invalid protect-notes arguments; use --help")
    }
    if flags.NArg() != 0 || *limit < 1 || *limit > 1000 {
        return errors.New("provide a limit between 1 and 1000 without extra arguments")
    }
    box, err := securestore.New(config.Secret("DATA_ENCRYPTION_KEY"))
    if err != nil || box == nil { return errors.New("valid service encryption key required for protect-notes") }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    pool, err := db.OpenMySQL(ctx, config.Secret("MYSQL_DSN"))
    if err != nil { return errors.New("cannot open explicitly configured MySQL") }
    defer pool.Close()
    targets := []struct{ table, key, column string }{
        {"care_incident","id","resolution_note"},
        {"care_incident_event","id","note"},
        {"alert_review","id","reason"},
    }
    for _, target := range targets {
        if err := processTable(ctx, pool.DB, box, target.table, target.key, target.column,
            *limit, *apply, *decrypt, output); err != nil { return err }
    }
    return nil
}

func processTable(ctx context.Context, database *sql.DB, box *securestore.Box,
    table, key, column string, limit int, apply, decrypt bool, output io.Writer) error {
    // This read is intentionally NOT a locking read: a full scan for legacy
    // plaintext can otherwise lock many unrelated rows before LIMIT is reached.
    filter := column + " IS NOT NULL AND " + column + " NOT LIKE ?"
    if decrypt { filter = column + " IS NOT NULL AND " + column + " LIKE ?" }
    query := "SELECT "+key+","+column+" FROM "+table+" WHERE "+filter+" ORDER BY "+key+" LIMIT ?"
    rows, err := database.QueryContext(ctx, query, securestore.Prefix+"%", limit)
    if err != nil { return err }
    type item struct { id int64; text string }
    items := make([]item, 0, limit)
    for rows.Next() {
        var v item
        if err = rows.Scan(&v.id,&v.text); err != nil { break }
        if strings.HasPrefix(v.text,securestore.Prefix) == decrypt { items = append(items,v) }
    }
    if err == nil { err = rows.Err() }
    rows.Close()
    if err != nil { return err }
    if apply && len(items) != 0 {
        tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
        if err != nil { return err }
        defer tx.Rollback()
        for _, v := range items {
            // Lock only the selected primary key; fail if an unexpected writer
            // changed a candidate between the initial scan and its conversion.
            var current string
            if err := tx.QueryRowContext(ctx, "SELECT "+column+" FROM "+table+
                " WHERE "+key+"=? FOR UPDATE",v.id).Scan(&current); err != nil { return err }
            if current != v.text {
                return fmt.Errorf("%s.%s changed during backfill; transaction rolled back",table,column)
            }
            var transformed string
            if decrypt { transformed,err = box.Open(current,"care-text") } else { transformed,err = box.Seal(current,"care-text") }
            if err != nil { return fmt.Errorf("%s.%s conversion failed; transaction rolled back",table,column) }
            if _,err = tx.ExecContext(ctx,"UPDATE "+table+" SET "+column+"=? WHERE "+key+"=?",transformed,v.id); err != nil {
                return fmt.Errorf("%s.%s update failed; transaction rolled back",table,column)
            }
        }
        if err := tx.Commit(); err != nil { return err }
    }
    _,err = fmt.Fprintf(output,"%s.%s candidates=%d limit=%d apply=%t decrypt=%t\n",table,column,len(items),limit,apply,decrypt)
    return err
}
