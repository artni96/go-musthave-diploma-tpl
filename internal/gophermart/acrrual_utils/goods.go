package acrrual_utils

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"go.uber.org/zap"
)

type Bill struct {
	OrderNumber string `json:"order"`
	Goods       []Good `json:"goods"`
}

type Good struct {
	Description string `json:"description"`
	Price       int    `json:"price"`
}

func (s *FileScanner) CollectGoodsData() ([]Good, error) {
	var result []Good
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := Good{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		result = append(result, object)

	}
	return result, nil
}

func GenerateBill(orderNumber string, logger *zap.Logger) Bill {
	filename := "data/goods.json"
	defaultBill := Bill{
		OrderNumber: orderNumber,
		Goods: []Good{
			{Description: "Чайник Bork", Price: 7000},
		},
	}

	scanner, err := NewFileScanner(filename)
	if err != nil {
		logger.Debug("file with goods not found", zap.Error(err), zap.String("filename", filename))
		return defaultBill
	}
	availableGoods, err := scanner.CollectGoodsData()
	if err != nil || availableGoods == nil {
		logger.Debug("failed to collect goods data", zap.Error(err), zap.String("filename", filename))
		return defaultBill
	}

	randGoods := getRandomGoods(availableGoods)

	bill := Bill{
		OrderNumber: orderNumber,
		Goods:       randGoods,
	}
	logger.Debug("successfully generated bill", zap.Any("order number", bill.OrderNumber), zap.Any("bill", bill))
	return bill
}

func getRandomGoods(availableGoods []Good) []Good {

	var goods []Good
	randAmount := rand.Intn((len(availableGoods) - 1) + 1)

	for i := 1; i <= randAmount; i++ {
		num := rand.Intn(len(availableGoods) - 1)
		goods = append(goods, availableGoods[num])
	}
	return goods
}
