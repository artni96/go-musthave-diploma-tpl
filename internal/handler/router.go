package handler

import (
	"context"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"github.com/artni96/go-musthave-diploma-tpl/internal/service"
	"github.com/go-chi/chi/v5"
)

func InitRouter(ctx *context.Context, app *config.App, cfg *config.Config, userRepository *repository.UserRepository, userService *service.UserService) *chi.Mux {
	mainRouter := chi.NewRouter()
	userRouter := UserRouter(ctx, app, cfg, userRepository, userService)
	mainRouter.Mount("/", userRouter)
	return mainRouter
}
