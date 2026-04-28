package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("wrong password")

type UserServiceInterface interface {
	Create(ctx context.Context, user model.UserCreateRequest) (string, error)
	Login(ctx context.Context, user model.UserLoginRequest) (model.User, error)
	BuildJWTString(userID string, cfg *config.Config) (string, error)
}
type UserService struct {
	repository repository.UserRepositoryInterface
	app        *config.App
}

func NewUserService(repository repository.UserRepositoryInterface, app *config.App) *UserService {
	return &UserService{
		repository: repository,
		app:        app,
	}
}

func (s *UserService) Create(ctx context.Context, user model.UserCreateRequest) (string, error) {
	hashedPassword, err := s.hashPassword(user.Password)
	if err != nil {
		return "", err
	}
	user.Password = hashedPassword
	userID, err := s.repository.Create(ctx, user)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *UserService) Login(ctx context.Context, user model.UserLoginRequest) (model.User, error) {
	userEntity, err := s.repository.GetByLogin(ctx, user.Login)
	if err != nil {
		return model.User{}, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(userEntity.Password), []byte(user.Password))
	if err != nil {
		return model.User{}, fmt.Errorf("%w: %s", ErrWrongPassword, err)
	}
	return userEntity, nil
}

func (s *UserService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func (s *UserService) BuildJWTString(userID string, cfg *config.Config) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.TokenExp)),
		},
		UserID: userID,
	})
	tokenString, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}
	return tokenString, nil
}
