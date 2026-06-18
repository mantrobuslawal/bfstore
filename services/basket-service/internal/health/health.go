package health

import (
	"context"
	"database/sql"
	"time"
)

// Checker performs basic service health checks.
type Checker struct {
	db *sql.DB
}

// NewChecker creates a health checker.
func NewChecker(db *sql.DB) *Checker {
	return &Checker{db: db}
}

// Ready checks whether the service is ready to handle traffic.
func (c *Checker) Ready(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return c.db.PingContext(checkCtx)
}

// Live checks whether the process is alive.
//
// This is deliberately simple at this stage. More checks can be added later.
func (c *Checker) Live(_ context.Context) error {
	return nil
}
