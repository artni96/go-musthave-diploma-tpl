package balances

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
)

type BalanceService struct {
	repository balances.BalanceRepositoryInterface
	app        *config.App
}

type BalanceServiceInterface interface {
	Get(ctx context.Context, userID string) (model.BalanceResponse, error)
	Withdraw(ctx *context.Context, data model.TransactionCreate) error
}

func NewBalanceService(repository balances.BalanceRepositoryInterface, app *config.App) *BalanceService {
	return &BalanceService{
		repository: repository,
		app:        app,
	}
}

func (s *BalanceService) Get(ctx context.Context, userID string) (model.BalanceResponse, error) {
	result, err := s.repository.Get(ctx, userID)
	if err != nil {
		return model.BalanceResponse{}, err
	}
	return result, nil
}

func (s *BalanceService) Withdraw(ctx *context.Context, data model.TransactionCreate) error {
	data.Sum *= 100
	err := s.repository.Withdraw(ctx, data)
	if err != nil {
		return err
	}
	return nil
}
