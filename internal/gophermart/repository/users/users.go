package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/jackc/pgx/v5/pgconn"
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
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failure to begin transaction: %w", err)
	}

	defer tx.Rollback()

	var userID string
	insertUserQuery := "INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id"
	err = tx.GetContext(ctx, &userID, insertUserQuery, user.Login, user.Password)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.Info("User already exists", zap.Error(err), zap.String("login", user.Login))
			return "", ErrUserAlreadyExists
		}
		r.logger.Debug("failed to create user", zap.Error(err))
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	insertBalanceQuery := "INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, $2, $3)"
	_, err = tx.ExecContext(ctx, insertBalanceQuery, userID, 0, 0)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.Info("User balance already exists", zap.Error(err), zap.String("login", user.Login))
			return "", ErrUserAlreadyExists
		}
		r.logger.Debug("failed to create user balance", zap.Error(err))
		return "", fmt.Errorf("failed to create user balance: %w", err)
	}
	if err = tx.Commit(); err != nil {
		r.logger.Debug("failed to create user", zap.Error(err), zap.String("login", user.Login))
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	r.logger.Debug("user successfully created", zap.String("username", user.Login))
	return userID, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (model.User, error) {
	selectQuery := "SELECT id, login, password FROM users WHERE login = $1;"
	responseEntity := model.User{}
	err := r.db.GetContext(ctx, &responseEntity, selectQuery, login)
	if err != nil {
		r.logger.Debug("user not found", zap.String("login", login))
		return responseEntity, ErrUserNotFound
	}
	return responseEntity, nil
}

func NewUserRepository(db *sqlx.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}
