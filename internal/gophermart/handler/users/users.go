package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	config2 "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/handler/middlewares"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/logger"
	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/model"
	usersrepo "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/repository/users"
	usersserv "github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/service/users"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/artni96/go-musthave-diploma-tpl/api/docs"
)

type UserHandler struct {
	repository usersrepo.UserRepositoryInterface
	service    usersserv.UserServiceInterface
	logger     *zap.Logger
	ctx        *context.Context
	cfg        *config2.Config
}

func NewUserHandler(ctx *context.Context, app *config2.App, repository usersrepo.UserRepositoryInterface, service usersserv.UserServiceInterface) *UserHandler {
	return &UserHandler{
		logger:     app.Logger,
		ctx:        ctx,
		cfg:        app.Config,
		repository: repository,
		service:    service,
	}
}

// Create godoc
// @Summary User registration
// @Description User creation by login and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body model.UserCreateRequest true "User registration by login and password"
// @Success      200 "User successfully created and authenticated"
// @Failure      400
// @Failure      409 "User already exists"
// @Failure      500
// @Router       /api/user/register [post]
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
	w.Write([]byte(`{"message":"user successfully created and authenticated"}`))
}

// Login godoc
// @Summary User authentification and authorization
// @Description User authentification and authorization by login and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body model.UserLoginRequest true "User authentification and authorization by login and password"
// @Success      200
// @Failure      400
// @Failure      401 "Wrong login or password"
// @Failure      500
// @Router       /api/user/login [post]
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
			handler.ErrorResponse(w, "wrong login or password", http.StatusUnauthorized, h.logger, logMessage, zapcore.InfoLevel)
			return
		}
		logMessage := logger.LogMessage{
			Message: "failed to login - wrong login or password",
			Fields: []zap.Field{
				zap.Error(err), zap.String("login", user.Login),
			},
		}
		handler.ErrorResponse(w, "wrong login or password", http.StatusUnauthorized, h.logger, logMessage, zapcore.InfoLevel)
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

func UserRouter(ctx *context.Context, app *config2.App, repository *usersrepo.UserRepository, service *usersserv.UserService) http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.PanicRecoverer(app.Logger))
	r.Use(middleware.RequestID)
	r.Use(config2.GzipMiddleware)
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
