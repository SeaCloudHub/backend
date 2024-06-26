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
		ID:              user.ID,
		Email:           user.Email,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		AvatarURL:       user.AvatarURL,
		IsActive:        true,
		StorageCapacity: 10 << 30,
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

func (s *UserStore) UpdateNameAndAvatar(ctx context.Context, id uuid.UUID, avatar string, firstName string, lastName string) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"avatar_url": avatar,
			"first_name": firstName,
			"last_name":  lastName,
		}).
		Error
}

func (s *UserStore) UpdateLastSignInAt(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("last_signin_at", time.Now()).
		Error
}

func (s *UserStore) UpdateRootID(ctx context.Context, id, rootID uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("root_id", rootID).
		Error
}

func (s *UserStore) UpdateStorageUsage(ctx context.Context, id uuid.UUID, usage uint64) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("storage_usage", usage).
		Error
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*identity.User, error) {
	var userSchema UserSchema
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&userSchema).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, identity.ErrIdentityNotFound
		}

		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	return userSchema.ToDomainUser(), nil
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

func (s *UserStore) GetAll(ctx context.Context) ([]identity.User, error) {
	var userSchemas []UserSchema
	if err := s.db.WithContext(ctx).Find(&userSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	users := make([]identity.User, 0, len(userSchemas))
	for _, userSchema := range userSchemas {
		users = append(users, *userSchema.ToDomainUser())
	}

	return users, nil
}

func (s *UserStore) List(ctx context.Context, pager *pagination.Pager, filter identity.Filter) ([]identity.User, error) {
	var (
		userSchemas []UserSchema
		total       int64
	)

	query := s.db.Model(&userSchemas)
	if filter.Keyword != "" {
		query = query.Where("search_vector @@ PLAINTO_TSQUERY('english_nostop', REPLACE(REPLACE(?, '@', ' '), '.', ' '))", filter.Keyword)

		if id, err := uuid.Parse(filter.Keyword); err == nil {
			query = query.Or("id = ?", id)
		}
	}

	if err := query.WithContext(ctx).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	pager.SetTotal(total)

	offset, limit := pager.Do()
	if err := query.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&userSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	users := make([]identity.User, 0, len(userSchemas))
	for _, userSchema := range userSchemas {
		users = append(users, *userSchema.ToDomainUser())
	}

	return users, nil
}

func (s *UserStore) ListByEmails(ctx context.Context, emails []string) ([]identity.User, error) {
	var userSchemas []UserSchema
	if err := s.db.WithContext(ctx).Where("email IN ?", emails).Find(&userSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	users := make([]identity.User, 0, len(userSchemas))
	for _, userSchema := range userSchemas {
		users = append(users, *userSchema.ToDomainUser())
	}

	return users, nil
}

func (s *UserStore) ListByIDs(ctx context.Context, ids []string) ([]identity.User, error) {
	var userSchemas []UserSchema
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&userSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	users := make([]identity.User, 0, len(userSchemas))
	for _, userSchema := range userSchemas {
		users = append(users, *userSchema.ToDomainUser())
	}

	return users, nil
}

func (s *UserStore) FuzzySearch(ctx context.Context, query string) ([]identity.User, error) {
	var userSchemas []UserSchema

	if err := s.db.WithContext(ctx).
		Where("similarity(email, ?) > 0.1", query).
		Limit(10).
		Order(fmt.Sprintf("similarity(email, '%s') DESC", query)).
		Find(&userSchemas).Error; err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	users := make([]identity.User, 0, len(userSchemas))
	for _, userSchema := range userSchemas {
		users = append(users, *userSchema.ToDomainUser())
	}

	return users, nil
}

func (s *UserStore) UpdateStorageCapacity(ctx context.Context, id uuid.UUID, storageCapacity uint64) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Update("storage_capacity", storageCapacity).
		Error
}

func (s *UserStore) ToggleActive(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  gorm.Expr("NOT is_active"),
			"blocked_at": gorm.Expr("CASE WHEN is_active THEN NOW() ELSE NULL END"),
		}).
		Error
}

func (s *UserStore) Update(ctx context.Context, user *identity.User) error {
	userSchema := UserSchema{
		Email:             user.Email,
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		AvatarURL:         user.AvatarURL,
		IsActive:          user.IsActive,
		IsAdmin:           user.IsAdmin,
		PasswordChangedAt: user.PasswordChangedAt,
		RootID:            user.RootID,
		StorageUsage:      user.StorageUsage,
		StorageCapacity:   user.StorageCapacity,
		BlockedAt:         user.BlockedAt,
	}

	return s.db.WithContext(ctx).Model(&UserSchema{}).
		Where("id = ?", user.ID).
		Updates(&userSchema).
		Error
}

func (s *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&UserSchema{}, id).Error
}
