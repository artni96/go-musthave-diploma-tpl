package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler"
	"go.uber.org/zap"
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

func PanicRecoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Info("got panic",
						zap.String("error message", fmt.Sprintf("panic recovered: %v\n", recovered)),
						zap.String("call stack", string(debug.Stack())),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					resp, _ := json.Marshal(struct {
						Error string `json:"error"`
					}{
						Error: "Internal Server Error",
					})
					w.Write(resp)
					return
				}
			}()

			next.ServeHTTP(w, r)

		}
		return http.HandlerFunc(fn)
	}
}

func AuthorizationMiddleware(app *config.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token, err := r.Cookie("jwt")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			userID := handler.GetUserID(token.Value, app.Config)
			if userID == "" {
				app.Logger.Debug("user id not found in cookie", zap.String("token", token.Value))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			r.Header.Set("UserID", userID)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
