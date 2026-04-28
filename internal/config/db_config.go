package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jmoiron/sqlx"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDBConnection(ctx context.Context, cfg *Config, upgradeDB bool) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	err = db.PingContext(pingCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database connection: %w", err)
	}
	if upgradeDB {
		if err := runMigrations(db); err != nil {
			return nil, err
		}
	}

	return db, nil
}

func runMigrations(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
