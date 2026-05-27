package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	orderrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/orders"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/orders"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/validators"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type OrderHandler struct {
	repository  orderrepo.OrderRepositoryInterface
	service     ordersserv.OrderServiceInterface
	logger      *zap.Logger
	ctx         *context.Context
	cfg         *config.Config
	ordersQueue chan model.OrderQueue
}

func NewOrderHandler(ctx *context.Context, app *config.App, repository orderrepo.OrderRepositoryInterface, service ordersserv.OrderServiceInterface, queue chan model.OrderQueue) *OrderHandler {
	return &OrderHandler{
		repository:  repository,
		service:     service,
		logger:      app.Logger,
		ctx:         ctx,
		cfg:         app.Config,
		ordersQueue: queue,
	}
}

// GetList godoc
// @Summary Getting user order list (authorization required)
// @Description possible statuses:
// @Description -NEW - order accepted, has not been processed yet;
// @Description -PROCESSING - order is being processed;
// @Description -INVALID - accrual system refused order;
// @Description -PROCESSED - order has been successfully processed
// @Tags orders
// @Accept json
// @Produce json
// @Success      200 {array} model.OrderResponse
// @Success      204
// @Failure      401
// @Failure      500
// @Router       /api/user/orders [get]
func (h *OrderHandler) GetList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := h.service.GetList(*h.ctx, userID)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "Failed to fetch orders",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp, err := json.Marshal(orders)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "Failed to marshal orders",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// Create order godoc
// @Summary Order registration (authorization required)
// @Description Order number has to be valid for Luhn algorithm
// @Tags orders
// @Accept text/plain
// @Produce json
// @Param request body integer true "Order registration"
// @Success      200 "Order being processed"
// @Success      202
// @Failure      400
// @Failure      401
// @Failure      409 "Order already created by another user"
// @Failure      422 "Invalid order number"
// @Failure      500
// @Router       /api/user/orders [post]
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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

	h.ordersQueue <- model.OrderQueue{
		UserID: userID,
		Number: OrderNumber,
	}
}

func OrderRouter(ctx *context.Context, app *config.App, repository *orderrepo.OrderRepository, service *ordersserv.OrderService, ordersQueue chan model.OrderQueue) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	orderHandler := NewOrderHandler(ctx, app, repository, service, ordersQueue)
	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
		r.Get("/", orderHandler.GetList)
		r.Post("/", orderHandler.Create)
	})
	return r
}
