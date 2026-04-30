package orders

import (
	"context"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository/orders"
)

type OrderService struct {
	repository orders.OrderRepositoryInterface
	app        *config.App
}

type OrderServiceInterface interface {
	Create(ctx context.Context, order model.OrderCreateRequest) (string, error)
	Update(ctx context.Context, order model.OrderUpdateRequest) error
	UpdateStatus(ctx context.Context, order model.OrderStatusUpdateRequest) error
}

func NewOrderService(repository orders.OrderRepositoryInterface, app *config.App) *OrderService {
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

func (s *OrderService) Update(ctx context.Context, order model.OrderUpdateRequest) error {
	err := s.repository.Update(ctx, order)
	if err != nil {
		return err
	}
	return nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, order model.OrderStatusUpdateRequest) error {
	err := s.repository.UpdateStatus(ctx, order)
	if err != nil {
		return err
	}
	return nil
}
