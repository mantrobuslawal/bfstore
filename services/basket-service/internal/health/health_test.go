package health

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckerLive(t *testing.T) {
	t.Parallel()

	checker := NewChecker(nil)
	if err := checker.Live(context.Background()); err != nil {
		t.Fatalf("Live() error = %v, want nil", err)
	}
}

func TestCheckerReady(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v, want nil", err)
	}
	defer db.Close()

	mock.ExpectPing()

	checker := NewChecker(db)
	if err := checker.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v, want nil", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
