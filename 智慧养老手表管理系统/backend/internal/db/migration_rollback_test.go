package db

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	rollbackRefPattern  = regexp.MustCompile(`(?i)DELETE\s+FROM\s+schema_migrations\s+WHERE\s+name\s*=\s*'([^']+)'`)
	migrationNumPattern = regexp.MustCompile(`^(\d{4})_`)
)

// TestMigrationRollbackPairs keeps forward migrations and rollback scripts in
// sync using the filesystem as the only source of truth. No migration name
// list and no migration count is hardcoded here.
func TestMigrationRollbackPairs(t *testing.T) {
	mysqlNames, err := MySQLMigrationNames()
	if err != nil {
		t.Fatal(err)
	}
	pgNames, err := LegacyMigrationNames()
	if err != nil {
		t.Fatal(err)
	}
	forward := map[string]map[string]bool{
		"mysql":    nameSet(mysqlNames),
		"postgres": nameSet(pgNames),
	}
	covered := map[string]map[string]bool{
		"mysql":    {},
		"postgres": {},
	}
	dir := filepath.Join("..", "..", "migrations", "rollback")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		fileName := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(fileName, ".sql") {
			continue
		}
		var dialect, expectedPrefix string
		switch {
		case strings.HasPrefix(fileName, "mysql_"):
			dialect, expectedPrefix = "mysql", "mysql_"
		case strings.HasPrefix(fileName, "postgres_"):
			dialect, expectedPrefix = "postgres", "postgres_"
		default:
			t.Errorf("rollback script %q has no mysql_ or postgres_ prefix", fileName)
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, fileName))
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		// Only MySQL rollback scripts are replayed by the statement splitter in
		// MigrateMySQL and its rollback test; PostgreSQL applies whole files.
		if dialect == "mysql" {
			for _, line := range strings.Split(body, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "--") && strings.Contains(trimmed, ";") {
					t.Errorf("%s: a semicolon inside a comment is split into a statement by MigrateMySQL", fileName)
				}
			}
		}
		matches := rollbackRefPattern.FindAllStringSubmatch(body, -1)
		if len(matches) != 1 {
			t.Errorf("%s: expected exactly one schema_migrations delete, found %d", fileName, len(matches))
			continue
		}
		ref := matches[0][1]
		if !forward[dialect][ref] {
			t.Errorf("%s references unknown %s migration %q", fileName, dialect, ref)
			continue
		}
		expected := expectedPrefix + path.Base(ref)
		if fileName != expected {
			t.Errorf("%s should be named %s to match %q", fileName, expected, ref)
		}
		if covered[dialect][ref] {
			t.Errorf("%s duplicates the rollback for %q", fileName, ref)
		}
		covered[dialect][ref] = true
	}

	// MySQL is the operational primary: every forward migration between the
	// oldest and newest covered numbers must have its own rollback script.
	// Both bounds are derived from the filesystem, never written as constants.
	numbers := make([]int, 0, len(covered["mysql"]))
	for ref := range covered["mysql"] {
		n, ok := migrationNumber(path.Base(ref))
		if !ok {
			t.Fatalf("covered migration %q has no numeric prefix", ref)
		}
		numbers = append(numbers, n)
	}
	if len(numbers) == 0 {
		t.Fatal("no MySQL rollback scripts found")
	}
	sort.Ints(numbers)
	lowest := numbers[0]
	// The upper bound comes from forward migrations, not from rollback scripts:
	// deleting the newest rollback must not silently shrink this gate.
	highest := 0
	for _, name := range mysqlNames {
		if n, ok := migrationNumber(path.Base(name)); ok && n > highest {
			highest = n
		}
	}
	for _, name := range mysqlNames {
		n, ok := migrationNumber(path.Base(name))
		if !ok || n < lowest || n > highest {
			continue
		}
		if !covered["mysql"][name] {
			t.Errorf("MySQL migration %s has no rollback script", name)
		}
	}
}

func nameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

func migrationNumber(fileName string) (int, bool) {
	m := migrationNumPattern.FindStringSubmatch(fileName)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}
