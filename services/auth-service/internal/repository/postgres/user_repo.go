package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
)

// UserRepo implements UserRepository
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// Create creates a new user
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, tenant_id, full_name, email, username, phone, avatar_url, 
			is_email_verified, is_phone_verified, status, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	
	_, err := r.db.Exec(ctx, query,
		user.ID, user.TenantID, user.FullName, user.Email, user.Username,
		user.Phone, user.AvatarURL, user.IsEmailVerified, user.IsPhoneVerified,
		user.Status, user.CreatedBy, user.UpdatedBy, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

// GetByID gets user by ID
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, full_name, email, username, phone, avatar_url,
			is_email_verified, is_phone_verified, status, last_login_at,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`
	
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.TenantID, &user.FullName, &user.Email, &user.Username,
		&user.Phone, &user.AvatarURL, &user.IsEmailVerified, &user.IsPhoneVerified,
		&user.Status, &user.LastLoginAt, &user.CreatedBy, &user.UpdatedBy,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail gets user by email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, full_name, email, username, phone, avatar_url,
			is_email_verified, is_phone_verified, status, last_login_at,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`
	
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.TenantID, &user.FullName, &user.Email, &user.Username,
		&user.Phone, &user.AvatarURL, &user.IsEmailVerified, &user.IsPhoneVerified,
		&user.Status, &user.LastLoginAt, &user.CreatedBy, &user.UpdatedBy,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// GetByUsername gets user by username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, full_name, email, username, phone, avatar_url,
			is_email_verified, is_phone_verified, status, last_login_at,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM users WHERE username = $1 AND deleted_at IS NULL
	`
	
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.TenantID, &user.FullName, &user.Email, &user.Username,
		&user.Phone, &user.AvatarURL, &user.IsEmailVerified, &user.IsPhoneVerified,
		&user.Status, &user.LastLoginAt, &user.CreatedBy, &user.UpdatedBy,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// GetByEmailAndTenant gets user by email within a tenant
func (r *UserRepo) GetByEmailAndTenant(ctx context.Context, email string, tenantID uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, full_name, email, username, phone, avatar_url,
			is_email_verified, is_phone_verified, status, last_login_at,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, email, tenantID).Scan(
		&user.ID, &user.TenantID, &user.FullName, &user.Email, &user.Username,
		&user.Phone, &user.AvatarURL, &user.IsEmailVerified, &user.IsPhoneVerified,
		&user.Status, &user.LastLoginAt, &user.CreatedBy, &user.UpdatedBy,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// List lists users with filter
func (r *UserRepo) List(ctx context.Context, filter domain.UserFilter) ([]domain.User, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, *filter.TenantID)
		argIdx++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(full_name ILIKE $%d OR email ILIKE $%d OR username ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	var total int64
	countQuery := `SELECT COUNT(*) FROM users ` + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	sort := filter.Sort
	if sort == "" {
		sort = "created_at"
	}
	order := filter.Order
	if order == "" {
		order = "DESC"
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, full_name, email, username, phone, avatar_url,
			is_email_verified, is_phone_verified, status, last_login_at,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM users %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sort, order, argIdx, argIdx+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID, &user.TenantID, &user.FullName, &user.Email, &user.Username,
			&user.Phone, &user.AvatarURL, &user.IsEmailVerified, &user.IsPhoneVerified,
			&user.Status, &user.LastLoginAt, &user.CreatedBy, &user.UpdatedBy,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

// Update updates a user
func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users SET
			full_name = $1, email = $2, username = $3, phone = $4,
			avatar_url = $5, is_email_verified = $6, is_phone_verified = $7,
			status = $8, updated_by = $9, updated_at = $10
		WHERE id = $11 AND deleted_at IS NULL
	`
	
	_, err := r.db.Exec(ctx, query,
		user.FullName, user.Email, user.Username, user.Phone,
		user.AvatarURL, user.IsEmailVerified, user.IsPhoneVerified,
		user.Status, user.UpdatedBy, time.Now(), user.ID,
	)
	return err
}

// Delete soft deletes a user
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	query := `
		UPDATE users SET
			status = $1, updated_by = $2, updated_at = $3, deleted_at = $4
		WHERE id = $5 AND deleted_at IS NULL
	`
	
	now := time.Now()
	_, err := r.db.Exec(ctx, query,
		domain.UserStatusBlocked, deletedBy, now, now, id,
	)
	return err
}

// UpdateLastLogin updates last login timestamp
func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $1, updated_at = $2 WHERE id = $3`
	now := time.Now()
	_, err := r.db.Exec(ctx, query, now, now, id)
	return err
}

// CreateCredential creates a new credential
func (r *UserRepo) CreateCredential(ctx context.Context, cred *domain.Credential) error {
	query := `
		INSERT INTO user_credentials (id, user_id, type, credential_hash, provider_user_id, 
			provider_data, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.Exec(ctx, query,
		cred.ID, cred.UserID, cred.Type, cred.CredentialHash,
		cred.ProviderUserID, cred.ProviderData, cred.IsActive,
		cred.CreatedAt, cred.UpdatedAt,
	)
	return err
}

