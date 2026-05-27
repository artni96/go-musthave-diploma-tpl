package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDBConnection(ctx context.Context, cfg *Config, upgradeDB bool, logger *zap.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	isConnected := false
	for i := 0; i <= 9; i++ {
		time.Sleep(time.Duration(i) * time.Second)
		logger.Info("connecting to database", zap.Int("attempt", i+1))
		err = db.PingContext(pingCtx)
		if err == nil {
			isConnected = true
			break
		}
	}
	if !isConnected {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	logger.Info("successfully connected to database")

	if upgradeDB {
		if err = runMigrations(db, logger); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func runMigrations(db *sqlx.DB, logger *zap.Logger) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		logger.Info("failed to create database driver", zap.Error(err))
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		logger.Info("failed to initialize migrator", zap.Error(err))
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Info("failed to run migrations", zap.Error(err))
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	logger.Info("migrations implemented successfully")
	return nil
}
