package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"github.com/artni96/go-musthave-diploma-tpl/internal/service"
	"go.uber.org/zap"
)

func run(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := config.InitDBConnection(ctx, cfg)
	if err != nil {
		return err
	}

	appLogger, err := logger.InitLogger(cfg.Debug)
	if err != nil {
		log.Fatal("failed to initialize logger")
	}

	app := config.App{
		Config: cfg,
		DB:     db,
		Logger: appLogger,
	}
	app.Logger.Info("starting server", zap.String("server address", cfg.RunAddress))

	userRepository := repository.NewUserRepository(db, app.Logger)
	userService := service.NewUserService(userRepository, &app)
	mainRouter := handler.InitRouter(&ctx, &app, cfg, userRepository, userService)

	newServer := &http.Server{
		Addr:    app.Config.RunAddress,
		Handler: mainRouter,
	}

	go func() {
		err = newServer.ListenAndServe()
		if err != nil {
			app.Logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownChan
	app.Logger.Info("shutting app down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err = db.Close(); err != nil {
		app.Logger.Info("failed to close database", zap.Error(err))
	} else {
		app.Logger.Info("database connection gracefully closed")
	}

	if err = newServer.Shutdown(shutdownCtx); err != nil {
		app.Logger.Info("failed to shutdown server", zap.Error(err))
	} else {
		app.Logger.Info("server stopped gracefully")
	}
	app.Logger.Info("app stopped gracefully")
	return nil
}
