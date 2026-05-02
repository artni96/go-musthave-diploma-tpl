package orders

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ErrOrderAlreadyExists = errors.New("order already exists")
var ErrProcessingOrder = errors.New("order being processed")
var ErrOrderNotFound = errors.New("order not found")
var ErrUnavailableStatus = errors.New("status is unavailable")

var AvailableOrderStatus = []string{"PROCESSING", "PROCESSED", "REGISTERED", "INVALID"}

type OrderRepositoryInterface interface {
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

func (r *OrderRepository) Create(ctx context.Context, inputOrder model.OrderCreateRequest) (string, error) {
	var order model.Order
	insertQuery := "INSERT INTO orders (user_id, number, uploaded_at, accrual, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, number, accrual, status, uploaded_at"
	err := r.db.GetContext(ctx, &order, insertQuery, inputOrder.UserID, inputOrder.Number, inputOrder.UploadedAt, 0, "REGISTERED")

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
		return fmt.Errorf("BulkCreate - failure to begin transaction: %w", err)
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
	
	if data.Accrual == 0 {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit bulk create: %w", err)
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
		return fmt.Errorf("failed to commit bulk create: %w", err)
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
