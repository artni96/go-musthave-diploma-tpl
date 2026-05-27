package withdrawals

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type WithdrawalRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type WithdrawalRepositoryInterface interface {
	GetList(ctx context.Context, userID string) ([]model.WithdrawalResponse, error)
}

func NewWithdrawalRepository(db *sqlx.DB, logger *zap.Logger) *WithdrawalRepository {
	return &WithdrawalRepository{
		db:     db,
		logger: logger,
	}
}

func (r *WithdrawalRepository) GetList(ctx context.Context, userID string) ([]model.WithdrawalResponse, error) {
	var result []model.WithdrawalResponse
	selectQuery := `SELECT "order", abs(sum::numeric / 100) as sum, processed_at FROM transactions WHERE user_id = $1 AND sum < 0 order by processed_at desc `
	err := r.db.SelectContext(ctx, &result, selectQuery, userID)
	if err != nil {
		r.logger.Debug("failed to get user withdrawals", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	r.logger.Debug("got user withdrawals", zap.String("userID", userID))
	return result, nil
}
