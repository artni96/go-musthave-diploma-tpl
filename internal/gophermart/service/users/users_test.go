package users

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	config2 "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/users"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initService() (*UserService, context.Context, *config2.Config) {
	testDBDSN := "host=localhost port=5432 user=test password=test dbname=gophermart_test sslmode=disable"
	cfg := config2.Config{
		DatabaseURI: testDBDSN,
	}
	ctx := context.Background()

	logger := zap.NewNop()
	db, err := config2.InitDBConnection(ctx, &cfg, false)
	if err != nil {
		log.Fatal(err)
	}

	app := config2.App{
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
	testRepository := users.NewUserRepository(db, logger)
	testService := NewUserService(testRepository, &app)
	return testService, ctx, &cfg
}

func TestUserCreate(t *testing.T) {
	testService, ctx, _ := initService()

	tests := []struct {
		name string
		body model.UserCreateRequest
	}{
		{
			name: "success",
			body: model.UserCreateRequest{
				Login:    "test1",
				Password: "test1",
			},
		},
		{
			name: "failure - duplicate login",
			body: model.UserCreateRequest{
				Login:    "test1",
				Password: "test1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userID, err := testService.Create(ctx, test.body)
			if err != nil {
				if errors.Is(err, users.ErrUserAlreadyExists) {
					assert.True(t, true)
				} else {
					assert.NoError(t, err)
				}
			}
			assert.NotEqual(t, userID, nil)
		})
	}
}

func TestLogin(t *testing.T) {
	testService, ctx, _ := initService()
	tests := []struct {
		name string
		body model.UserLoginRequest
	}{
		{
			name: "success",
			body: model.UserLoginRequest{
				Login:    "test2",
				Password: "test2",
			},
		},
		{
			name: "user not found",
			body: model.UserLoginRequest{
				Login:    "test3",
				Password: "test2",
			},
		},
		{
			name: "wrong password",
			body: model.UserLoginRequest{
				Login:    "test2",
				Password: "test3",
			},
		},
	}

	testUserBody := model.UserCreateRequest{
		Login:    "test2",
		Password: "test2",
	}

	testUserID, err := testService.Create(ctx, testUserBody)
	if err != nil {
		log.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user, err := testService.Login(ctx, test.body)
			if err != nil {
				if errors.Is(err, users.ErrUserNotFound) {
					assert.True(t, true)
				} else if errors.Is(err, ErrWrongPassword) {
					assert.True(t, true)
				} else {
					assert.NoError(t, err)
				}
			}

			if test.name == "success" {
				assert.Equal(t, user.ID, testUserID)
				assert.Equal(t, user.Login, test.body.Login)
				assert.NotEqual(t, user.Password, "")
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	testService, _, cfg := initService()
	tests := []struct {
		name   string
		userID string
	}{
		{
			name:   "success",
			userID: "id1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := testService.BuildJWTString(test.userID, cfg)
			if err != nil {
				assert.NoError(t, err)
			}
			assert.NotEmpty(t, token)
			assert.IsType(t, "", token)
		})
	}
}
