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
