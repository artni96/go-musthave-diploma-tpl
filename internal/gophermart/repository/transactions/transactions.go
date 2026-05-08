package transactions

import (
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var ErrNotEnoughMoney = errors.New("not enough money")

type TransactionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type TransactionRepositoryInterface interface {
}

func NewTransactionRepository(db *sqlx.DB, logger *zap.Logger) *TransactionRepository {
	return &TransactionRepository{
		db:     db,
		logger: logger,
	}
}
