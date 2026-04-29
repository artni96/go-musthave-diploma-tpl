package handler

import (
	"fmt"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
	"github.com/artni96/go-musthave-diploma-tpl/internal/service/users"
	"github.com/golang-jwt/jwt/v4"
)

func GetUserID(tokenString string, cfg *config.Config) string {
	claims := &users.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.SecretKey), nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}
	return claims.UserID
}
