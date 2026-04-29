package orders

import (
	"context"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository/orders"
)

type OrderService struct {
	repository *orders.OrderRepository
	app        *config.App
}

func NewOrderService(repository *orders.OrderRepository, app *config.App) *OrderService {
	return &OrderService{
		repository: repository,
		app:        app,
	}
}

func (s *OrderService) Create(ctx context.Context, order model.OrderCreateRequest) (string, error) {
	order.UploadedAt = time.Now().Format(time.RFC3339)
	orderID, err := s.repository.Create(ctx, order)
	if err != nil {
		return "", err
	}
	return orderID, nil
}
