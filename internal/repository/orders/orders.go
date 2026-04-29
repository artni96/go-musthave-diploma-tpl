package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ErrOrderAlreadyExists = errors.New("order already exists")

type OrderRepositoryInterface interface {
	Create(ctx context.Context, user model.UserCreateRequest) (string, error)
}

type OrderRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewOrderRepository(db *sqlx.DB, logger *zap.Logger) *OrderRepository {
	return &OrderRepository{
		db:     db,
		logger: logger,
	}
}

func (r *OrderRepository) Create(ctx context.Context, order model.OrderCreateRequest) (string, error) {
	var createdOrder model.Order
	insertQuery := "INSERT INTO orders (user_id, order_number, uploaded_at, accrual, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, order_number, accrual, status, uploaded_at"
	err := r.db.GetContext(ctx, &createdOrder, insertQuery, order.UserID, order.OrderNumber, order.UploadedAt, 0, "PROCESSING")

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.Info("order already exists", zap.String("user_id", order.UserID), zap.String("order_number", order.OrderNumber))
			return "", fmt.Errorf("order for the order already exists: %w", ErrOrderAlreadyExists)
		}
		r.logger.Info("failed to create new order", zap.Error(err), zap.String("user_id", order.UserID), zap.String("order_number", order.OrderNumber))
		return "", fmt.Errorf("failed to create new order: %w", err)
	}
	return createdOrder.OrderNumber, nil
}
