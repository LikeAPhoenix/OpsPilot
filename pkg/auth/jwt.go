package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims 定义服务内统一使用的访问令牌声明。
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Manager 封装 JWT access token 的签发与解析能力。
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager 创建 JWT 管理器。
func NewManager(secret string, ttl time.Duration) (*Manager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("jwt secret is required")
	}
	if ttl <= 0 {
		return nil, errors.New("jwt ttl must be greater than zero")
	}
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

// IssueAccessToken 为指定用户签发 access token。
func (m *Manager) IssueAccessToken(userID string) (string, time.Time, error) {
	if strings.TrimSpace(userID) == "" {
		return "", time.Time{}, errors.New("user id is required")
	}

	expiresAt := time.Now().Add(m.ttl)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// ParseAccessToken 解析 access token 并返回 claims。
func (m *Manager) ParseAccessToken(token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrInvalidToken)
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid || strings.TrimSpace(claims.UserID) == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
