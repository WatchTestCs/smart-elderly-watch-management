// care-supervisor is a local, explicit recovery command for installations
// migrated from implicit administrator privileges.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"suining-care/internal/carerepo"
)

func main() {
	adminID := flag.Int64("admin-id", 0, "existing administrator user_id to make the first supervisor")
	flag.Parse()
	dsn := os.Getenv("MYSQL_DSN")
	if *adminID <= 0 || dsn == "" || flag.NArg() != 0 {
		log.Fatal("usage: MYSQL_DSN=<secret> care-supervisor --admin-id=<existing admin ID>")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()
	if err := carerepo.LockAuthority(ctx, tx); err != nil {
		log.Fatal(err)
	}
	var existing int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM care_staff_permission p JOIN users u ON u.user_id=p.admin_id WHERE p.duty='supervisor' AND u.role='1' AND u.account_status='active'").Scan(&existing); err != nil {
		log.Fatal(err)
	}
	if existing != 0 {
		log.Fatal("a supervisor already exists; use authenticated duty assignment instead")
	}
	exists, err := carerepo.AccountExists(ctx, tx, *adminID)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Fatal("target is not an existing administrator")
	}
	if err := carerepo.SetDuty(ctx, tx, *adminID, *adminID, "supervisor"); err != nil {
		log.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO care_access_audit(admin_id,action,subject_admin_id) VALUES(?,'supervisor_local_bootstrap',?)", *adminID, *adminID); err != nil {
		log.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("first care supervisor explicitly assigned to administrator %d\n", *adminID)
}
