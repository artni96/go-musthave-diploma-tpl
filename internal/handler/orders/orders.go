package orders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type OrderHandler struct {
	logger *zap.Logger
	ctx    *context.Context
	cfg    *config.Config
}

func NewOrderHandler(ctx *context.Context, app *config.App) *OrderHandler {
	return &OrderHandler{
		logger: app.Logger,
		ctx:    ctx,
		cfg:    app.Config,
	}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	fmt.Println(OrderNumber)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func OrderRouter(ctx *context.Context, app *config.App) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	orderHandler := NewOrderHandler(ctx, app)
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
