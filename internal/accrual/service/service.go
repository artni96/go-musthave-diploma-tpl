package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Bill struct {
	OrderNumber string `json:"order"`
	Goods       []Good `json:"goods"`
}

type Good struct {
	Description string `json:"description"`
	Price       int    `json:"price"`
}

type OrderAccrualResponse struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

type FileScanner struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewFileScanner(filename string) (*FileScanner, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	return &FileScanner{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (s *FileScanner) Close() error {
	return s.file.Close()
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
