package acrrual_utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

var ErrMechanicsFileNotFound = errors.New("file with metrics not found")
var ErrFailedToReadMechanicsFile = errors.New("failed to read file with metrics")

func (s *FileScanner) CollectMechanicsData() ([]Mechanic, error) {
	var result []Mechanic
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := Mechanic{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, fmt.Errorf("could not unmarshal object: %w", err)
		}
		result = append(result, object)

	}
	return result, nil
}

func UploadMechanics(filename string, accrualAddr string, logger *zap.Logger) error {
	accrualAddr = fmt.Sprintf("http://%s/api/goods", accrualAddr)
	scanner, err := NewFileScanner(filename)
	if err != nil {
		logger.Debug("file with mechanics not found", zap.Error(err), zap.String("filename", filename))
		return ErrMechanicsFileNotFound
	}
	mechanicsFromFile, err := scanner.CollectMechanicsData()
	if err != nil {
		logger.Debug("failed to collect mechanics data", zap.Error(err), zap.String("filename", filename))
		return ErrFailedToReadMechanicsFile
	}

	for _, mechanic := range mechanicsFromFile {
		body, err := json.Marshal(mechanic)
		if err != nil {
			logger.Debug("failed to marshal mechanic", zap.Error(err), zap.String("filename", filename), zap.String("mechanic match", mechanic.Match))
			continue
		}
		reader := bytes.NewReader(body)
		res, err := http.Post(accrualAddr, "application/json", reader)
		if err != nil {
			logger.Debug("failed to create mechanic request", zap.Error(err), zap.String("mechanic match", mechanic.Match))
		}
		if res.StatusCode == http.StatusOK {
			logger.Debug("mechanic successfully uploaded", zap.String("mechanic match", mechanic.Match))
		} else if res.StatusCode == http.StatusConflict {
			logger.Debug("mechanic already been uploaded", zap.String("mechanic match", mechanic.Match))
		}
	}
	return nil
}
