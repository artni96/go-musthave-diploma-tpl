package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	au "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/acrrual_utils"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/routers"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	balancesrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/balances"
	ordersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/orders"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/users"
	withdrawalsrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/withdrawals"
	balancesserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/balances"
	ordersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/orders"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/users"
	withdrawalsserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/withdrawals"
	"go.uber.org/zap"

	_ "github.com/artni96/go-musthave-diploma-tpl/api/docs"
)

func run(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := config.InitDBConnection(ctx, cfg, true)
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

	userRepository := usersrepo.NewUserRepository(db, app.Logger)
	userService := usersserv.NewUserService(userRepository, &app)

	orderRepository := ordersrepo.NewOrderRepository(db, app.Logger)
	orderService := ordersserv.NewOrderService(orderRepository, &app)
	ordersQueue := make(chan model.OrderQueue, 100)

	balanceRepository := balancesrepo.NewBalanceRepository(db, app.Logger)
	balanceService := balancesserv.NewBalanceService(balanceRepository, &app)

	withdrawalRepository := withdrawalsrepo.NewWithdrawalRepository(db, app.Logger)
	withdrawalService := withdrawalsserv.NewWithdrawalService(withdrawalRepository, &app)
	mainRouter := routers.InitRouter(&ctx, &app, userRepository, userService, orderRepository, orderService, ordersQueue, balanceRepository, balanceService, withdrawalRepository, withdrawalService)

	newServer := &http.Server{
		Addr:    app.Config.RunAddress,
		Handler: mainRouter,
	}

	go func() {

		err = au.UploadMechanics("data/mechanics.json", cfg.AccrualSystemAddress, app.Logger)
		if err != nil {
			app.Logger.Info("failed to upload mechanics", zap.Error(err))
		}

		app.Logger.Info("launching gophermart server", zap.String("server address", cfg.RunAddress))
		err = newServer.ListenAndServe()
		if err != nil {
			app.Logger.Fatal("failed to launch gophermart server", zap.Error(err))
		}
	}()

	go func() {
		wg := ordersserv.WorkersPool(&ctx, orderService, ordersQueue, &app)
		defer wg.Wait()
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
