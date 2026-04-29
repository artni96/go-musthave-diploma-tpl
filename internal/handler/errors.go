package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/artni96/go-musthave-diploma-tpl/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ErrorResponseStruct struct {
	Error string `json:"error"`
}

type ListErrorResponseStruct struct {
	Error []string `json:"error"`
}

func ErrorResponse(w http.ResponseWriter, errMessage string, statusCode int, logger *zap.Logger, logMessage logger.LogMessage, loggerLevel zapcore.Level) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if loggerLevel == zapcore.DebugLevel {
		logger.Debug(logMessage.Message, logMessage.Fields...)
	} else if loggerLevel == zapcore.InfoLevel {
		logger.Info(logMessage.Message, logMessage.Fields...)
	} else if loggerLevel == zapcore.WarnLevel {
		logger.Warn(logMessage.Message, logMessage.Fields...)
	} else if loggerLevel == zapcore.ErrorLevel {
		logger.Error(logMessage.Message, logMessage.Fields...)
	} else if loggerLevel == zapcore.FatalLevel {
		logger.Fatal(logMessage.Message, logMessage.Fields...)
	} else if loggerLevel == zapcore.PanicLevel {
		logger.Panic(logMessage.Message, logMessage.Fields...)
	}

	if statusCode >= 500 {
		resp, err := json.Marshal(&ErrorResponseStruct{Error: errMessage})
		if err != nil {
			w.Write([]byte(`{"error": "Server error. Please try again."}`))
			return
		}
		w.Write(resp)
		return

	}

	splitMessage := strings.Split(errMessage, "\n")
	if len(splitMessage) > 1 {
		var errs []string
		for _, line := range splitMessage {
			errs = append(errs, line)
		}
		resp, err := json.Marshal(ListErrorResponseStruct{errs})
		if err != nil {
			w.Write([]byte(`{"error": "invalid request"}`))
			return
		}
		w.Write(resp)
		return
	}
	errResp := ErrorResponseStruct{
		Error: errMessage,
	}
	resp, err := json.Marshal(errResp)
	if err != nil {
		w.Write([]byte(`{"error": "invalid request"}`))
		return
	}
	w.Write(resp)
	return
}
