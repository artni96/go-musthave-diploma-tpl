package routers

import (
	"context"

	_ "github.com/artni96/go-musthave-diploma-tpl/api/docs"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/balances"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/orders"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/users"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/withdrawals"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	balancesrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	orderrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/orders"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/users"
	withdrawalsrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/withdrawals"
	balancesserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/balances"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/orders"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/users"
	withdrawalsserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/withdrawals"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func InitRouter(
	ctx *context.Context,
	app *config.App,
	userRepository *usersrepo.UserRepository,
	userService *usersserv.UserService,
	orderRepository *orderrepo.OrderRepository,
	orderService *ordersserv.OrderService,
	ordersQueue chan model.OrderQueue,
	balanceRepository balancesrepo.BalanceRepositoryInterface,
	balanceService balancesserv.BalanceServiceInterface,
	withdrawalRepository withdrawalsrepo.WithdrawalRepositoryInterface,
	withdrawalService withdrawalsserv.WithdrawalServiceInterface,
) *chi.Mux {
	mainRouter := chi.NewRouter()
	userRouter := users.UserRouter(ctx, app, userRepository, userService)
	orderRouter := orders.OrderRouter(ctx, app, orderRepository, orderService, ordersQueue)
	balanceRouter := balances.BalanceRouter(ctx, app, balanceRepository, balanceService)
	withdrawalRouter := withdrawals.WithdrawalRouter(ctx, app, withdrawalRepository, withdrawalService)

	mainRouter.Get("/swagger/*", httpSwagger.WrapHandler)
	mainRouter.Mount("/api/user", userRouter)
	mainRouter.Mount("/api/user/orders", orderRouter)
	mainRouter.Mount("/api/user/balance", balanceRouter)
	mainRouter.Mount("/api/user/withdrawals", withdrawalRouter)
	return mainRouter
}
