package mysql

import (
	"context"
	"time"

	domainuser "OpsPilot/internal/domain/user"

	"gorm.io/gorm"
)

// Store 使用 MySQL 持久化用户账号。
type Store struct {
	db *gorm.DB
}

type userModel struct {
	ID           string `gorm:"column:id;primaryKey"`
	Username     string `gorm:"column:username;uniqueIndex;size:128;not null"`
	PasswordHash string `gorm:"column:password_hash;size:255;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userModel) TableName() string {
	return "users"
}

// NewStore 创建用户仓储并确保表结构存在。
func NewStore(db *gorm.DB) (*Store, error) {
	if err := db.AutoMigrate(&userModel{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Create 保存一个新用户。
func (s *Store) Create(ctx context.Context, user *domainuser.User) error {
	model := userModel{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
	return s.db.WithContext(ctx).Create(&model).Error
}

// GetByUsername 根据用户名查询用户。
func (s *Store) GetByUsername(ctx context.Context, username string) (*domainuser.User, error) {
	var model userModel
	err := s.db.WithContext(ctx).Where("username = ?", username).Take(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &domainuser.User{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}
