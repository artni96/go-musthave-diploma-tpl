package balances

import (
	"context"
	"errors"
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ErrNotEnoughMoney = errors.New("not enough money")

type BalanceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type BalanceRepositoryInterface interface {
	Get(ctx *context.Context, userID string) (model.BalanceResponse, error)
	Withdraw(ctx *context.Context, data model.TransactionCreate) error
}

func NewBalanceRepository(db *sqlx.DB, logger *zap.Logger) *BalanceRepository {
	return &BalanceRepository{
		db:     db,
		logger: logger,
	}
}

func (r *BalanceRepository) Get(ctx *context.Context, userID string) (model.BalanceResponse, error) {
	var userBalance model.BalanceResponse
	selectQuery := "SELECT (current::numeric/100) as current, (withdrawn::numeric/100) as transactions FROM balance WHERE user_id = $1"
	err := r.db.GetContext(*ctx, &userBalance, selectQuery, userID)
	if err != nil {
		r.logger.Debug("failed to get balances", zap.String("user_id", userID), zap.Error(err))
		return userBalance, fmt.Errorf("failed to get user balances: %w", err)
	}
	return userBalance, nil
}

func (r *BalanceRepository) Withdraw(ctx *context.Context, data model.TransactionCreate) error {

	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	var currentBalance float64
	currentBalanceQuery := "SELECT current FROM balance where user_id = $1"

	err = tx.GetContext(*ctx, &currentBalance, currentBalanceQuery, data.UserID)
	if err != nil {
		return fmt.Errorf("failed to get current user balance: %w", err)
	}

	if currentBalance < data.Sum {
		r.logger.Debug("not enough money", zap.Float64("current balance", currentBalance), zap.Float64("to withdraw", data.Sum), zap.String("user_id", data.UserID))
		return ErrNotEnoughMoney
	}

	insertTransactionRequest := `INSERT INTO transactions (user_id, "order", sum, processed_at) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(*ctx, insertTransactionRequest, data.UserID, data.Order, -data.Sum, data.ProcessedAt)
	if err != nil {
		r.logger.Debug("failed to create new transaction error", zap.Error(err), zap.String("user_id", data.UserID), zap.String("order", data.Order), zap.Float64("sum", data.Sum))
		return fmt.Errorf("failed to create new transaction error: %w", err)
	}

	updateBalanceRequest := "UPDATE balance SET current=$1, withdrawn=$2 WHERE user_id = $3"
	_, err = tx.ExecContext(*ctx, updateBalanceRequest, currentBalance-data.Sum, data.Sum, data.UserID)
	if err != nil {
		r.logger.Debug("failed to update user balance after making transaction", zap.Error(err), zap.String("user_id", data.UserID))
		return fmt.Errorf("failed to update user balance: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
