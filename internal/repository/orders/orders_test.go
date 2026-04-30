package orders

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initRepository() (*OrderRepository, *context.Context) {
	testDBDSN := "host=localhost port=5432 user=test password=test dbname=gophermart_test sslmode=disable"
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	logger := zap.NewNop()
	db, err := config.InitDBConnection(ctx, &cfg, false)
	if err != nil {
		log.Fatal(err)
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create database driver: %w", err))
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://../../../migrations", "postgres", driver)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize test migrator: %w", err))
	}
	if err := migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to clean up test database: %w", err))
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to run migrations: %w", err))
	}
	testRepository := NewOrderRepository(db, logger)
	return testRepository, &ctx
}

func TestCreate(t *testing.T) {
	repo, ctx := initRepository()

	type req struct {
		order model.OrderCreateRequest
	}
	tests := []struct {
		name string
		req  req
	}{
		{
			name: "success",
			req: req{
				order: model.OrderCreateRequest{
					UserID:     "1d210e7d-cda1-497e-8a3d-928f0bd9da3c",
					Number:     "1",
					UploadedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
		{
			name: "failure - order duplicate",
			req: req{
				order: model.OrderCreateRequest{
					UserID:     "1d210e7d-cda1-497e-8a3d-928f0bd9da3c",
					Number:     "1",
					UploadedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
		{
			name: "failure - created by another user",
			req: req{
				order: model.OrderCreateRequest{
					UserID:     "1d210e7d-cda1-497e-8a3d-928f0bd9da3d",
					Number:     "1",
					UploadedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := repo.Create(*ctx, tt.req.order)
			if err != nil {
				if tt.name == "success" {
					t.Error(err)
				} else if tt.name == "failure - order duplicate" {
					assert.ErrorIs(t, err, ErrProcessingOrder)
					assert.Equal(t, res, tt.req.order.Number)
				} else if tt.name == "failure - created by another user" {
					assert.ErrorIs(t, err, ErrOrderAlreadyExists)
					assert.Empty(t, res)
				}

			}
			if tt.name == "success" {
				assert.Equal(t, tt.req.order.Number, res)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	repo, ctx := initRepository()

	_, err := repo.Create(*ctx, model.OrderCreateRequest{
		UserID:     "1d210e7d-cda1-497e-8a3d-928f0bd9da3d",
		Number:     "1",
		UploadedAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		req  model.OrderUpdateRequest
	}{
		{
			name: "success",
			req: model.OrderUpdateRequest{
				Number:  "1",
				Accrual: 700,
				Status:  "PROCESSED",
			},
		},
		{
			name: "failure - wrong number",
			req: model.OrderUpdateRequest{
				Number:  "2",
				Accrual: 700,
				Status:  "PROCESSED",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = repo.Update(*ctx, tt.req)
			if tt.name == "success" {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrOrderNotFound)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	repo, ctx := initRepository()
	_, err := repo.Create(*ctx, model.OrderCreateRequest{
		UserID:     "1d210e7d-cda1-497e-8a3d-928f0bd9da3d",
		Number:     "1",
		UploadedAt: time.Now().Format(time.RFC3339),
	})
	tests := []struct {
		name string
		req  model.OrderStatusUpdateRequest
	}{
		{
			name: "success",
			req: model.OrderStatusUpdateRequest{
				Status: "PROCESSED",
				Number: "1",
			},
		},
		{
			name: "failure - wrong number",
			req: model.OrderStatusUpdateRequest{
				Status: "PROCESSED",
				Number: "2",
			},
		},
		{
			name: "failure - wrong status",
			req: model.OrderStatusUpdateRequest{
				Status: "APPLIED",
				Number: "1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = repo.UpdateStatus(*ctx, tt.req)
			if tt.name == "success" {
				assert.NoError(t, err)
			} else if tt.name == "failure - wrong number" {
				assert.ErrorIs(t, err, ErrOrderNotFound)
			} else {
				assert.ErrorIs(t, err, ErrUnavailableStatus)
			}
		})
	}
}
