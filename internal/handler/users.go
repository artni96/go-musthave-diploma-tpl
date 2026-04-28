package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"github.com/artni96/go-musthave-diploma-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type UserHandler struct {
	repository *repository.UserRepository
	service    *service.UserService
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config.Config
}

func NewUserHandler(ctx *context.Context, logger *zap.Logger, cfg *config.Config, repository *repository.UserRepository, service *service.UserService) *UserHandler {
	return &UserHandler{
		logger:     logger,
		ctx:        ctx,
		cfg:        cfg,
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
		return
	}
	user := model.UserCreateRequest{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		h.logger.Info("failed to unmarshal body", zap.Error(err), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID, err := h.service.Create(*h.ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			h.logger.Info("user already exists", zap.String("Login", user.Login), zap.String("layer", "user handler"))
			w.WriteHeader(http.StatusConflict)
			return
		}
		h.logger.Info("failed to create user", zap.Error(err), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusInternalServerError)
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
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		h.logger.Info("failed to read body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := model.UserLoginRequest{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		h.logger.Info("failed to unmarshal body", zap.Error(err), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if user.Login == "" || user.Password == "" {
		h.logger.Debug("invalid request body", zap.String("request body", string(body)), zap.String("expected fields", "login, password"), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userResponse, err := h.service.Login(*h.ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			h.logger.Info("user not found", zap.String("Login", user.Login), zap.String("layer", "user handler"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.logger.Info("failed to login", zap.Error(err), zap.String("user", user.Login), zap.String("layer", "user handler"))
		w.WriteHeader(http.StatusUnauthorized)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return
}

func UserRouter(
	ctx *context.Context,
	app *config.App,
	cfg *config.Config,

	userRepository *repository.UserRepository,
	userService *service.UserService,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	userHandler := NewUserHandler(ctx, app.Logger, cfg, userRepository, userService)

	r.Route("/api/user", func(r chi.Router) {
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Method %s is not allowed", r.Method)))
			return
		})
		r.Post("/register", userHandler.Create)
		r.Post("/login", userHandler.Login)
	})
	return r
}
