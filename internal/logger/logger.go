package logger

import (
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)
	r.responseData.status = status
}

func RequestLoggerMiddleware(logger *zap.Logger) func(h http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logFn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			responseData := &responseData{
				status: 0,
				size:   0,
			}
			lw := loggingResponseWriter{
				ResponseWriter: w,
				responseData:   responseData,
			}
			next.ServeHTTP(&lw, r)

			duration := time.Since(start)

			logger.Info("Request done",
				zap.String("URI", r.RequestURI),
				zap.String("method", r.Method),
				zap.Int("duration", int(duration)),
				zap.Int("response_status", responseData.status),
				zap.Int("response_size", responseData.size),
			)

		}
		return http.HandlerFunc(logFn)
	}
}

func InitLogger(debug bool) (*zap.Logger, error) {
	logLevel := zapcore.InfoLevel
	if debug {
		logLevel = zapcore.DebugLevel
	}

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	logFileConfig := zap.NewDevelopmentEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(logFileConfig)

	logFile, err := os.OpenFile("./internal/logger/logs.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	fileOut := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), logLevel)
	stdOut := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), logLevel)

	loggerCore := zapcore.NewTee(fileOut, stdOut)

	logger := zap.New(loggerCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	defer logger.Sync()

	return logger, nil
}

type LogMessage struct {
	Message string
	Fields  []zap.Field
}
