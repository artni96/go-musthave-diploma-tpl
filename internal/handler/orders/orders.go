package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

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
		UserID:      userID,
		OrderNumber: strconv.Itoa(OrderNumber),
	}
	_, err = h.service.Create(*h.ctx, newOrder)
	if err != nil {
		if errors.Is(err, orderrepo.ErrProcessingOrder) {
			logMessage := logger.LogMessage{
				Message: "failed to process order - order being processed",
				Fields:  []zap.Field{zap.Error(err), zap.Int("OrderNumber", OrderNumber)},
			}
			handler.ErrorResponse(w, err.Error(), http.StatusAccepted, h.logger, logMessage, zap.InfoLevel)
			return
		} else if errors.Is(err, orderrepo.ErrOrderAlreadyExists) {
			logMessage := logger.LogMessage{
				Message: "failed to process order - order already created by another user",
				Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", OrderNumber)},
			}
			handler.ErrorResponse(w, err.Error(), http.StatusConflict, h.logger, logMessage, zap.InfoLevel)
			return
		}
		logMessage := logger.LogMessage{
			Message: "failed to process order",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return
	}
	w.WriteHeader(http.StatusCreated)
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
