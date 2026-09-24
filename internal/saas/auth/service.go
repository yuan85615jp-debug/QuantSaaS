package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims carried in JWT for SaaS users and agents.
type Claims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"` // user | admin | agent
	jwt.RegisteredClaims
}

// Service signs and parses tokens using the configured secret.
type Service struct {
	secret     []byte
	expireHour int
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		secret:     []byte(cfg.JWT.Secret),
		expireHour: cfg.JWT.ExpireHour,
	}
}

// SignToken issues a JWT for the given user id and role.
func (s *Service) SignToken(userID uint, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.expireHour) * time.Hour)),
			Issuer:    "quantsaas",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ParseToken validates signature and expiry, returns claims.
func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
