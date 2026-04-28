package routers

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/orders"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/users"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"github.com/artni96/go-musthave-diploma-tpl/internal/service"
	"github.com/go-chi/chi/v5"
)

func InitRouter(ctx *context.Context, app *config.App, userRepository *repository.UserRepository, userService *service.UserService) *chi.Mux {
	mainRouter := chi.NewRouter()
	userRouter := users.UserRouter(ctx, app, userRepository, userService)
	orderRouter := orders.OrderRouter(ctx, app)

	mainRouter.Mount("/api/user", userRouter)
	mainRouter.Mount("/api/user/orders", orderRouter)
	return mainRouter
}
