package transactions

import (
	"context"
	"fmt"
	"net/http"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/middlewares"
	transactionsrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/transactions"
	transactionsserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/transactions"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type TransactionHandler struct {
	repository transactionsrepo.TransactionRepositoryInterface
	service    transactionsserv.TransactionServiceInterface
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewTransactionHandler(ctx *context.Context, app *config.App, repository transactionsrepo.TransactionRepositoryInterface, service transactionsserv.TransactionServiceInterface) *TransactionHandler {
	return &TransactionHandler{
		ctx:        ctx,
		logger:     app.Logger,
		cfg:        app.Config,
		repository: repository,
		service:    service,
	}
}

func TransactionRouter(ctx *context.Context, app *config.App, repository transactionsrepo.TransactionRepositoryInterface, service transactionsserv.TransactionServiceInterface) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))
	r.Use(middlewares.AuthorizationMiddleware(app))

	//transactionHandler := NewTransactionHandler(ctx, app, service, repository)
	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
	})
	return r
}
