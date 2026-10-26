package sqlite

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

const (
	makiPreviousMigration int64 = 20261024194546592
	makiMigration         int64 = 20261026130000000
)

// TestMigration20261026130000000AllowsMakiAndReverses mirrors the Hermes
// coverage: Maki must be insertable after the migration, an unknown harness
// must still be rejected, rollback must refuse while Maki rows exist, and the
// down migration must restore the exact prior constraint.
func TestMigration20261026130000000AllowsMakiAndReverses(t *testing.T) {
	db := openMigratedDatabaseCopy(t, makiPreviousMigration)
	upTo(t, db, makiPreviousMigration)
	var before string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE name = 'sessions'`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `INSERT INTO projects (id, path, registered_at) VALUES ('maki-project', '/tmp/maki-project', CURRENT_TIMESTAMP)`)
	insert := `INSERT INTO sessions (id, project_id, num, harness, created_at, updated_at, activity_last_at) VALUES (?, 'maki-project', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	mustExec(t, db, insert, "existing-hermes", 1, "hermes")
	upTo(t, db, makiMigration)
	if _, err := db.Exec(insert, "maki-session", 2, "maki"); err != nil {
		t.Fatalf("insert maki session after migration: %v", err)
	}
	if _, err := db.Exec(insert, "unknown", 3, "unknown-agent"); err == nil {
		t.Fatal("unknown harness bypassed the CHECK constraint")
	}
	gooseMu.Lock()
	rollbackErr := goose.DownTo(db, "migrations", makiPreviousMigration)
	gooseMu.Unlock()
	if rollbackErr == nil {
		t.Fatal("rollback accepted existing maki session")
	}
	mustExec(t, db, `UPDATE sessions SET harness = '' WHERE harness = 'maki'`)
	downTo(t, db, makiPreviousMigration)
	var after string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE name = 'sessions'`).Scan(&after); err != nil || after != before {
		t.Fatalf("down migration did not restore original schema: %v", err)
	}
	if _, err := db.Exec(insert, "maki-after-down", 4, "maki"); err == nil || !strings.Contains(err.Error(), "CHECK") {
		t.Fatalf("maki insertion after downgrade = %v; want CHECK failure", err)
	}
	var integrity string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity after downgrade = %q, %v", integrity, err)
	}
}

// TestMakiLegacyRepairKeepsEveryHarness covers the AO startup repair on a
// legacy schema that already carries Maki: the exact historical repairs must
// still match, and Maki must survive exactly once.
func TestMakiLegacyRepairKeepsEveryHarness(t *testing.T) {
	db := openMigratedDatabaseCopy(t, makiMigration)
	upTo(t, db, makiMigration)
	mustExec(t, db, `PRAGMA writable_schema = ON`)
	mustExec(t, db, `UPDATE sqlite_master SET sql = replace(sql, '''command-code'', ', '') WHERE type = 'table' AND name = 'sessions'`)
	mustExec(t, db, `PRAGMA writable_schema = RESET`)
	if err := reconcileHarnessConstraint(db); err != nil {
		t.Fatalf("legacy repair with Maki present: %v", err)
	}
	var schema string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'sessions'`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	for _, harness := range []string{"'maki'", "'hermes'", "'command-code'"} {
		if strings.Count(schema, harness) != 1 {
			t.Fatalf("legacy repair lost or duplicated %s: %s", harness, schema)
		}
	}
	if err := reconcileHarnessConstraint(db); err != nil {
		t.Fatalf("repeated repair: %v", err)
	}
}
