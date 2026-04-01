package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	domainuser "OpsPilot/internal/domain/user"
	pkgauth "OpsPilot/pkg/auth"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUsernameTaken      = errors.New("用户名已存在")
)

// IDGenerator 定义注册时生成用户 ID 的能力。
type IDGenerator interface {
	NextString() string
}

// Service 封装账号注册和登录逻辑。
type Service struct {
	users  domainuser.Repository
	ids    IDGenerator
	tokens *pkgauth.Manager
}

// LoginResult 表示登录成功后的返回值。
type LoginResult struct {
	UserID      string
	AccessToken string
	ExpiresAt   time.Time
}

// NewService 创建认证应用服务。
func NewService(users domainuser.Repository, ids IDGenerator, tokens *pkgauth.Manager) *Service {
	return &Service{
		users:  users,
		ids:    ids,
		tokens: tokens,
	}
}

// Register 创建一个新用户。
func (s *Service) Register(ctx context.Context, username, password string) (*domainuser.User, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	existing, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domainuser.User{
		ID:           s.ids.NextString(),
		Username:     username,
		PasswordHash: string(passwordHash),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login 校验用户名密码并签发 access token。
func (s *Service) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, expiresAt, err := s.tokens.IssueAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		UserID:      user.ID,
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
	}, nil
}
