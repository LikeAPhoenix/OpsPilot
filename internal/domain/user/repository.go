package user

import "context"

// Repository 定义账号的最小读写能力。
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByUsername(ctx context.Context, username string) (*User, error)
}
