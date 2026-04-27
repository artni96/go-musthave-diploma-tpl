package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type UserRepositoryInterface interface {
	Create(ctx context.Context, user model.UserCreateRequest) (string, error)
	GetByLogin(ctx context.Context, login string) (model.User, error)
}

var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func (r *UserRepository) Create(ctx context.Context, user model.UserCreateRequest) (string, error) {
	var userID string
	insertQuery := "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id"
	err := r.db.GetContext(ctx, &userID, insertQuery, user.Login, user.Password)

	if err != nil {
		if err.Error() == "duplicate key value violates unique constraint \"users_login_key\"" {
			r.logger.Info("User already exists", zap.Error(err), zap.String("login", user.Login))
			return "", fmt.Errorf("%w: %s", ErrUserAlreadyExists, user.Login)
		}
		r.logger.Debug("failed to create user", zap.Error(err), zap.String("layer", "user repository"))
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.Debug("user successfully created", zap.String("username", user.Login), zap.String("layer", "user repository"))
	return userID, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (model.User, error) {
	selectQuery := "SELECT id, login, password FROM users WHERE login = $1;"
	responseEntity := model.User{}
	err := r.db.GetContext(ctx, &responseEntity, selectQuery, login)
	if err != nil {
		r.logger.Debug("user not found", zap.String("login", login), zap.String("layer", "user repository"))
		return responseEntity, fmt.Errorf("%w: %s", ErrUserNotFound, login)
	}
	return responseEntity, nil
}

func NewUserRepository(db *sqlx.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}
