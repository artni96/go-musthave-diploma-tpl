package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/model"
	"github.com/artni96/go-musthave-diploma-tpl/internal/repository"
	"go.uber.org/zap"
)

type UserServiceInterface interface {
	Create(ctx context.Context, user model.UserCreateRequest) error
	Login(ctx context.Context, user model.UserLoginRequest) (model.User, error)
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

func (s *UserService) Create(ctx context.Context, user model.UserCreateRequest) error {
	hashedPassword, err := s.hashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashedPassword
	err = s.repository.Create(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (s *UserService) Login(ctx context.Context, user model.UserLoginRequest) (model.User, error) {
	hashedPassword, err := s.hashPassword(user.Password)
	if err != nil {
		return model.User{}, err
	}
	user.Password = hashedPassword

	userEntity, err := s.repository.Login(ctx, user)
	if err != nil {
		return model.User{}, err
	}
	return userEntity, nil
}

func (s *UserService) hashPassword(password string) (string, error) {
	key := sha256.Sum256([]byte(s.app.Config.SecretKey))
	strKey := string(key[:])
	fmt.Println(strKey)
	aesblock, err := aes.NewCipher(key[:])
	if err != nil {
		s.app.Logger.Info("failed to create new aesblock", zap.Error(err))
		return "", fmt.Errorf("failed to create new aesblock: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		s.app.Logger.Info("failed to create new aesgcm", zap.Error(err))
		return "", fmt.Errorf("failed to create new aesgcm: %w", err)
	}

	nonce := []byte("123456789012")
	if err != nil {
		s.app.Logger.Info("failed to generate random nonce", zap.Error(err))
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	dst := aesgcm.Seal(nil, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(dst), nil
}

func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
