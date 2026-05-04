package routers

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/balances"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/orders"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/users"
	balancesrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	orderrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/orders"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/users"
	balancesserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/balances"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/orders"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/users"
	"github.com/go-chi/chi/v5"
)

func InitRouter(
	ctx *context.Context,
	app *config.App,
	userRepository *usersrepo.UserRepository,
	userService *usersserv.UserService,
	orderRepository *orderrepo.OrderRepository,
	orderService *ordersserv.OrderService,
	ordersQueue chan string,
	balanceRepository balancesrepo.BalanceRepositoryInterface,
	balanceService balancesserv.BalanceServiceInterface,
) *chi.Mux {
	mainRouter := chi.NewRouter()
	userRouter := users.UserRouter(ctx, app, userRepository, userService)
	orderRouter := orders.OrderRouter(ctx, app, orderRepository, orderService, ordersQueue)
	balanceRouter := balances.BalanceRouter(ctx, app, balanceRepository, balanceService)

	mainRouter.Mount("/api/user", userRouter)
	mainRouter.Mount("/api/user/orders", orderRouter)
	mainRouter.Mount("/api/user/balance", balanceRouter)
	return mainRouter
}
