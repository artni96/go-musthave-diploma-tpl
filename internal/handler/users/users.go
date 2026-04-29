package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/repository/users"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/service/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type UserHandler struct {
	repository *usersrepo.UserRepository
	service    *usersserv.UserService
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewUserHandler(ctx *context.Context, app *config.App, repository *usersrepo.UserRepository, service *usersserv.UserService) *UserHandler {
	return &UserHandler{
		logger:     app.Logger,
		ctx:        ctx,
		cfg:        app.Config,
		repository: repository,
		service:    service,
	}
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to read body",
			Fields:  []zap.Field{zap.Error(err)},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusBadRequest, h.logger, logMessage, zapcore.DebugLevel)
		return
	}
	user := model.UserCreateRequest{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to unmarshal body",
			Fields:  []zap.Field{zap.Error(err)},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusBadRequest, h.logger, logMessage, zapcore.DebugLevel)
		return
	}
	userID, err := h.service.Create(*h.ctx, user)
	if err != nil {
		if errors.Is(err, usersrepo.ErrUserAlreadyExists) {
			logMessage := logger.LogMessage{
				Message: "user already exists",
				Fields:  []zap.Field{zap.String("Login", user.Login)},
			}
			handler.ErrorResponse(w, "user already exists", http.StatusConflict, h.logger, logMessage, zapcore.InfoLevel)
			return
		}

		logMessage := logger.LogMessage{
			Message: "failed to create user",
			Fields:  []zap.Field{zap.Error(err), zap.String("Login", user.Login)},
		}
		handler.ErrorResponse(w, "failed to create user", http.StatusInternalServerError, h.logger, logMessage, zapcore.InfoLevel)
		return

	}

	token, err := h.service.BuildJWTString(userID, h.cfg)
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(h.cfg.TokenExp),
		HttpOnly: true,
		Path:     "/",
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"user successfully created"}`))
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to read body",
			Fields:  []zap.Field{zap.Error(err)},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusBadRequest, h.logger, logMessage, zapcore.DebugLevel)
		return
	}

	user := model.UserLoginRequest{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		logMessage := logger.LogMessage{
			Message: "failed to unmarshal body",
			Fields:  []zap.Field{zap.Error(err)},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusBadRequest, h.logger, logMessage, zapcore.DebugLevel)
		return
	}

	if user.Login == "" || user.Password == "" {
		logMessage := logger.LogMessage{
			Message: "invalid request body",
			Fields: []zap.Field{zap.String(
				"request body", string(body)), zap.String("expected fields", "login, password"),
			},
		}
		handler.ErrorResponse(w, "invalid request body", http.StatusBadRequest, h.logger, logMessage, zapcore.DebugLevel)
		return
	}

	userResponse, err := h.service.Login(*h.ctx, user)
	if err != nil {
		if errors.Is(err, usersrepo.ErrUserNotFound) {
			logMessage := logger.LogMessage{
				Message: "user not found",
				Fields: []zap.Field{
					zap.String("Login", user.Login),
				},
			}
			handler.ErrorResponse(w, "wrong user or password", http.StatusUnauthorized, h.logger, logMessage, zapcore.InfoLevel)
			return
		}
		logMessage := logger.LogMessage{
			Message: "failed to login - wrong user or password",
			Fields: []zap.Field{
				zap.Error(err), zap.String("user", user.Login),
			},
		}
		handler.ErrorResponse(w, "wrong user or password", http.StatusUnauthorized, h.logger, logMessage, zapcore.InfoLevel)
		return
	}

	token, err := h.service.BuildJWTString(userResponse.ID, h.cfg)
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(h.cfg.TokenExp),
		HttpOnly: true,
		Path:     "/",
	})

	w.WriteHeader(http.StatusOK)
	return
}

func UserRouter(ctx *context.Context, app *config.App, repository *usersrepo.UserRepository, service *usersserv.UserService) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config.GzipMiddleware)
	r.Use(middlewares.RequestLoggerMiddleware(app.Logger))

	handler := NewUserHandler(ctx, app, repository, service)

	r.Route("/", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"Method is not allowed"}`)))
			return
		})
		r.Post("/register", handler.Create)
		r.Post("/login", handler.Login)
	})
	return r
}
