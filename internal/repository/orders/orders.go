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
var ErrProcessingOrder = errors.New("order being processed")

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

func (r *OrderRepository) Create(ctx context.Context, inputOrder model.OrderCreateRequest) (string, error) {
	var order model.Order
	insertQuery := "INSERT INTO orders (user_id, order_number, uploaded_at, accrual, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, order_number, accrual, status, uploaded_at"
	err := r.db.GetContext(ctx, &order, insertQuery, inputOrder.UserID, inputOrder.OrderNumber, inputOrder.UploadedAt, 0, "REGISTERED")

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			selectQuery := "SELECT * FROM orders WHERE order_number = $1"
			r.db.GetContext(ctx, &order, selectQuery, inputOrder.OrderNumber)
			if order.UserID == inputOrder.UserID {
				r.logger.Info("order is being processed", zap.String("user_id", inputOrder.UserID), zap.String("order_number", inputOrder.OrderNumber))
				return order.OrderNumber, fmt.Errorf("%w: %s", ErrProcessingOrder, inputOrder.OrderNumber)
			}
			r.logger.Info("order already exists", zap.String("user_id", inputOrder.UserID), zap.String("order_number", inputOrder.OrderNumber))
			return "", fmt.Errorf("order for the order already exists: %w", ErrOrderAlreadyExists)
		}
		r.logger.Info("failed to create new order", zap.Error(err), zap.String("user_id", inputOrder.UserID), zap.String("order_number", inputOrder.OrderNumber))
		return "", fmt.Errorf("failed to create new order: %w", err)
	}
	return order.OrderNumber, nil
}
