package balances

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/logger"
	balancesrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	balancesserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/balances"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type BalanceHandler struct {
	repository balancesrepo.BalanceRepositoryInterface
	service    balancesserv.BalanceServiceInterface
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewBalanceHandler(ctx context.Context, repository balancesrepo.BalanceRepositoryInterface, service balancesserv.BalanceServiceInterface, logger *zap.Logger, cfg *config.Config) *BalanceHandler {
	return &BalanceHandler{
		repository: repository,
		service:    service,
		logger:     logger,
		ctx:        &ctx,
		cfg:        cfg,
	}
}

func (h *BalanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	balance, err := h.service.Get(*h.ctx, userID)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "Error getting balance",
			Fields:  []zap.Field{zap.Error(err), zap.String("userID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
	}

	resp, err := json.Marshal(balance)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "Failed to marshal response",
			Fields:  []zap.Field{zap.Error(err), zap.String("userID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)

	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func BalanceRouter(ctx *context.Context, app *config.App, repository balancesrepo.BalanceRepositoryInterface, service balancesserv.BalanceServiceInterface) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	balanceHandler := NewBalanceHandler(*ctx, repository, service, app.Logger, app.Config)
	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
		r.Get("/", balanceHandler.Get)
	})
	return r
}
