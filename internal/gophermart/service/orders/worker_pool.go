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
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

func WorkersPool(ctx *context.Context, orderService OrderServiceInterface, orderQueue <-chan model.OrderQueue, app *config.App) {
	var wg sync.WaitGroup

	for i := 1; i < runtime.NumCPU(); i++ {
		wg.Go(func() { orderWorker(ctx, orderService, &wg, orderQueue, app) })
	}
}

func orderWorker(ctx *context.Context, service OrderServiceInterface, wg *sync.WaitGroup, orderQueue <-chan model.OrderQueue, app *config.App) {
	defer wg.Done()

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
				app.Logger.Info("failed to change order status", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber), zap.String("status to change", "INVALID"))
				return
			}
			app.Logger.Info("failed to register in accrual", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
			return
		}

		if statusCode == http.StatusAccepted {
			err = service.UpdateStatus(*ctx, model.OrderStatusUpdateRequest{
				Number: strconv.Itoa(orderNumber),
				Status: "PROCESSING",
			})

			if err != nil {
				service.UpdateStatus(*ctx, model.OrderStatusUpdateRequest{
					Number: strconv.Itoa(orderNumber),
					Status: "INVALID",
				})
				app.Logger.Info("failed to change order status", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber), zap.String("status to change", "INVALID"))
				return
			}
			orderData, err := checkOrderStatusInAccrual(app.Config, orderNumber, app.Logger)
			if err != nil {
				app.Logger.Info("failed to get order data", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
				return
			}
			err = service.Update(*ctx, model.OrderUpdateRequest{
				Number:  orderData.Order,
				Status:  orderData.Status,
				Accrual: orderData.Accrual * 100,
				UserID:  userID,
			})
			if err != nil {
				app.Logger.Info("failed to update order", zap.Error(err), zap.String("UserID", userID), zap.Int("OrderNumber", orderNumber))
				return
			}
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
	reader := bytes.NewReader(body)
	registerOrderReq, err := http.Post(fmt.Sprintf("http://%s/api/orders", cfg.AccrualSystemAddress), "application/json", reader)
	if err != nil {
		logger.Debug("failed to register a new order via accrual system", zap.Error(err), zap.Int("orderNumber", orderNumber))
		return 0, fmt.Errorf("failed to register a new order via accrual system: %w", err)
	}
	defer registerOrderReq.Body.Close()
	return registerOrderReq.StatusCode, nil
}

func checkOrderStatusInAccrual(cfg *config.Config, orderNumber int, logger *zap.Logger) (OrderAccrualResponse, error) {
	var respBody OrderAccrualResponse
	orderStatusReq, err := http.Get(fmt.Sprintf("http://%s/api/orders/%d", cfg.AccrualSystemAddress, orderNumber))
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
