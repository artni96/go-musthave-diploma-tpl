package balances

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

func initRepository() (*BalanceRepository, *context.Context, string, string) {
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
	testRepository := NewBalanceRepository(db, logger)

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to start transaction: %w", err))
	}
	insertUserQuery := "INSERT INTO users (login, password) VALUES ($1, $2) returning id;"
	insertBalanceQuery := "INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, $2, $3);"
	var user1 string
	err = tx.GetContext(ctx, &user1, insertUserQuery, "test1", "test1")
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, insertBalanceQuery, user1, 50000, 0)
	if err != nil {
		log.Fatal("failed to insert balance", zap.Error(err), zap.String("user", user1))
	}
	var user2 string
	err = tx.GetContext(ctx, &user2, insertUserQuery, "test2", "test2")
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, insertBalanceQuery, user2, 100000, 10000)
	if err != nil {
		log.Fatal("failed to insert balance", zap.Error(err), zap.String("user", user2))
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(fmt.Errorf("failed to commit at fixture creation: %w", err))
	}
	return testRepository, &ctx, user1, user2
}

func TestWithdraw(t *testing.T) {
	repo, ctx, user1, _ := initRepository()
	db, err := testDBConn(*ctx)
	if err != nil {
		log.Fatal("failed to initialize test db conn", zap.Error(err))
	}

	type req struct {
		data model.TransactionCreate
	}

	type want struct {
		current   int
		withdrawn int
	}

	tests := []struct {
		name string
		req  req
		want want
	}{
		{
			name: "success",
			req: req{
				data: model.TransactionCreate{
					UserID:      user1,
					Sum:         40000,
					Order:       "4444333322221111",
					ProcessedAt: time.Now().Format(time.RFC3339),
				},
			},
			want: want{
				current:   10000,
				withdrawn: 40000,
			},
		},
		{
			name: "failure - not enough money",
			req: req{
				data: model.TransactionCreate{
					UserID:      user1,
					Sum:         60000,
					Order:       "4444333322221111",
					ProcessedAt: time.Now().Format(time.RFC3339),
				},
			},
			want: want{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = repo.Withdraw(ctx, tt.req.data)
			if err != nil {
				if tt.name == "failure - not enough money" {
					assert.True(t, errors.Is(err, ErrNotEnoughMoney))
				} else {
					t.Fatal("failed to withdraw", zap.Error(err), zap.String("user", user1))
				}
			}
			type userBalanceRes struct {
				Current   int `db:"current"`
				Withdrawn int `db:"withdrawn"`
			}

			var userBalance userBalanceRes
			userBalanceSelectQuery := "SELECT current, withdrawn FROM balance WHERE user_id = $1"
			err = db.GetContext(*ctx, &userBalance, userBalanceSelectQuery, tt.req.data.UserID)
			if err != nil {
				if tt.name == "failure - not enough money" {
					assert.True(t, errors.Is(err, ErrNotEnoughMoney))
				} else {
					t.Fatal("failed to get user balance", zap.Error(err), zap.String("user", user1))
				}
			}
			type transactionRes struct {
				Sum         int    `db:"sum"`
				ProcessedAt string `db:"processed_at"`
			}
			transactionSelectQuery := `SELECT sum, processed_at FROM withdrawals WHERE user_id = $1 and "order" = $2`
			var transaction transactionRes
			err = db.GetContext(*ctx, &transaction, transactionSelectQuery, tt.req.data.UserID, tt.req.data.Order)
			if err != nil {
				t.Fatal("failed to check transaction", zap.Error(err), zap.String("user", user1), zap.String("order", tt.req.data.Order))
			}

			if tt.name == "success" {
				assert.Equal(t, tt.want.current, userBalance.Current)
				assert.Equal(t, tt.want.withdrawn, userBalance.Withdrawn)
				assert.Equal(t, tt.want.withdrawn, -transaction.Sum)
				assert.IsType(t, time.Now().Format(time.RFC3339), transaction.ProcessedAt)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	repo, ctx, user1, user2 := initRepository()

	type want struct {
		current   float64
		withdrawn float64
	}
	tests := []struct {
		name   string
		want   want
		userID string
	}{
		{
			name: "success - user 1",
			want: want{
				current:   500,
				withdrawn: 0,
			},
			userID: user1,
		},
		{
			name: "success - user 2",
			want: want{
				current:   1000,
				withdrawn: 100,
			},
			userID: user2,
		},
		{
			name: "failure - user not found",
			want: want{
				current:   1000,
				withdrawn: 100,
			},
			userID: "8be4df61-93ca-11d2-aa0d-00e098032b8c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := repo.Get(ctx, tt.userID)
			if err != nil {
				if tt.name == "failure - user not found" {
					assert.Error(t, err)
					assert.Equal(t, err.Error(), "failed to get user balances: sql: no rows in result set")
				} else {
					log.Fatal(err)
				}
			}
			if tt.name != "failure - user not found" {
				assert.Equal(t, tt.want.current, res.Current)
				assert.Equal(t, tt.want.withdrawn, res.Withdrawn)
			}
		})
	}
}
