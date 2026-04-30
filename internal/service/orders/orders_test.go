package orders

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository/orders"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initService() (*OrderService, context.Context) {
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

	app := config.App{
		DB:     db,
		Config: &cfg,
		Logger: logger,
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
	testRepository := orders.NewOrderRepository(db, logger)
	testService := NewOrderService(testRepository, &app)
	return testService, ctx
}

func TestCreate(t *testing.T) {
	serv, ctx := initService()

	tests := []struct {
		name string
		req  model.OrderCreateRequest
	}{
		{
			name: "success",
			req: model.OrderCreateRequest{
				UserID: "1d210e7d-cda1-497e-8a3d-928f0bd9da3c",
				Number: "1",
			},
		},
		{
			name: "failure - order duplicate",
			req: model.OrderCreateRequest{
				UserID: "1d210e7d-cda1-497e-8a3d-928f0bd9da3c",
				Number: "1",
			},
		},
		{
			name: "failure - order created by another user",
			req: model.OrderCreateRequest{
				UserID: "1d210e7d-cda1-497e-8a3d-928f0bd9da3d",
				Number: "1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := serv.Create(ctx, tt.req)
			if err != nil {
				if tt.name == "failure - order duplicate" {
					assert.ErrorIs(t, err, orders.ErrProcessingOrder)
				} else if tt.name == "failure - order created by another user" {
					assert.ErrorIs(t, err, orders.ErrOrderAlreadyExists)
				} else {
					t.Error(fmt.Errorf("failed to create order: %w", err))
				}
			}
			if tt.name == "success" {
				assert.NotEmpty(t, res)
				assert.IsType(t, "string", res)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	serv, ctx := initService()

	_, err := serv.Create(ctx, model.OrderCreateRequest{
		UserID: "1d210e7d-cda1-497e-8a3d-928f0bd9da3c",
		Number: "1",
	})
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name string
		req  model.OrderUpdateRequest
	}{
		{
			name: "success",
			req: model.OrderUpdateRequest{
				Number:  "1",
				Accrual: 100,
			},
		},
		{
			name: "failure - wrong number",
			req: model.OrderUpdateRequest{
				Number:  "2",
				Accrual: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = serv.Update(ctx, tt.req)
			if err != nil {
				if tt.name == "failure - wrong number" {
					assert.ErrorIs(t, err, orders.ErrOrderNotFound)
				} else {
					t.Error(err)
				}
			}
			if tt.name == "success" {
				assert.Empty(t, err)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	serv, ctx := initService()

	_, err := serv.Create(ctx, model.OrderCreateRequest{
		UserID: "1d210e7d-cda1-497e-8a3d-928f0bd9da3c",
		Number: "1",
	})
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name string
		req  model.OrderStatusUpdateRequest
	}{
		{
			name: "success",
			req: model.OrderStatusUpdateRequest{
				Number: "1",
				Status: "PROCESSING",
			},
		},
		{
			name: "failure - wrong number",
			req: model.OrderStatusUpdateRequest{
				Number: "2",
				Status: "PROCESSING",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = serv.UpdateStatus(ctx, tt.req)
			if err != nil {
				if tt.name == "failure - wrong number" {
					assert.ErrorIs(t, err, orders.ErrOrderNotFound)
				} else {
					t.Error(err)
				}
			}
			if tt.name == "success" {
				assert.Empty(t, err)
			}
		})
	}
}
