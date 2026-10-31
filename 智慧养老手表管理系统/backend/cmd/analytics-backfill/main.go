package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"suining-care/internal/analytics"
	"suining-care/internal/db"
)

func main() {
	apply := flag.Bool("apply", false, "queue current MySQL rows to rebuild PostgreSQL projections")
	flag.Parse()
	if !*apply {
		log.Fatal("use --apply to queue a projection rebuild")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	primary, err := db.OpenMySQL(ctx, os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatal("MySQL connection failed")
	}
	defer primary.Close()
	n, err := analytics.Backfill(ctx, primary)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("queued %d analysis events", n)
}
