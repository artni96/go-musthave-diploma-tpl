package orders

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ErrOrderAlreadyExists = errors.New("order already exists")
var ErrProcessingOrder = errors.New("order being processed")
var ErrOrderNotFound = errors.New("order not found")
var ErrUnavailableStatus = errors.New("status is unavailable")

var AvailableOrderStatus = []string{"PROCESSING", "PROCESSED", "NEW", "INVALID"}

type OrderRepositoryInterface interface {
	GetList(ctx context.Context, userID string) ([]model.OrderResponse, error)
	Create(ctx context.Context, order model.OrderCreateRequest) (string, error)
	Update(ctx context.Context, data model.OrderUpdateRequest) error
	UpdateStatus(ctx context.Context, data model.OrderStatusUpdateRequest) error
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

func (r *OrderRepository) GetList(ctx context.Context, userID string) ([]model.OrderResponse, error) {
	var result []model.OrderResponse

	selectQuery := "SELECT number, status, (accrual / 100.0) as accrual, uploaded_at FROM orders WHERE user_id = $1 order by uploaded_at desc"
	err := r.db.SelectContext(ctx, &result, selectQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user orders: %w", err)
	}
	return result, nil
}

func (r *OrderRepository) Create(ctx context.Context, inputOrder model.OrderCreateRequest) (string, error) {
	var order model.Order
	insertQuery := "INSERT INTO orders (user_id, number, uploaded_at, accrual, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, number, accrual, status, uploaded_at"
	err := r.db.GetContext(ctx, &order, insertQuery, inputOrder.UserID, inputOrder.Number, inputOrder.UploadedAt, 0, "NEW")

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			selectQuery := "SELECT * FROM orders WHERE number = $1"
			r.db.GetContext(ctx, &order, selectQuery, inputOrder.Number)
			if order.UserID == inputOrder.UserID {
				r.logger.Info("order is being processed", zap.String("user_id", inputOrder.UserID), zap.String("order_number", inputOrder.Number))
				return order.Number, fmt.Errorf("%w: %s", ErrProcessingOrder, inputOrder.Number)
			}
			r.logger.Info("order already exists", zap.String("user_id", inputOrder.UserID), zap.String("order_number", inputOrder.Number))
			return "", fmt.Errorf("order for the order already exists: %w", ErrOrderAlreadyExists)
		}
		r.logger.Info("failed to create new order", zap.Error(err), zap.String("user_id", inputOrder.UserID), zap.String("order_number", inputOrder.Number))
		return "", fmt.Errorf("failed to create new order: %w", err)
	}
	return order.Number, nil
}

func (r *OrderRepository) Update(ctx context.Context, data model.OrderUpdateRequest) error {

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failure to begin transaction: %w", err)
	}

	defer tx.Rollback()

	updateOrderQuery := "UPDATE orders SET accrual=$1, status=$2 WHERE number=$3"
	res, err := tx.ExecContext(ctx, updateOrderQuery, data.Accrual, data.Status, data.Number)
	if err != nil {
		r.logger.Info("failed to update order", zap.Error(err), zap.String("Order number", data.Number))
		return fmt.Errorf("failed to update order: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.logger.Info("failed to update order", zap.Error(err), zap.String("Order number", data.Number))
		return fmt.Errorf("failed to update order: %w", ErrOrderNotFound)
	}
	if rowsAffected == 0 {
		r.logger.Info("failed to update order", zap.String("Order number", data.Number))
		return fmt.Errorf("failed to update order: %w", ErrOrderNotFound)
	}

	insertTransactionQuery := `INSERT INTO transactions (user_id, "order", sum, processed_at) VALUES ($1, $2, $3, $4)`
	res, err = tx.ExecContext(ctx, insertTransactionQuery, data.UserID, data.Number, data.Accrual, time.Now().Format(time.RFC3339))
	if err != nil {
		r.logger.Info("failed to insert transaction", zap.Error(err), zap.String("Order number", data.Number), zap.String("user id", data.UserID))
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	if data.Accrual == 0 {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		return nil
	}

	var balance model.Balance
	selectBalanceQuery := "SELECT id, current FROM balance WHERE user_id = $1"
	err = tx.GetContext(ctx, &balance, selectBalanceQuery, data.UserID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			insertBalanceQuery := "INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, $2, $3)"
			_, err = tx.ExecContext(ctx, insertBalanceQuery, data.UserID, data.Accrual, 0)
			if err != nil {
				r.logger.Info("failed to create user balance at order update", zap.Error(err), zap.String("Order number", data.Number), zap.String("User ID", data.UserID))
				return fmt.Errorf("failed to create user balance: %w", err)
			}
		} else {
			r.logger.Info("failed to get user balance at order update", zap.Error(err), zap.String("Order number", data.Number), zap.String("User ID", data.UserID))
			return fmt.Errorf("failed to get user balance: %w", err)
		}
	} else {
		updateBalanceQuery := "UPDATE balance SET current = $1 WHERE user_id = $2"
		_, err = tx.ExecContext(ctx, updateBalanceQuery, data.Accrual+balance.Current, data.UserID)
		if err != nil {
			r.logger.Info("failed to update user balance at order update", zap.Error(err), zap.String("Order number", data.Number), zap.String("User ID", data.UserID))
			return fmt.Errorf("failed to update user balance: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		updateOrderStatusQuery := "UPDATE orders SET status = $1 WHERE number = $2"
		tx.ExecContext(ctx, updateOrderStatusQuery, "INVALID", data.Number)
		r.logger.Debug("failed to commit transaction", zap.Error(err), zap.String("Order number", data.Number), zap.String("User ID", data.UserID))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, data model.OrderStatusUpdateRequest) error {
	isStatusCorrect := slices.Contains(AvailableOrderStatus, data.Status)
	if !isStatusCorrect {
		r.logger.Debug("failed to update order", zap.Error(ErrUnavailableStatus), zap.String("Order status", data.Status))
		return ErrUnavailableStatus
	}
	updateQuery := "UPDATE orders SET status = $1 WHERE number = $2"
	res, err := r.db.ExecContext(ctx, updateQuery, "PROCESSING", data.Number)
	if err != nil {
		r.logger.Info("failed to update order status", zap.Error(err), zap.String("id", data.Number))
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.logger.Info("failed to update order status", zap.Error(err), zap.String("id", data.Number))
		return ErrOrderNotFound
	}
	if rowsAffected == 0 {
		r.logger.Info("failed to update order status", zap.String("id", data.Number))
		return ErrOrderNotFound
	}
	return nil
}
