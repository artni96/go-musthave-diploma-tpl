package orders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"sync"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/acrrual_utils"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	"go.uber.org/zap"
)

type OrderAccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func WorkersPool(ctx *context.Context, orderService OrderServiceInterface, orderQueue <-chan model.OrderQueue, app *config.App) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := 0; i < runtime.NumCPU(); i++ {
		app.Logger.Debug("worker waiting for task", zap.Int("Worker number", i))
		wg.Go(func() { orderWorker(ctx, orderService, orderQueue, app) })
	}
	return &wg
}

func orderWorker(ctx *context.Context, service OrderServiceInterface, orderQueue <-chan model.OrderQueue, app *config.App) {

	for order := range orderQueue {
		orderNumber := order.Number
		userID := order.UserID

		app.Logger.Debug("processing order", zap.Int("orderNumber", orderNumber))
		statusCode, err := registerInAccrual(app.Config, orderNumber, app.Logger)
		if err != nil {
			err = service.UpdateStatus(*ctx, model.OrderStatusUpdateRequest{
				Number: strconv.Itoa(orderNumber),
				Status: "INVALID",
			})
			if err != nil {
				app.Logger.Debug("failed to change order status", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber), zap.String("status to change", "INVALID"))
				return
			}
			app.Logger.Debug("failed to register in accrual", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
			return
		}

		if statusCode == http.StatusAccepted {
			app.Logger.Debug("successfully registered in accrual system", zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
			err = service.UpdateStatus(*ctx, model.OrderStatusUpdateRequest{
				Number: strconv.Itoa(orderNumber),
				Status: "PROCESSING",
			})

			if err != nil {
				service.UpdateStatus(*ctx, model.OrderStatusUpdateRequest{
					Number: strconv.Itoa(orderNumber),
					Status: "INVALID",
				})
				app.Logger.Debug("failed to change order status", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber), zap.String("status to change", "INVALID"))
				return
			}
			orderData, err := checkOrderStatusInAccrual(app.Config, orderNumber, app.Logger)
			if err != nil {
				app.Logger.Debug("failed to get order data", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
				return
			}
			app.Logger.Debug("order data", zap.String("order number", orderData.Order), zap.String("status", orderData.Status), zap.Float64("accrual", orderData.Accrual))
			err = service.Update(*ctx, model.OrderUpdateRequest{
				Number:  orderData.Order,
				Status:  orderData.Status,
				Accrual: int(orderData.Accrual * 100),
				UserID:  userID,
			})
			if err != nil {
				app.Logger.Debug("failed to update order", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
				return
			}
			app.Logger.Debug("order successfully handled", zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber), zap.String("status", "PROCESSING"))
		} else {
			app.Logger.Debug("failed to handle order", zap.Int("order number", order.Number))
		}
	}
}

func registerInAccrual(cfg *config.Config, orderNumber int, logger *zap.Logger) (int, error) {
	bill := acrrual_utils.GenerateBill(strconv.Itoa(orderNumber), logger)
	body, err := json.Marshal(bill)
	if err != nil {
		logger.Debug("failed to marshal bill", zap.Error(err), zap.Int("orderNumber", orderNumber), zap.String("goods", fmt.Sprintf("%+v", bill.Goods)))
		return 0, fmt.Errorf("failed to marshal body: %w", err)
	}

	test, err := http.Get(fmt.Sprintf("%s/api/orders/%d", cfg.AccrualSystemAddress, orderNumber))
	if err != nil {
		logger.Debug("failed to fetch orders", zap.Error(err), zap.Int("orderNumber", orderNumber), zap.Int("status", test.StatusCode))
	}
	defer test.Body.Close()
	testBody, err := io.ReadAll(test.Body)
	if err != nil {
		logger.Debug("failed to read body", zap.Error(err), zap.Int("orderNumber", orderNumber))
	}
	logger.Debug("successfully fetched orders", zap.Int("orderNumber", orderNumber), zap.String("body", string(testBody)))

	reader := bytes.NewReader(body)
	accrualRegisterOrderURL := fmt.Sprintf("%s/api/orders", cfg.AccrualSystemAddress)
	registerOrderReq, err := http.Post(accrualRegisterOrderURL, "application/json", reader)
	if err != nil {
		logger.Debug("failed to register a new order via accrual system", zap.Error(err), zap.Int("orderNumber", orderNumber))
		return 0, fmt.Errorf("failed to register a new order via accrual system: %w", err)
	}
	logger.Debug("successful post request to `/api/orders`", zap.Int("orderNumber", orderNumber))
	defer registerOrderReq.Body.Close()
	logger.Debug("order successfully registered in accrual system", zap.Int("orderNumber", orderNumber), zap.String("goods", fmt.Sprintf("%+v", bill.Goods)), zap.Int("status code", registerOrderReq.StatusCode))
	return registerOrderReq.StatusCode, nil
}

func checkOrderStatusInAccrual(cfg *config.Config, orderNumber int, logger *zap.Logger) (OrderAccrualResponse, error) {
	var respBody OrderAccrualResponse
	orderStatusReq, err := http.Get(fmt.Sprintf("%s/api/orders/%d", cfg.AccrualSystemAddress, orderNumber))
	if err != nil {
		logger.Debug("failed to fetch order status", zap.Error(err))
		return respBody, err
	}
	defer orderStatusReq.Body.Close()
	respBodyBytes, err := io.ReadAll(orderStatusReq.Body)
	if err != nil {
		logger.Debug("failed to read order status", zap.Error(err))
		return respBody, err
	}

	err = json.Unmarshal(respBodyBytes, &respBody)
	if err != nil {
		logger.Debug("failed to read order status", zap.Error(err))
		return respBody, err
	}
	return respBody, nil
}
