package dbtest

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"suining-care/internal/db"
)

var disposableSchema = regexp.MustCompile(`^suining_(test|restore)_[0-9a-f]{8}$`)
var fixtureContainer = regexp.MustCompile(`^[0-9a-f]{12,64}$`)

func fixtureIdentity(t testing.TB, ctx context.Context, pool *db.Database) (name, server string) {
	t.Helper()
	if err := pool.QueryRowContext(ctx, "SELECT DATABASE(),@@server_uuid").Scan(&name, &server); err != nil {
		t.Fatal(err)
	}
	if !disposableSchema.MatchString(name) {
		t.Fatal("expected a disposable MySQL fixture schema")
	}
	return
}

// mysqlClient uses the same client transport as the coordinated restore drill.
// Verify the server identity before issuing a dump or importing into a fixture.
func mysqlClient(t testing.TB, ctx context.Context, server string) func(string, []string, []byte) []byte {
	t.Helper()
	cfg, err := mysql.ParseDSN(os.Getenv("TEST_MYSQL_DSN"))
	if err != nil || cfg.Net != "tcp" {
		t.Fatal("MySQL fixture snapshot requires a dedicated TCP test DSN")
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		t.Fatal(err)
	}
	container := os.Getenv("TEST_MYSQL_CONTAINER_ID")
	if container != "" && !fixtureContainer.MatchString(container) {
		t.Fatal("expected an isolated MySQL fixture container ID")
	}
	client := func(tool string, args []string, input []byte) []byte {
		t.Helper()
		program := tool
		invocation := []string{"--no-defaults", "--protocol=TCP", "--host=" + host, "--port=" + port, "--user=" + cfg.User}
		if container != "" {
			program = "docker"
			invocation = []string{"exec", "-i", "-e", "MYSQL_PWD", container, tool,
				"--no-defaults", "--protocol=TCP", "--host=127.0.0.1", "--port=3306", "--user=" + cfg.User}
		}
		binary, err := exec.LookPath(program)
		if err != nil {
			t.Fatal("fixture snapshot requires MySQL client tools or the isolated fixture container")
		}
		cli := exec.CommandContext(ctx, binary, append(invocation, args...)...)
		cli.Env = append(os.Environ(), "MYSQL_PWD="+cfg.Passwd)
		cli.Stdin = bytes.NewReader(input)
		output, err := cli.Output()
		if err != nil {
			t.Fatalf("%s failed with SQL output withheld: %v", tool, err)
		}
		return output
	}
	if got := client("mysql", []string{"--batch", "--skip-column-names", "-e", "SELECT @@server_uuid"}, nil); strings.TrimSpace(string(got)) != server {
		t.Fatal("MySQL client is not connected to the fixture server")
	}
	return client
}

func DumpMySQL(t testing.TB, ctx context.Context, pool *db.Database) []byte {
	t.Helper()
	name, server := fixtureIdentity(t, ctx, pool)
	client := mysqlClient(t, ctx, server)
	archive := client("mysqldump", []string{"--single-transaction", "--quick", "--routines", "--triggers", "--hex-blob",
		"--no-tablespaces", "--set-gtid-purged=OFF", "--column-statistics=0", name}, nil)
	if len(archive) == 0 {
		t.Fatal("fixture dump did not capture SQL")
	}
	return archive
}

func RestoreMySQL(t testing.TB, ctx context.Context, pool *db.Database, archive []byte) {
	t.Helper()
	name, server := fixtureIdentity(t, ctx, pool)
	var tables int
	if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatal("refusing to restore into a nonempty fixture schema")
	}
	mysqlClient(t, ctx, server)("mysql", []string{name}, archive)
}

var fixtureSnapshot struct {
	sync.Mutex
	dsn, container, server string
	archive                []byte
}

// Capture only the pristine, migrated database, before returning it to a test.
// The archive is process-local and is never written to disk or shared across servers.
func snapshotArchive(t testing.TB, ctx context.Context, pool *db.Database) ([]byte, bool) {
	t.Helper()
	_, server := fixtureIdentity(t, ctx, pool)
	dsn, container := os.Getenv("TEST_MYSQL_DSN"), os.Getenv("TEST_MYSQL_CONTAINER_ID")
	fixtureSnapshot.Lock()
	defer fixtureSnapshot.Unlock()
	if fixtureSnapshot.archive != nil && fixtureSnapshot.dsn == dsn && fixtureSnapshot.container == container && fixtureSnapshot.server == server {
		return fixtureSnapshot.archive, false
	}
	if err := db.MigrateMySQL(ctx, pool); err != nil {
		t.Fatal(err)
	}
	archive := DumpMySQL(t, ctx, pool)
	fixtureSnapshot.dsn, fixtureSnapshot.container, fixtureSnapshot.server = dsn, container, server
	fixtureSnapshot.archive = archive
	return archive, true
}

// MySQLSnapshot is an opt-in alternative for ordinary integration fixtures.
// Each call still creates and drops a distinct schema. Migration-specific tests
// should use MySQL instead, so the historical migration path remains exercised.
func MySQLSnapshot(t testing.TB) *db.Database {
	t.Helper()
	pool := emptyMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	archive, populated := snapshotArchive(t, ctx, pool)
	if !populated {
		RestoreMySQL(t, ctx, pool, archive)
		names, err := db.MySQLMigrationNames()
		if err != nil {
			t.Fatal(err)
		}
		var completed, pending int
		if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE schema_checksum<>''").Scan(&completed); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migration_steps").Scan(&pending); err != nil {
			t.Fatal(err)
		}
		if completed != len(names) || pending != 0 {
			t.Fatalf("incomplete fixture migration ledger: completed=%d want=%d pending=%d", completed, len(names), pending)
		}
		// Recheck the ledger and full schema fingerprint using the production
		// migrator; a corrupted or outdated archive must fail the test.
		if err := db.MigrateMySQL(ctx, pool); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}
