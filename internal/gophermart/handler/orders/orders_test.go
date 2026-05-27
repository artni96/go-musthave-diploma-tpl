package orders

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	ordersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/orders"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/orders"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initConfig() (*context.Context, *sqlx.DB, *config.Config) {
	testDBDSN := config.TestsDBDSN()
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	db, err := config.InitDBConnection(ctx, &cfg, false, zap.NewNop())
	if err != nil {
		log.Fatal(err)
	}
	return &ctx, db, &cfg
}

func initHandler(ctx context.Context, db *sqlx.DB, cfg config.Config) (*OrderHandler, chan model.OrderQueue) {
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
	testRepository := ordersrepo.NewOrderRepository(db, logger)
	testService := ordersserv.NewOrderService(testRepository, &app)
	testQueue := make(chan model.OrderQueue, 100)
	h := NewOrderHandler(&ctx, &app, testRepository, testService, testQueue)

	return h, testQueue
}

func initUser(login string, password string) string {
	testDBDSN := config.TestsDBDSN()
	cfg := config.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	db, err := config.InitDBConnection(ctx, &cfg, false, zap.NewNop())
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

func initOrder(db *sqlx.DB, ctx context.Context, userID string, number string, accrual int, status string, uploadedAt string) {
	insertQuery := "INSERT INTO orders (user_id, accrual, status, number, uploaded_at) VALUES ($1, $2, $3, $4, $5)"
	_, err := db.ExecContext(ctx, insertQuery, userID, accrual, status, number, uploadedAt)
	if err != nil {
		log.Fatal("failed to insert test order", zap.Error(err), zap.String("user_id", userID))
	}

}

func TestCreate(t *testing.T) {
	ctx, db, cfg := initConfig()
	h, queue := initHandler(*ctx, db, *cfg)
	user1 := initUser("test1", "test1")
	user2 := initUser("test2", "test2")

	type want struct {
		status      int
		contentType string
		response    string
	}

	type request struct {
		userID string
		body   int
	}

	tests := []struct {
		name    string
		method  string
		request request
		want    want
	}{
		{
			name:   "response 202",
			method: http.MethodPost,
			request: request{
				userID: user1,
				body:   4444333322221111,
			},
			want: want{
				status:      http.StatusAccepted,
				contentType: "application/json",
			},
		},
		{
			name:   "response 200",
			method: http.MethodPost,
			request: request{
				userID: user1,
				body:   4444333322221111,
			},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
				response:    "{\"error\":\"order being processed\"}",
			},
		},
		{
			name:   "response 400",
			method: http.MethodPost,
			request: request{
				userID: user2,
				body:   4444333322221111,
			},
			want: want{
				status:      http.StatusBadRequest,
				contentType: "application/json",
			},
		},
		{
			name:   "response 401",
			method: http.MethodPost,
			request: request{
				userID: "",
				body:   4444333322221111,
			},
			want: want{
				status:      http.StatusUnauthorized,
				contentType: "application/json",
			},
		},
		{
			name:   "response 409",
			method: http.MethodPost,
			request: request{
				userID: user2,
				body:   4444333322221111,
			},
			want: want{
				status:      http.StatusConflict,
				contentType: "application/json",
				response:    "{\"error\":\"order already created by another user\"}",
			},
		},
		{
			name:   "response 422",
			method: http.MethodPost,
			request: request{
				userID: user2,
				body:   12345,
			},
			want: want{
				status:      http.StatusUnprocessableEntity,
				contentType: "application/json",
				response:    "{\"error\":\"invalid order number\"}",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			go func() {
				for range queue {
				}
			}()
			var req *http.Request
			if tt.name == "response 400" {
				reqBody, err := json.Marshal(tt.request)
				if err != nil {
					t.Error(err)
				}
				reader := bytes.NewReader(reqBody)
				req = httptest.NewRequest(tt.method, "/api/user/orders", reader)
			} else {
				reqBody := bytes.NewBufferString(strconv.Itoa(tt.request.body))
				req = httptest.NewRequest(tt.method, "/api/user/orders", reqBody)
			}

			req.Header.Set("Content-Type", "text/plain")
			req.Header.Set("UserID", tt.request.userID)
			w := httptest.NewRecorder()
			h.Create(w, req)

			res := w.Result()
			defer res.Body.Close()
			respBody, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatal(err)
			}
			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			if tt.want.response != "" {
				assert.Equal(t, tt.want.response, string(respBody))
			}
		})
	}
}

func TestGetList(t *testing.T) {
	ctx, db, cfg := initConfig()
	h, _ := initHandler(*ctx, db, *cfg)
	user1 := initUser("test1", "test1")
	user2 := initUser("test2", "test2")

	type want struct {
		status      int
		contentType string
	}
	type request struct {
		userID string
		method string
	}

	type orders []struct {
		userID  string
		accrual int
		status  string
		number  string
	}

	tests := []struct {
		name    string
		request request
		want    want
		orders  orders
	}{
		{
			name:    "response 200",
			request: request{userID: user1, method: http.MethodGet},
			want: want{
				status:      http.StatusOK,
				contentType: "application/json",
			},
			orders: orders{
				{userID: user1, accrual: 10000, status: "PROCESSED", number: "79927398713"},
				{userID: user1, accrual: 20000, status: "PROCESSING", number: "49927398716"},
				{userID: user1, accrual: 0, status: "INVALID", number: "1234567812345670"},
			},
		},
		{
			name:    "response 204",
			request: request{userID: user2, method: http.MethodGet},
			want: want{
				status:      http.StatusNoContent,
				contentType: "application/json",
			},
		},
		{
			name:    "response 401",
			request: request{userID: "", method: http.MethodGet},
			want: want{
				status:      http.StatusUnauthorized,
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.orders != nil {
				for i, order := range tt.orders {
					uploadedAt := time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339)
					initOrder(db, *ctx, order.userID, order.number, order.accrual, order.status, uploadedAt)
				}
			}
			req1 := httptest.NewRequest(tt.request.method, "/api/user/orders", nil)
			req1.Header.Set("UserID", tt.request.userID)
			w1 := httptest.NewRecorder()
			h.GetList(w1, req1)
			res := w1.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.status, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			if tt.orders != nil {
				respBody, err := io.ReadAll(res.Body)
				if err != nil {
					log.Fatal(err)
				}
				var checkOrders []model.OrderResponse
				err = json.Unmarshal(respBody, &checkOrders)
				if err != nil {
					log.Fatal(err)
				}
				assert.Equal(t, len(checkOrders), len(tt.orders))
				for i, order := range checkOrders {
					assert.Equal(t, tt.orders[len(tt.orders)-1-i].status, order.Status)
					assert.Equal(t, tt.orders[len(tt.orders)-1-i].number, order.Number)
					assert.IsType(t, 0.0, order.Accrual)
					assert.Equal(t, float64(tt.orders[len(tt.orders)-1-i].accrual/100), order.Accrual)
				}

				req2 := httptest.NewRequest(tt.request.method, "/api/user/orders", nil)
				req2.Header.Set("UserID", user2)
				w2 := httptest.NewRecorder()
				h.GetList(w2, req2)
				res2 := w2.Result()
				defer res2.Body.Close()
				assert.Equal(t, http.StatusNoContent, res2.StatusCode)
			}
		})
	}
}
