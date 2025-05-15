package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	accessSecret  []byte
	refreshSecret []byte
)

type CustomClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func SetJwtKeys(access, refresh string) {
	accessSecret = []byte(access)
	refreshSecret = []byte(refresh)
}

func GenerateToken(userID string) (string, error) {
	claims := CustomClaims{
		Username: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Markdocs",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(accessSecret)
}

func ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwt.Token) (any, error) { return accessSecret, nil },
	)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("unknown claims type (invalid key), cannot proceed")
}
