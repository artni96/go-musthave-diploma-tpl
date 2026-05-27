package balances

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	balancesrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	balancesserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/balances"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/validators"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type BalanceHandler struct {
	repository balancesrepo.BalanceRepositoryInterface
	service    balancesserv.BalanceServiceInterface
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewBalanceHandler(ctx context.Context, app *config.App, repository balancesrepo.BalanceRepositoryInterface, service balancesserv.BalanceServiceInterface) *BalanceHandler {
	return &BalanceHandler{
		repository: repository,
		service:    service,
		logger:     app.Logger,
		ctx:        &ctx,
		cfg:        app.Config,
	}
}

// Get godoc
// @Summary Getting user balance (authorization required)
// @Description Getting user balance:
// @Tags balance
// @Accept json
// @Produce json
// @Success      200 {object} model.BalanceResponse
// @Failure      401
// @Failure      500
// @Router       /api/user/balance [get]
func (h *BalanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	balance, err := h.service.Get(h.ctx, userID)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "Error getting balance",
			Fields:  []zap.Field{zap.Error(err), zap.String("userID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return
	}

	resp, err := json.Marshal(balance)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "Failed to marshal response",
			Fields:  []zap.Field{zap.Error(err), zap.String("userID", userID)},
		}
		handler.ErrorResponse(w, "Internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.InfoLevel)
		return

	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// Withdraw godoc
// @Summary Withdraw bonuses for order (authorization required)
// @Description Withdraw bonuses for order:
// @Tags balance
// @Accept json
// @Produce json
// @Param request body model.TransactionCreateRequest true "Withdraw bonuses for order"
// @Success      200
// @Failure      401
// @Failure      402 "Not enough money"
// @Failure      422 "Invalid order number"
// @Failure      500
// @Router       /api/user/balance/withdraw [post]
func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := r.Header.Get("UserID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to read body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusUnprocessableEntity, h.logger, logMessage, zap.DebugLevel)
	}
	defer r.Body.Close()
	var transaction model.TransactionCreate
	err = json.Unmarshal(body, &transaction)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to unmarshal body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusUnprocessableEntity, h.logger, logMessage, zap.DebugLevel)
	}
	if transaction.Order == "0" {
		logMessage := logger.LogMessage{
			Message: "invalid request body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.String("OrderNumber", transaction.Order)},
		}
		handler.ErrorResponse(w, "invalid request body - number should be a digit", http.StatusUnprocessableEntity, h.logger, logMessage, zap.DebugLevel)
		return
	}

	digitOrder, err := strconv.Atoi(transaction.Order)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "invalid request body",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
		}
		handler.ErrorResponse(w, "invalid request body - number should be a digit", http.StatusUnprocessableEntity, h.logger, logMessage, zap.DebugLevel)
		return
	}

	if !validators.LuhnValidator(digitOrder) {
		logMessage := logger.LogMessage{
			Message: "invalid request body - order number is not luhn valid",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.String("OrderNumber", transaction.Order)},
		}
		handler.ErrorResponse(w, "invalid order number", http.StatusUnprocessableEntity, h.logger, logMessage, zap.DebugLevel)
		return

	}

	transaction.UserID = userID
	transaction.ProcessedAt = time.Now().Format(time.RFC3339)

	err = h.service.Withdraw(h.ctx, transaction)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, balancesrepo.ErrNotEnoughMoney) {
			logMessage := logger.LogMessage{
				Message: err.Error(),
				Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID)},
			}
			handler.ErrorResponse(w, err.Error(), http.StatusPaymentRequired, h.logger, logMessage, zap.DebugLevel)
			return
		} else if errors.As(err, &pgErr) {
			logMessage := logger.LogMessage{
				Message: "order already has transaction",
				Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.String("OrderNumber", transaction.Order)},
			}
			handler.ErrorResponse(w, "order already has transaction", http.StatusPaymentRequired, h.logger, logMessage, zap.DebugLevel)
			return
		}
		logMessage := logger.LogMessage{
			Message: "failed to withdraw bonuses for order",
			Fields:  []zap.Field{zap.Error(err), zap.String("UserID", userID), zap.String("OrderNumber", transaction.Order), zap.Float64("Sum", transaction.Sum)},
		}
		handler.ErrorResponse(w, "internal server error", http.StatusInternalServerError, h.logger, logMessage, zap.DebugLevel)
		return
	}
}

func BalanceRouter(ctx *context.Context, app *config.App, repository balancesrepo.BalanceRepositoryInterface, service balancesserv.BalanceServiceInterface) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	balanceHandler := NewBalanceHandler(*ctx, app, repository, service)
	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
		r.Get("/", balanceHandler.Get)
		r.Post("/withdraw", balanceHandler.Withdraw)
	})
	return r
}