// GetCredentialsByUserID gets all credentials for a user
func (r *UserRepo) GetCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Credential, error) {
	query := `
		SELECT id, user_id, type, credential_hash, provider_user_id,
			provider_data, is_active, last_used_at, created_at, updated_at
		FROM user_credentials WHERE user_id = $1 ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []domain.Credential
	for rows.Next() {
		var cred domain.Credential
		err := rows.Scan(
			&cred.ID, &cred.UserID, &cred.Type, &cred.CredentialHash,
			&cred.ProviderUserID, &cred.ProviderData, &cred.IsActive,
			&cred.LastUsedAt, &cred.CreatedAt, &cred.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}

	return creds, nil
}

// GetActiveCredentialByType gets active credential by type
func (r *UserRepo) GetActiveCredentialByType(ctx context.Context, userID uuid.UUID, credType domain.CredentialType) (*domain.Credential, error) {
	query := `
		SELECT id, user_id, type, credential_hash, provider_user_id,
			provider_data, is_active, last_used_at, created_at, updated_at
		FROM user_credentials WHERE user_id = $1 AND type = $2 AND is_active = true
		ORDER BY created_at DESC LIMIT 1
	`
	
	var cred domain.Credential
	err := r.db.QueryRow(ctx, query, userID, credType).Scan(
		&cred.ID, &cred.UserID, &cred.Type, &cred.CredentialHash,
		&cred.ProviderUserID, &cred.ProviderData, &cred.IsActive,
		&cred.LastUsedAt, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// UpdateCredential updates a credential
func (r *UserRepo) UpdateCredential(ctx context.Context, cred *domain.Credential) error {
	query := `
		UPDATE user_credentials SET
			credential_hash = $1, is_active = $2, last_used_at = $3, updated_at = $4
		WHERE id = $5
	`
	
	_, err := r.db.Exec(ctx, query,
		cred.CredentialHash, cred.IsActive, cred.LastUsedAt,
		time.Now(), cred.ID,
	)
	return err
}

// GetProfileByUserID gets user profile
func (r *UserRepo) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	query := `
		SELECT id, avatar, user_id, full_name, nik_number, place_birth,
			date_of_birth, mobile, address, zip_code, extras
		FROM profile WHERE user_id = $1
	`
	
	var profile domain.Profile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.Avatar, &profile.UserID, &profile.FullName,
		&profile.NIKNumber, &profile.PlaceBirth, &profile.DateOfBirth,
		&profile.Mobile, &profile.Address, &profile.ZipCode, &profile.Extras,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// CreateOrUpdateProfile creates or updates user profile
func (r *UserRepo) CreateOrUpdateProfile(ctx context.Context, profile *domain.Profile) error {
	query := `
		INSERT INTO profile (id, avatar, user_id, full_name, nik_number, place_birth,
			date_of_birth, mobile, address, zip_code, extras)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE SET
			avatar = EXCLUDED.avatar, full_name = EXCLUDED.full_name,
			nik_number = EXCLUDED.nik_number, place_birth = EXCLUDED.place_birth,
			date_of_birth = EXCLUDED.date_of_birth, mobile = EXCLUDED.mobile,
			address = EXCLUDED.address, zip_code = EXCLUDED.zip_code,
			extras = EXCLUDED.extras
	`
	
	_, err := r.db.Exec(ctx, query,
		profile.ID, profile.Avatar, profile.UserID, profile.FullName,
		profile.NIKNumber, profile.PlaceBirth, profile.DateOfBirth,
		profile.Mobile, profile.Address, profile.ZipCode, profile.Extras,
	)
	return err
}

// GetAttributesByUserID gets user attributes
func (r *UserRepo) GetAttributesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Attribute, error) {
	query := `
		SELECT id, user_id, key, value, created_at, updated_at
		FROM user_attributes WHERE user_id = $1
	`
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attrs []domain.Attribute
	for rows.Next() {
		var attr domain.Attribute
		err := rows.Scan(&attr.ID, &attr.UserID, &attr.Key, &attr.Value, &attr.CreatedAt, &attr.UpdatedAt)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, attr)
	}

	return attrs, nil
}

// SetAttribute sets a user attribute
func (r *UserRepo) SetAttribute(ctx context.Context, attr *domain.Attribute) error {
	query := `
		INSERT INTO user_attributes (id, user_id, key, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, key) DO UPDATE SET
			value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`
	
	_, err := r.db.Exec(ctx, query,
		attr.ID, attr.UserID, attr.Key, attr.Value, attr.CreatedAt, attr.UpdatedAt,
	)
	return err
}
