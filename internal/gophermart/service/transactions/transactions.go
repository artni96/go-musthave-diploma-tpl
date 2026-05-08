package transactions

import (
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/transactions"
)

type TransactionService struct {
	repository transactions.TransactionRepositoryInterface
	app        *config.App
}

type TransactionServiceInterface interface {
}

func NewTransactionService(repository transactions.TransactionRepositoryInterface, app *config.App) *TransactionService {
	return &TransactionService{
		repository: repository,
		app:        app,
	}
}
