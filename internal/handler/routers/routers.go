package routers

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/orders"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/users"
	orderrepo "github.com/artni96/go-musthave-diploma-tpl/internal/repository/orders"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/repository/users"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/service/orders"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/service/users"
	"github.com/go-chi/chi/v5"
)

func InitRouter(ctx *context.Context, app *config.App, userRepository *usersrepo.UserRepository, userService *usersserv.UserService, orderRepository *orderrepo.OrderRepository, orderService *ordersserv.OrderService, ordersQueue chan string, transactionsQueue chan string) *chi.Mux {
	mainRouter := chi.NewRouter()
	userRouter := users.UserRouter(ctx, app, userRepository, userService)
	orderRouter := orders.OrderRouter(ctx, app, orderRepository, orderService, ordersQueue, transactionsQueue)

	mainRouter.Mount("/api/user", userRouter)
	mainRouter.Mount("/api/user/orders", orderRouter)
	return mainRouter
}
