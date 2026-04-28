package users

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"github.com/artni96/go-musthave-diploma-tpl/internal/service"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initHandler() *UserHandler {
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
	if err = migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to clean up test database: %w", err))
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to run migrations: %w", err))
	}
	testRepository := repository.NewUserRepository(db, logger)
	testService := service.NewUserService(testRepository, &app)
	h := NewUserHandler(&ctx, &app, testRepository, testService)
	return h
}

func TestCreate(t *testing.T) {
	h := initHandler()

	type want struct {
		contentType string
		statusCode  int
		message     string
	}

	type request struct {
		body   string
		method string
	}

	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "response 200",
			request: request{
				body:   "{\"login\":\"test\",\"password\":\"test\"}",
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				message:     "{\"message\": \"user successfully created\"}",
			},
		},
		{
			name: "response 409",
			request: request{
				body:   "{\"login\":\"test\",\"password\":\"test\"}",
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusConflict,
				message:     "{\"error\": \"user already exists\"}",
			},
		},
		{
			name: "response 400 - invalid body",
			request: request{
				body:   "{\"login\":\"test\",\"password\":\"test}",
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusBadRequest,
				message:     "{\"error\": \"invalid request body\"}",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := strings.NewReader(tt.request.body)
			req := httptest.NewRequest(tt.request.method, "/api/users/register", reqBody)
			w := httptest.NewRecorder()
			h.Create(w, req)

			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.message, string(resBody))
			if tt.want.statusCode == http.StatusCreated {
				assert.Equal(t, res.Cookies()[0].Name, "jwt")
				assert.NotEqual(t, res.Cookies()[0].Value, nil)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	h := initHandler()

	type want struct {
		contentType string
		statusCode  int
		message     string
	}

	type body struct {
		login    string
		password string
	}

	type request struct {
		body   body
		method string
	}

	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "response 200",
			request: request{
				body:   body{login: "test", password: "test"},
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				message:     "",
			},
		},
		{
			name: "response 400 - invalid body",
			request: request{
				body:   body{login: "test", password: "test"},
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusBadRequest,
				message:     "{\"error\": \"invalid request body\"}",
			},
		},
		{
			name: "response 401 - wrong login",
			request: request{
				body:   body{login: "test", password: "test1"},
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusUnauthorized,
				message:     "{\"error\": \"wrong user or password\"}",
			},
		},
		{
			name: "response 401 - wrong password",
			request: request{
				body:   body{login: "test1", password: "test"},
				method: http.MethodPost,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusUnauthorized,
				message:     "{\"error\": \"wrong user or password\"}",
			},
		},
	}
	newUserStrBody := fmt.Sprintf("{\"login\":\"test\",\"password\":\"test\"}")
	newUserBody := strings.NewReader(newUserStrBody)

	req := httptest.NewRequest(http.MethodPost, "/api/users/register", newUserBody)
	w1 := httptest.NewRecorder()
	h.Create(w1, req)
	res := w1.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loginStrBody string

			if tt.name == "response 400 - invalid body" {
				loginStrBody = fmt.Sprintf("{\"username\":\"%s\",\"password\":\"%s\"}", tt.request.body.login, tt.request.body.password)
			} else {
				loginStrBody = fmt.Sprintf("{\"login\":\"%s\",\"password\":\"%s\"}", tt.request.body.login, tt.request.body.password)
			}

			loginBody := strings.NewReader(loginStrBody)
			req = httptest.NewRequest(tt.request.method, "/api/users/login", loginBody)
			w2 := httptest.NewRecorder()
			h.Login(w2, req)
			res = w2.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.message, string(resBody))
		})
	}
}
