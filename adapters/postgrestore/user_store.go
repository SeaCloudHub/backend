package postgrestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/pagination"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(ctx context.Context, user *identity.User) error {
	userSchema := UserSchema{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		AvatarURL: user.AvatarURL,
		IsActive:  true,
	}

	return s.db.WithContext(ctx).Create(&userSchema).Error
}

func (s *UserStore) UpdateAdmin(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("is_admin", gorm.Expr("NOT is_admin")).
		Error
}

func (s *UserStore) UpdatePasswordChangedAt(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("password_changed_at", time.Now()).
		Error
}

func (s *UserStore) UpdateLastSignInAt(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("last_signin_at", time.Now()).
		Error
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	var userSchema UserSchema
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&userSchema).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, identity.ErrIdentityNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return &identity.User{
		ID:                userSchema.ID,
		Email:             userSchema.Email,
		FirstName:         userSchema.FirstName,
		LastName:          userSchema.LastName,
		AvatarURL:         userSchema.AvatarURL,
		IsActive:          userSchema.IsActive,
		IsAdmin:           userSchema.IsAdmin,
		PasswordChangedAt: userSchema.PasswordChangedAt,
		CreatedAt:         userSchema.CreatedAt,
		UpdatedAt:         userSchema.UpdatedAt,
	}, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*identity.User, error) {
	var userSchema UserSchema
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&userSchema).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, identity.ErrIdentityNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return &identity.User{
		ID:                userSchema.ID,
		Email:             userSchema.Email,
		FirstName:         userSchema.FirstName,
		LastName:          userSchema.LastName,
		AvatarURL:         userSchema.AvatarURL,
		IsActive:          userSchema.IsActive,
		IsAdmin:           userSchema.IsAdmin,
		PasswordChangedAt: userSchema.PasswordChangedAt,
		CreatedAt:         userSchema.CreatedAt,
		UpdatedAt:         userSchema.UpdatedAt,
	}, nil
}

func (s *UserStore) List(ctx context.Context, pager *pagination.Pager) ([]identity.User, error) {
	var (
		userSchemas []UserSchema
		total       int64
	)

	if err := s.db.WithContext(ctx).Model(&userSchemas).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	pager.SetTotal(total)

	offset, limit := pager.Do()
	if err := s.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&userSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	users := make([]identity.User, 0, len(userSchemas))
	for _, userSchema := range userSchemas {
		users = append(users, identity.User{
			ID:                userSchema.ID,
			Email:             userSchema.Email,
			FirstName:         userSchema.FirstName,
			LastName:          userSchema.LastName,
			AvatarURL:         userSchema.AvatarURL,
			IsActive:          userSchema.IsActive,
			IsAdmin:           userSchema.IsAdmin,
			PasswordChangedAt: userSchema.PasswordChangedAt,
			CreatedAt:         userSchema.CreatedAt,
			UpdatedAt:         userSchema.UpdatedAt,
		})
	}

	return users, nil
}