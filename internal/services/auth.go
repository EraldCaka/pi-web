package services

import (
	"errors"
	"time"

	"github.com/EraldCaka/pi-web/internal/config"
	"github.com/EraldCaka/pi-web/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string      `json:"user_id"`
	Email  string      `json:"email"`
	Role   models.Role `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	cfg config.JWTConfig
}

func NewAuthService(cfg config.JWTConfig) *AuthService {
	return &AuthService{cfg: cfg}
}

func (s *AuthService) Sign(user *models.User) (string, error) {
	claims := Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.Expiry())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.Secret))
}

func (s *AuthService) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
