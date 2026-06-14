package dbmetrics

import (
	"context"
	"database/sql"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const defaultMeterName = "github.com/mantrobuslawal/bfstore/pkg/platform/dbmetrics"

// Config describes a database pool metrics registration.
type Config struct {
	MeterName string
	DBSystem  string
	DBName    string
}

// Register registers observable database connection pool metrics for db.
func Register(db *sql.DB, cfg Config) error {
	if db == nil {
		return errors.New("dbmetrics: db is nil")
	}

	meterName := cfg.MeterName
	if meterName == "" {
		meterName = defaultMeterName
	}

	dbSystem := cfg.DBSystem
	if dbSystem == "" {
		dbSystem = "unknown"
	}

	attrs := []attribute.KeyValue{
		attribute.String("db.system", dbSystem),
	}

	if cfg.DBName != "" {
		attrs = append(attrs, attribute.String("db.name", cfg.DBName))
	}

	meter := otel.Meter(meterName)

	maxOpenConnections, err := meter.Int64ObservableGauge(
		"db.client.connections.max",
		metric.WithDescription("Maximum number of open connections configured for the database pool."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	openConnections, err := meter.Int64ObservableGauge(
		"db.client.connections.open",
		metric.WithDescription("Number of established database connections, both in use and idle."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	inUseConnections, err := meter.Int64ObservableGauge(
		"db.client.connections.in_use",
		metric.WithDescription("Number of database connections currently in use."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	idleConnections, err := meter.Int64ObservableGauge(
		"db.client.connections.idle",
		metric.WithDescription("Number of idle database connections."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	waitCount, err := meter.Int64ObservableGauge(
		"db.client.connections.wait_count",
		metric.WithDescription("Total number of times callers waited for a database connection."),
		metric.WithUnit("{wait}"),
	)
	if err != nil {
		return err
	}

	waitDuration, err := meter.Int64ObservableGauge(
		"db.client.connections.wait_duration",
		metric.WithDescription("Total time callers spent waiting for a database connection."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	maxIdleClosed, err := meter.Int64ObservableGauge(
		"db.client.connections.max_idle_closed",
		metric.WithDescription("Total number of database connections closed due to the max idle connections limit."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	maxIdleTimeClosed, err := meter.Int64ObservableGauge(
		"db.client.connections.max_idle_time_closed",
		metric.WithDescription("Total number of database connections closed due to the max idle time limit."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	maxLifetimeClosed, err := meter.Int64ObservableGauge(
		"db.client.connections.max_lifetime_closed",
		metric.WithDescription("Total number of database connections closed due to the max lifetime limit."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			stats := db.Stats()
			options := metric.WithAttributes(attrs...)

			observer.ObserveInt64(maxOpenConnections, int64(stats.MaxOpenConnections), options)
			observer.ObserveInt64(openConnections, int64(stats.OpenConnections), options)
			observer.ObserveInt64(inUseConnections, int64(stats.InUse), options)
			observer.ObserveInt64(idleConnections, int64(stats.Idle), options)
			observer.ObserveInt64(waitCount, stats.WaitCount, options)
			observer.ObserveInt64(waitDuration, stats.WaitDuration.Milliseconds(), options)
			observer.ObserveInt64(maxIdleClosed, stats.MaxIdleClosed, options)
			observer.ObserveInt64(maxIdleTimeClosed, stats.MaxIdleTimeClosed, options)
			observer.ObserveInt64(maxLifetimeClosed, stats.MaxLifetimeClosed, options)

			return nil
		},
		maxOpenConnections,
		openConnections,
		inUseConnections,
		idleConnections,
		waitCount,
		waitDuration,
		maxIdleClosed,
		maxIdleTimeClosed,
		maxLifetimeClosed,
	)

	return err
}
