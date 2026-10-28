// Command protect-notes previews or applies field encryption/decryption offline.
// Stop all writers and back up the database and original key before --apply.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"suining-care/internal/config"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

func main() {
	apply := flag.Bool("apply", false, "apply after backup with every writer stopped")
	decrypt := flag.Bool("decrypt", false, "restore plaintext notes for old binaries")
	flag.Parse()
	cfg := config.Load()
	box, err := securestore.New(cfg.DataKey)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.OpenMySQL(ctx, cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	for _, target := range []struct{ table, key, column string }{{"care_incident", "id", "resolution_note"}, {"care_incident_event", "id", "note"}, {"alert_review", "id", "reason"}} {
		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatal(err)
		}
		rows, err := tx.Query(ctx, "SELECT "+target.key+","+target.column+" FROM "+target.table+" WHERE "+target.column+" IS NOT NULL ORDER BY "+target.key+" FOR UPDATE")
		if err != nil {
			tx.Rollback(ctx)
			log.Fatal(err)
		}
		type item struct {
			id   int64
			text string
		}
		items := []item{}
		for rows.Next() {
			var value item
			if err = rows.Scan(&value.id, &value.text); err != nil {
				break
			}
			if strings.HasPrefix(value.text, securestore.Prefix) == *decrypt {
				items = append(items, value)
			}
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			tx.Rollback(ctx)
			log.Fatal(err)
		}
		fmt.Printf("%s.%s candidates=%d apply=%t decrypt=%t\n", target.table, target.column, len(items), *apply, *decrypt)
		if *apply {
			for _, value := range items {
				var text string
				if *decrypt {
					text, err = box.Open(value.text, "care-text")
				} else {
					text, err = box.Seal(value.text, "care-text")
				}
				if err == nil {
					_, err = tx.Exec(ctx, "UPDATE "+target.table+" SET "+target.column+"=$2 WHERE "+target.key+"=$1", value.id, text)
				}
				if err != nil {
					tx.Rollback(ctx)
					log.Fatal("field conversion failed; current table rolled back")
				}
			}
			if err = tx.Commit(ctx); err != nil {
				log.Fatal(err)
			}
		} else {
			tx.Rollback(ctx)
		}
	}
}
