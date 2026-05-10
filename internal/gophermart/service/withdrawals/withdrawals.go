package withdrawals

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/withdrawals"
)

type WithdrawalService struct {
	repository withdrawals.WithdrawalRepositoryInterface
	app        *config.App
}

type WithdrawalServiceInterface interface {
}

func NewWithdrawalService(repository withdrawals.WithdrawalRepositoryInterface, app *config.App) *WithdrawalService {
	return &WithdrawalService{
		repository: repository,
		app:        app,
	}
}

func (s *WithdrawalService) GetList(ctx context.Context, userID string) ([]model.WithdrawalResponse, error) {
	result, err := s.repository.GetList(ctx, userID)
	if err != nil {
		return nil, err
	}
	return result, nil
}
