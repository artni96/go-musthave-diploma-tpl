package orders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/middlewares"
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
	fmt.Println(userID)
	var OrderNumber int
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Warn("failed to read body", zap.Error(err))
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &OrderNumber)
	if err != nil {
		h.logger.Warn("failed to unmarshal body", zap.Error(err))
	}
	if !validators.LuhnValidator(OrderNumber) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid order number"))
		return
	}
	newOrder := model.OrderCreateRequest{
		UserID:      userID,
		OrderNumber: strconv.Itoa(OrderNumber),
	}
	orderID, err := h.service.Create(*h.ctx, newOrder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(orderID))
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
