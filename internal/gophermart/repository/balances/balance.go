package balances

import (
	"context"
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type BalanceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type BalanceRepositoryInterface interface {
	Get(ctx context.Context, userID string) (model.BalanceResponse, error)
}

func NewBalanceRepository(db *sqlx.DB, logger *zap.Logger) *BalanceRepository {
	return &BalanceRepository{
		db:     db,
		logger: logger,
	}
}

func (repo *BalanceRepository) Get(ctx context.Context, userID string) (model.BalanceResponse, error) {
	var userBalance model.BalanceResponse
	selectQuery := "SELECT (current/100) as current, (withdrawn/100) as withdrawn FROM balance WHERE user_id = $1"
	err := repo.db.GetContext(ctx, &userBalance, selectQuery, userID)
	if err != nil {
		repo.logger.Info("failed to get balances", zap.String("user_id", userID), zap.Error(err))
		return userBalance, fmt.Errorf("failed to get user balances: %w", err)
	}
	return userBalance, nil
}
