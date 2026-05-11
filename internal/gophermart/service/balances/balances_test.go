package balances

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	mocks "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/mocks/repository/balances"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initService(t *testing.T) (*BalanceService, *mocks.MockBalanceRepositoryInterface, context.Context, string, string) {
	//testDBDSN := "host=localhost port=5432 user=test password=test dbname=gophermart_test sslmode=disable"
	testDBDSN := config.TestsDBDSN()
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	logger := zap.NewNop()
	db, err := config.InitDBConnection(ctx, &cfg, false, logger)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repositoryMock := mocks.NewMockBalanceRepositoryInterface(ctrl)
	testService := NewBalanceService(repositoryMock, &app)

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
	return testService, repositoryMock, ctx, user1, user2
}

func TestWithdraw(t *testing.T) {
	serv, repo, ctx, user1, _ := initService(t)
	tests := []struct {
		name string
		body model.TransactionCreate
		want error
	}{
		{
			name: "success",
			body: model.TransactionCreate{
				UserID:      user1,
				Sum:         1000,
				Order:       "4444333322221111",
				ProcessedAt: time.Now().Format(time.RFC3339),
			},
			want: nil,
		},
		{
			name: "failure - not enough money",
			body: model.TransactionCreate{
				UserID:      user1,
				Sum:         100,
				Order:       "12345",
				ProcessedAt: time.Now().Format(time.RFC3339),
			},
			want: balances.ErrNotEnoughMoney,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.body.Sum = tt.body.Sum * 100
			repo.EXPECT().Withdraw(&ctx, tt.body).Return(tt.want)
			tt.body.Sum = tt.body.Sum / 100
			err := serv.Withdraw(&ctx, tt.body)
			if err != nil {
				if tt.name == "failure - not enough money" {
					assert.ErrorIs(t, err, tt.want)
				} else {
					t.Error(err)
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	serv, repo, ctx, user1, _ := initService(t)
	tests := []struct {
		name   string
		want   model.BalanceResponse
		userID string
		err    error
	}{
		{
			name: "success",
			want: model.BalanceResponse{
				Current:   0,
				Withdrawn: 0,
			},
			userID: user1,
			err:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo.EXPECT().Get(&ctx, tt.userID).Return(tt.want, nil)
			res, err := serv.Get(&ctx, tt.userID)
			if err != nil {
				log.Fatal(err)
			}
			assert.Equal(t, tt.want, res)
			assert.ErrorIs(t, tt.err, err)
		})
	}
}
