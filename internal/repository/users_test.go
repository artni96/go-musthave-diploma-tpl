package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func initRepository() (*UserRepository, *context.Context) {
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

	migrator, err := migrate.NewWithDatabaseInstance("file://../../migrations", "postgres", driver)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize test migrator: %w", err))
	}
	if err := migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to clean up test database: %w", err))
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(fmt.Errorf("failed to run migrations: %w", err))
	}
	testRepository := NewUserRepository(db, logger)
	return testRepository, &ctx
}

func TestCreate(t *testing.T) {
	repo, ctx := initRepository()

	type req struct {
		user model.UserCreateRequest
	}
	tests := []struct {
		name string
		req  req
	}{
		{
			name: "success",
			req: req{
				user: model.UserCreateRequest{
					Login:    "test",
					Password: "test",
				},
			},
		},
		{
			name: "duplicate login",
			req: req{
				user: model.UserCreateRequest{
					Login:    "test",
					Password: "test",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userID, err := repo.Create(*ctx, test.req.user)
			if err != nil {
				if errors.Is(err, ErrUserAlreadyExists) {
					assert.True(t, true)
				} else {
					assert.NoError(t, err)
				}

			}
			if test.name == "success" {
				assert.NotEmpty(t, userID)
				assert.IsType(t, "", userID)
			}
		})
	}
}

func TestGetByLogin(t *testing.T) {
	repo, ctx := initRepository()

	userID, err := repo.Create(*ctx, model.UserCreateRequest{
		Login:    "test",
		Password: "test",
	})
	if err != nil {
		log.Fatal(err)
	}

	tests := []struct {
		name  string
		login string
	}{
		{
			name:  "success",
			login: "test",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user, err := repo.GetByLogin(*ctx, test.login)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					assert.True(t, true)
				} else {
					assert.NoError(t, err)
				}
			}
			if test.name == "success" {
				assert.NotEmpty(t, user)
				assert.Equal(t, userID, user.ID)
				assert.NotEmpty(t, user.Password)
				assert.IsType(t, "", user.Password)
				assert.NotEmpty(t, user.Login)
				assert.Equal(t, test.login, user.Login)
			}
		})
	}
}
