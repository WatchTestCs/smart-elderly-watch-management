// Command transfer-postgres imports an offline legacy PG database into empty MySQL.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"suining-care/internal/db"
)

func main() {
	apply := flag.Bool("apply", false, "commit import; both application writers must be stopped")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	source, err := db.Open(ctx, os.Getenv("LEGACY_POSTGRES_URL"))
	if err != nil {
		log.Fatal("legacy PostgreSQL connection failed")
	}
	defer source.Close()
	target, err := db.OpenMySQL(ctx, os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatal("MySQL connection failed")
	}
	defer target.Close()
	if err = db.MigrateMySQL(ctx, target); err != nil {
		log.Fatal(err)
	}
	counts, err := db.TransferLegacy(ctx, source, target, *apply)
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"applied": *apply, "rows": counts})
}
