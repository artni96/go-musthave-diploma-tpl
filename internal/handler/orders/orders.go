package orders

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	orderrepo "github.com/artni96/go-musthave-diploma-tpl/internal/repository/orders"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/service/orders"
	"github.com/artni96/go-musthave-diploma-tpl/internal/validators"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type OrderHandler struct {
	repository *orderrepo.OrderRepository
	service    *ordersserv.OrderService
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewOrderHandler(ctx *context.Context, app *config.App, repository *orderrepo.OrderRepository, service *ordersserv.OrderService) *OrderHandler {
	return &OrderHandler{
		repository: repository,
		service:    service,
		logger:     app.Logger,
		ctx:        ctx,
		cfg:        app.Config,
	}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	var OrderNumber int
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to read body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "invalid request body - number should be a digit", http.StatusBadRequest, h.logger, logMessage, zap.DebugLevel)
		return
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &OrderNumber)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to read body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "invalid request body - number should be a digit", http.StatusBadRequest, h.logger, logMessage, zap.DebugLevel)
		return
	}
	if OrderNumber == 0 {
		logMessage := logger.LogMessage{
			Message: "invalid request body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber)},
		}
		handler.ErrorResponse(w, "invalid request body - number should be a digit", http.StatusBadRequest, h.logger, logMessage, zap.DebugLevel)
		return
	}
	if !validators.LuhnValidator(OrderNumber) {
		logMessage := logger.LogMessage{
			Message: "invalid request body - order number is not luhn valid",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber)},
		}
		handler.ErrorResponse(w, "invalid order number", http.StatusUnprocessableEntity, h.logger, logMessage, zap.DebugLevel)
		return

	}
	newOrder := model.OrderCreateRequest{
		UserID: userID,
		Number: strconv.Itoa(OrderNumber),
	}
	_, err = h.service.Create(*h.ctx, newOrder)
	if err != nil {
		if errors.Is(err, orderrepo.ErrProcessingOrder) {
			logMessage := logger.LogMessage{
				Message: "failed to process order - order being processed",
				Fields:  []zap.Field{zap.Error(err), zap.Int("OrderNumber", OrderNumber)},
			}
			handler.ErrorResponse(w, "order being processed", http.StatusOK, h.logger, logMessage, zap.InfoLevel)
			return
		} else if errors.Is(err, orderrepo.ErrOrderAlreadyExists) {
			logMessage := logger.LogMessage{
				Message: "failed to process order - order already created by another user",
				Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber)},
			}
			handler.ErrorResponse(w, "order already created by another user", http.StatusConflict, h.logger, logMessage, zap.InfoLevel)
			return
		}
		logMessage := logger.LogMessage{
			Message: "failed to process order",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		statusCode, err := registerInAccrual(h.cfg, OrderNumber, h.logger)
		if err != nil {
			err = h.service.UpdateStatus(*h.ctx, model.OrderStatusUpdateRequest{
				Number: strconv.Itoa(OrderNumber),
				Status: "INVALID",
			})
			if err != nil {
				h.logger.Info("failed to change order status", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber), zap.String("status to change", "INVALID"))
				wg.Done()

			}
			h.logger.Info("failed to register in accrual", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber))
			wg.Done()
		}

		if statusCode == http.StatusAccepted {
			err = h.service.UpdateStatus(*h.ctx, model.OrderStatusUpdateRequest{
				Number: strconv.Itoa(OrderNumber),
				Status: "PROCESSING",
			})

			if err != nil {
				h.service.UpdateStatus(*h.ctx, model.OrderStatusUpdateRequest{
					Number: strconv.Itoa(OrderNumber),
					Status: "INVALID",
				})
				h.logger.Info("failed to change order status", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber), zap.String("status to change", "INVALID"))
				wg.Done()
			}
			orderData, err := checkOrderStatusInAccrual(h.cfg, OrderNumber, h.logger)
			if err != nil {
				h.logger.Info("failed to get order data", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber))
				wg.Done()
			}
			err = h.service.Update(*h.ctx, model.OrderUpdateRequest{
				Number:  orderData.Order,
				Status:  orderData.Status,
				Accrual: orderData.Accrual,
			})
			if err != nil {
				h.logger.Info("failed to update order", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber))
				wg.Done()
			}
		}

		defer wg.Done()
	}()
	wg.Wait()

}

type Bill struct {
	OrderNumber string `json:"order"`
	Goods       []Good `json:"goods"`
}

type Good struct {
	Description string `json:"description"`
	Price       int    `json:"price"`
}

type OrderAccrualResponse struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

func registerInAccrual(cfg *config.Config, orderNumber int, logger *zap.Logger) (int, error) {
	testBill := Bill{
		OrderNumber: strconv.Itoa(orderNumber),
		Goods: []Good{
			{Description: "Чайник Bork", Price: 7000},
		},
	}
	body, err := json.Marshal(testBill)
	if err != nil {
		logger.Debug("failed to marshal bill", zap.Error(err), zap.Int("orderNumber", orderNumber), zap.String("goods", fmt.Sprintf("%+v", testBill.Goods)))
		return 0, fmt.Errorf("failed to marshal body: %w", err)
	}
	reader := bytes.NewReader(body)
	registerOrderReq, err := http.Post(fmt.Sprintf("http://%s/api/orders", cfg.AccrualSystemAddress), "application/json", reader)
	if err != nil {
		logger.Debug("failed to register a new order via accrual system", zap.Error(err), zap.Int("orderNumber", orderNumber))
		return 0, fmt.Errorf("failed to register a new order via accrual system: %w", err)
	}
	defer registerOrderReq.Body.Close()
	return registerOrderReq.StatusCode, nil
}

func checkOrderStatusInAccrual(cfg *config.Config, orderNumber int, logger *zap.Logger) (OrderAccrualResponse, error) {
	var respBody OrderAccrualResponse
	orderStatusReq, err := http.Get(fmt.Sprintf("http://%s/api/orders/%d", cfg.AccrualSystemAddress, orderNumber))
	if err != nil {
		logger.Debug("failed to fetch order status", zap.Error(err))
		return respBody, err
	}
	defer orderStatusReq.Body.Close()
	respBodyBytes, err := io.ReadAll(orderStatusReq.Body)
	if err != nil {
		logger.Debug("failed to read order status", zap.Error(err))
		return respBody, err
	}

	err = json.Unmarshal(respBodyBytes, &respBody)
	if err != nil {
		logger.Debug("failed to read order status", zap.Error(err))
		return respBody, err
	}
	return respBody, nil
}

func OrderRouter(ctx *context.Context, app *config.App, repository *orderrepo.OrderRepository, service *ordersserv.OrderService) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	orderHandler := NewOrderHandler(ctx, app, repository, service)
	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
		r.Post("/", orderHandler.Create)
	})
	return r
}
