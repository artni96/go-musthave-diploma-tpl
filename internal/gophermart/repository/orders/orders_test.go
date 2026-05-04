package orders

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func testDBConn(ctx context.Context) (*sqlx.DB, error) {
	testDBDSN := "host=localhost port=5432 user=test password=test dbname=gophermart_test sslmode=disable"
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	db, err := config.InitDBConnection(ctx, &cfg, false)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initRepository() (*OrderRepository, *context.Context, string, string) {
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

	migrator, err := migrate.NewWithDatabaseInstance("file://../../../../migrations", "postgres", driver)
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

	userIDQuery := "INSERT INTO users (login, password) VALUES ($1, $2) returning id;"
	var user1 string
	err = db.GetContext(ctx, &user1, userIDQuery, "test1", "test1")
	if err != nil {
		log.Fatal(err)
	}
	var user2 string
	err = db.GetContext(ctx, &user2, userIDQuery, "test2", "test2")
	if err != nil {
		log.Fatal(err)
	}
	return testRepository, &ctx, user1, user2
}

func TestCreate(t *testing.T) {
	repo, ctx, user1, user2 := initRepository()

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
					UserID:     user1,
					Number:     "1",
					UploadedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
		{
			name: "failure - order duplicate",
			req: req{
				order: model.OrderCreateRequest{
					UserID:     user1,
					Number:     "1",
					UploadedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
		{
			name: "failure - created by another user",
			req: req{
				order: model.OrderCreateRequest{
					UserID:     user2,
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

	repo, ctx, user1, user2 := initRepository()

	db, err := testDBConn(*ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.Create(*ctx, model.OrderCreateRequest{
		UserID:     user1,
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
				UserID:  user1,
			},
		},
		{
			name: "failure - wrong number (order not exists)",
			req: model.OrderUpdateRequest{
				Number:  "2",
				Accrual: 700,
				Status:  "PROCESSED",
				UserID:  user1,
			},
		},
		{
			name: "success - accrual 0",
			req: model.OrderUpdateRequest{
				Number:  "3",
				Accrual: 0,
				Status:  "PROCESSED",
				UserID:  user2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = repo.Update(*ctx, tt.req)
			if tt.name == "success" {
				assert.NoError(t, err)
				assert.Equal(t, tt.req.Status, "PROCESSED")

				var userBalance int
				selectQuery := `SELECT current FROM balance WHERE user_id = $1;`
				err = db.GetContext(*ctx, &userBalance, selectQuery, tt.req.UserID)
				assert.Equal(t, tt.req.Accrual, userBalance)
			} else {
				assert.ErrorIs(t, err, ErrOrderNotFound)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	repo, ctx, userID, _ := initRepository()
	_, err := repo.Create(*ctx, model.OrderCreateRequest{
		UserID:     userID,
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

func TestGetList(t *testing.T) {
	repo, ctx, user1, user2 := initRepository()

	tests := []struct {
		name   string
		userID string
		orders []model.OrderCreateRequest
	}{
		{
			name:   "success",
			userID: user1,
			orders: []model.OrderCreateRequest{
				{
					UserID: user1,
					Number: "1",
				},
				{
					UserID: user1,
					Number: "2",
				},
				{
					UserID: user1,
					Number: "3",
				},
			},
		},
		{
			name:   "success - no orders",
			userID: user2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, order := range tt.orders {
				curTime := time.Now().Add(time.Duration(i) * time.Second)
				order.UploadedAt = curTime.Format(time.RFC3339)
				_, err := repo.Create(*ctx, order)
				if err != nil {
					t.Fatal(err)
				}
			}

			userOrders, err := repo.GetList(*ctx, tt.userID)
			if err != nil {
				t.Fatal(err)
			}

			if tt.name == "success" {
				assert.Equal(t, len(tt.orders), len(userOrders))
				assert.IsType(t, []model.OrderResponse{}, userOrders)
				assert.Equal(t, err, nil)
				for i, order := range userOrders {
					assert.Equal(t, tt.orders[i].Number, userOrders[len(userOrders)-1-i].Number)
					assert.Equal(t, order.Accrual, 0.0)
					assert.Equal(t, order.Status, "NEW")
					assert.IsType(t, time.Now().Format(time.RFC3339), order.UploadedAt)
				}
			} else {
				assert.Empty(t, userOrders)
			}
		})
	}
}
