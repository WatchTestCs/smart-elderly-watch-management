// Command migrate 仅执行数据库迁移后退出（供 CI 验证迁移脚本）。
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"careold-go/internal/db"
)

func main() {
	legacy := flag.Bool("legacy-postgres", false, "run only the legacy PostgreSQL migrations")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if *legacy {
		pool, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatal("legacy PostgreSQL connection failed")
		}
		defer pool.Close()
		if err = db.Migrate(ctx, pool); err != nil {
			log.Fatal(err)
		}
	} else {
		pool, err := db.OpenMySQL(ctx, os.Getenv("MYSQL_DSN"))
		if err != nil {
			log.Fatal("MySQL connection failed: check MYSQL_DSN")
		}
		defer pool.Close()
		if err = db.MigrateMySQL(ctx, pool); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("migrations applied")
}
