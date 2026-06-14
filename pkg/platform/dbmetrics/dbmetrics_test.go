package dbmetrics

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestRegisterRejectsNilDB(t *testing.T) {
	t.Parallel()

	err := Register(nil, Config{
		DBSystem: "mysql",
		DBName:   "bfstore_catalog",
	})

	if err == nil {
		t.Fatal("Register() error = nil, want non-nil")
	}
}

func TestRegisterAcceptsSQLDB(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/bfstore_catalog?parseTime=true")
	if err != nil {
		t.Fatalf("sql.Open() error = %v, want nil", err)
	}
	defer db.Close()

	err = Register(db, Config{
		MeterName: "github.com/mantrobuslawal/bfstore/pkg/platform/dbmetrics/test",
		DBSystem:  "mysql",
		DBName:    "bfstore_catalog",
	})

	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
}

func TestRegisterAppliesDefaults(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/bfstore_catalog?parseTime=true")
	if err != nil {
		t.Fatalf("sql.Open() error = %v, want nil", err)
	}
	defer db.Close()

	if err := Register(db, Config{}); err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
}
