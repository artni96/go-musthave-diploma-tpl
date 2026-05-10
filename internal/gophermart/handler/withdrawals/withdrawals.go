package withdrawals

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/logger"
	transactionsrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/withdrawals"
	transactionsserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/withdrawals"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type WithdrawalHandler struct {
	repository transactionsrepo.WithdrawalRepositoryInterface
	service    transactionsserv.WithdrawalServiceInterface
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewWithdrawalHandler(ctx *context.Context, app *config.App, repository transactionsrepo.WithdrawalRepositoryInterface, service transactionsserv.WithdrawalServiceInterface) *WithdrawalHandler {
	return &WithdrawalHandler{
		ctx:        ctx,
		logger:     app.Logger,
		cfg:        app.Config,
		repository: repository,
		service:    service,
	}
}

// GetList godoc
// @Summary Getting user withdrawals list (authorization required)
// @Description list ordered by processed_at desc
// @Tags withdrawals
// @Accept json
// @Produce json
// @Success      200 {array} model.WithdrawalResponse
// @Success      204
// @Failure      401
// @Failure      500
// @Router       /api/user/withdrawals [get]
func (h *WithdrawalHandler) GetList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.repository.GetList(r.Context(), userID)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to get user withdrawals",
			Fields:  []zap.Field{zap.Error(err), zap.String("user_id", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp, err := json.Marshal(withdrawals)
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

func WithdrawalRouter(ctx *context.Context, app *config.App, repository transactionsrepo.WithdrawalRepositoryInterface, service transactionsserv.WithdrawalServiceInterface) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	withdrawalHandler := NewWithdrawalHandler(ctx, app, repository, service)
	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
		r.Get("/", withdrawalHandler.GetList)
	})
	return r
}
