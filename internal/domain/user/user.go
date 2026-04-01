package user

import "time"

// User 表示系统内最小可用账号实体。
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
