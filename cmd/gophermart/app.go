package main

import (
	"context"
	"log"
	"net/http"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/logger"
	"go.uber.org/zap"
)

func run(cfg *config.Config) error {
	ctx := context.Background()

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
	return http.ListenAndServe(app.Config.RunAddress, nil)
}
