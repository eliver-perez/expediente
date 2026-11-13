package storage

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestManagedPostgresLifecycleAndCrashRecovery(t *testing.T) {
	t.Setenv("AIDEX_TEST_POSTGRES_DSN", "")
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	database, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	var privileged bool
	if err = database.Reader.QueryRow("SELECT rolsuper OR rolcreatedb OR rolcreaterole FROM pg_roles WHERE rolname=current_user").Scan(&privileged); err != nil || privileged {
		database.Close()
		t.Fatal("runtime role is privileged", err)
	}
	if _, err = database.Writer.Exec("INSERT INTO roles VALUES('persistent','Persistent','global')"); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err = database.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err = database.Reader.QueryRow("SELECT count(*) FROM roles WHERE id='persistent'").Scan(&count); err != nil || count != 1 {
		database.Close()
		t.Fatal("restart lost committed data", count, err)
	}
	binary, err := postgresBinary("pg_ctl")
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	if output, e := exec.Command(binary, "-D", filepath.Join(directory, "postgresql", "data"), "-m", "immediate", "-w", "stop").CombinedOutput(); e != nil {
		database.Close()
		t.Fatal(e, string(output))
	}
	_ = database.Close()
	database, err = Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err = database.Reader.QueryRow("SELECT count(*) FROM roles WHERE id='persistent'").Scan(&count); err != nil || count != 1 {
		t.Fatal("crash recovery lost committed data", count, err)
	}
	if _, err = os.Stat(filepath.Join(directory, "documental.db")); !os.IsNotExist(err) {
		t.Fatal("unexpected legacy database")
	}
}

func TestManagedPostgresRecoversWithOpenPools(t *testing.T) {
	t.Setenv("AIDEX_TEST_POSTGRES_DSN", "")
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err = database.Writer.Exec("INSERT INTO roles VALUES('committed','Committed','global')"); err != nil {
		t.Fatal(err)
	}
	tx, err := database.Writer.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO roles VALUES('interrupted','Interrupted','global')"); err != nil {
		t.Fatal(err)
	}
	binary, err := postgresBinary("pg_ctl")
	if err != nil {
		t.Fatal(err)
	}
	if output, e := exec.Command(binary, "-D", filepath.Join(directory, "postgresql", "data"), "-m", "immediate", "-w", "stop").CombinedOutput(); e != nil {
		t.Fatal(e, string(output))
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		var count int
		probeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = database.Reader.QueryRowContext(probeCtx, "SELECT count(*) FROM roles WHERE id='committed'").Scan(&count)
		cancel()
		if err == nil && count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("private engine did not recover", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	var count int
	if err = database.Reader.QueryRow("SELECT count(*) FROM roles WHERE id='interrupted'").Scan(&count); err != nil || count != 0 {
		t.Fatal("interrupted transaction survived", count, err)
	}
}
