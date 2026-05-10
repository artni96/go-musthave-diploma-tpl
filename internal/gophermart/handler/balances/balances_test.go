package balances

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	repomocks "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/mocks/repository/balances"
	servmocks "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/mocks/service/balances"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initConfig() (*context.Context, *sqlx.DB, *config.Config) {
	testDBDSN := config.TestsDBDSN()
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	db, err := config.InitDBConnection(ctx, &cfg, false)
	if err != nil {
		log.Fatal(err)
	}
	return &ctx, db, &cfg
}

func initHandler(ctx context.Context, db *sqlx.DB, cfg config.Config, t *testing.T) (*BalanceHandler, *servmocks.MockBalanceServiceInterface) {
	logger := zap.NewNop()

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
	if err = migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to clean up test database: %w", err))
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to run migrations: %w", err))
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repositoryMock := repomocks.NewMockBalanceRepositoryInterface(ctrl)
	serviceMock := servmocks.NewMockBalanceServiceInterface(ctrl)

	h := NewBalanceHandler(ctx, &app, repositoryMock, serviceMock)

	return h, serviceMock
}

func initUser(login string, password string) string {
	testDBDSN := config.TestsDBDSN()
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	db, err := config.InitDBConnection(ctx, &cfg, false)
	if err != nil {
		log.Fatal(err)
	}
	var userID string
	insertQuery := "INSERT INTO users (login, password) VALUES ($1, $2) returning id"
	err = db.GetContext(ctx, &userID, insertQuery, login, password)
	if err != nil {
		log.Fatal("failed to insert test user", zap.Error(err))
	}
	return userID
}

func TestGet(t *testing.T) {
	ctx, db, cfg := initConfig()
	h, serv := initHandler(*ctx, db, *cfg, t)
	userID := initUser("test", "test")

	type want struct {
		status      int
		contentType string
		message     string
	}
	tests := []struct {
		name   string
		want   want
		userID string
	}{
		{
			name: "response 200",
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				message:     "{\"current\":100,\"withdrawn\":10.19}",
			},
			userID: userID,
		},
		{
			name: "response 401",
			want: want{
				status:      http.StatusUnauthorized,
				contentType: "application/json",
			},
			userID: "",
		},
		{
			name: "response 500",
			want: want{
				status:      http.StatusInternalServerError,
				contentType: "application/json",
				message:     "{\"error\":\"Internal server error\"}",
			},
			userID: userID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "response 200" {
				serv.EXPECT().Get(ctx, tt.userID).Return(model.BalanceResponse{Current: 100, Withdrawn: 10.19}, nil)
			} else if tt.name == "response 500" {
				serv.EXPECT().Get(ctx, tt.userID).Return(model.BalanceResponse{}, errors.New("some error"))
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.userID != "" {
				req.Header.Set("UserID", tt.userID)
			}
			h.Get(w, req)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, res.StatusCode, tt.want.status)
			assert.Equal(t, res.Header.Get("Content-Type"), tt.want.contentType)
			assert.Equal(t, w.Body.String(), tt.want.message)
			if tt.name == "response 200" {
				assert.NotEqual(t, res.ContentLength, 0)
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	ctx, db, cfg := initConfig()
	h, serv := initHandler(*ctx, db, *cfg, t)
	userID := initUser("test", "test")
	var pgErr *pgconn.PgError

	type want struct {
		status      int
		contentType string
		message     string
	}
	type request struct {
		body  string
		error error
	}
	tests := []struct {
		name    string
		request request
		want    want
		userID  string
	}{
		{
			name: "response 200",
			request: request{
				body:  "{\n\"order\": \"49927398716\",\n\"sum\": 250\n}",
				error: nil,
			},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				message:     "",
			},
			userID: userID,
		},
		{
			name: "response 402 - order already has transaction",
			request: request{
				body:  "{\n\"order\": \"49927398716\",\n\"sum\": 250\n}",
				error: pgErr,
			},
			want: want{
				status:      http.StatusPaymentRequired,
				contentType: "application/json",
				message:     "{\"error\":\"order already has transaction\"}",
			},
			userID: userID,
		},
		{
			name: "response 401",
			request: request{
				body:  "{\n\"order\": \"49927398716\",\n\"sum\": 250\n}",
				error: nil,
			},
			want: want{
				status:      http.StatusUnauthorized,
				contentType: "application/json",
				message:     "",
			},
			userID: "",
		},
		{
			name: "response 402",
			request: request{
				body:  "{\n\"order\": \"49927398716\",\n\"sum\": 250\n}",
				error: balances.ErrNotEnoughMoney,
			},
			want: want{
				status:      http.StatusPaymentRequired,
				contentType: "application/json",
				message:     "{\"error\":\"not enough money\"}",
			},
			userID: userID,
		},
		{
			name: "response 422",
			request: request{
				body:  "{\n\"order\": \"123456\",\n\"sum\": 250\n}",
				error: errors.New("some error"),
			},
			want: want{
				status:      http.StatusUnprocessableEntity,
				contentType: "application/json",
				message:     "{\"error\":\"invalid order number\"}",
			},
			userID: userID,
		},
		{
			name: "response 500",
			request: request{
				body:  "{\n\"order\": \"49927398716\",\n\"sum\": 250\n}",
				error: errors.New("some error"),
			},
			want: want{
				status:      http.StatusInternalServerError,
				contentType: "application/json",
				message:     "{\"error\":\"internal server error\"}",
			},
			userID: userID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.userID != "" {
				serv.EXPECT().Withdraw(ctx, gomock.Any()).Return(tt.request.error)
			}

			body := strings.NewReader(tt.request.body)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/withdraw", body)
			req.Header.Set("UserID", tt.userID)
			h.Withdraw(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.message, w.Body.String())
		})

	}
}
