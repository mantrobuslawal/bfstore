package database

import (
	"database/sql"
	"testing"
)

func TestInstrumentedMySQLDriverReturnsDriverName(t *testing.T) {
	t.Parallel()

	driverName, err := instrumentedMySQLDriver()
	if err != nil {
		t.Fatalf("instrumentedMySQLDriver() error = %v, want nil", err)
	}

	if driverName == "" {
		t.Fatal("driverName = empty string, want non-empty")
	}

	if driverName == baseMySQLDriverName {
		t.Fatalf("driverName = %q, want instrumented driver name", driverName)
	}
}

func TestInstrumentedMySQLDriverCanBeCalledRepeatedly(t *testing.T) {
	t.Parallel()

	firstDriverName, err := instrumentedMySQLDriver()
	if err != nil {
		t.Fatalf("first instrumentedMySQLDriver() error = %v, want nil", err)
	}

	secondDriverName, err := instrumentedMySQLDriver()
	if err != nil {
		t.Fatalf("second instrumentedMySQLDriver() error = %v, want nil", err)
	}

	if firstDriverName != secondDriverName {
		t.Fatalf("driver names differ: first = %q, second = %q", firstDriverName, secondDriverName)
	}
}

func TestSQLCanOpenWithInstrumentedMySQLDriver(t *testing.T) {
	t.Parallel()

	driverName, err := instrumentedMySQLDriver()
	if err != nil {
		t.Fatalf("instrumentedMySQLDriver() error = %v, want nil", err)
	}

	db, err := sql.Open(driverName, "root:root@tcp(127.0.0.1:3306)/bfstore_catalog?parseTime=true")
	if err != nil {
		t.Fatalf("sql.Open() error = %v, want nil", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("sql.Open() db = nil, want non-nil")
	}
}
