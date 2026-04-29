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
	"github.com/artni96/go-musthave-diploma-tpl/internal/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/repository/users"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/service/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
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
		h.logger.Info("failed to read body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}
	user := model.UserCreateRequest{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		h.logger.Info("failed to unmarshal body", zap.Error(err), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}
	userID, err := h.service.Create(*h.ctx, user)
	if err != nil {
		if errors.Is(err, usersrepo.ErrUserAlreadyExists) {
			h.logger.Info("user already exists", zap.String("Login", user.Login), zap.String("layer", "user handler"))
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error": "user already exists"}`))
			return
		}
		h.logger.Info("failed to create user", zap.Error(err), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
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
	w.Write([]byte(`{"message": "user successfully created"}`))
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		h.logger.Info("failed to read body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}

	user := model.UserLoginRequest{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		h.logger.Info("failed to unmarshal body", zap.Error(err), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}

	if user.Login == "" || user.Password == "" {
		h.logger.Debug("invalid request body", zap.String("request body", string(body)), zap.String("expected fields", "login, password"), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}

	userResponse, err := h.service.Login(*h.ctx, user)
	if err != nil {
		if errors.Is(err, usersrepo.ErrUserNotFound) {
			h.logger.Info("user not found", zap.String("Login", user.Login), zap.String("layer", "user handler"))
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "wrong user or password"}`))
			return
		}
		h.logger.Info("failed to login", zap.Error(err), zap.String("user", user.Login), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "wrong user or password"}`))
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
