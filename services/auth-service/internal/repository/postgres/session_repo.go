package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
)

// SessionRepo implements SessionRepository
type SessionRepo struct {
	db *pgxpool.Pool
}

// NewSessionRepo creates a new SessionRepo
func NewSessionRepo(db *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create creates a new session
func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, access_token_id, device_name,
			device_type, ip_address, user_agent, expires_at, revoked_at, last_activity_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	
	_, err := r.db.Exec(ctx, query,
		session.ID, session.UserID, session.RefreshTokenHash, session.AccessTokenID,
		session.DeviceName, session.DeviceType, session.IPAddress,
		session.UserAgent, session.ExpiresAt, session.RevokedAt,
		session.LastActivityAt, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

// GetByID gets session by ID
func (r *SessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, access_token_id, device_name,
			device_type, ip_address, user_agent, expires_at, revoked_at,
			last_activity_at, created_at, updated_at
		FROM sessions WHERE id = $1
	`
	
	session := &domain.Session{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.UserID, &session.RefreshTokenHash, &session.AccessTokenID,
		&session.DeviceName, &session.DeviceType, &session.IPAddress,
		&session.UserAgent, &session.ExpiresAt, &session.RevokedAt,
		&session.LastActivityAt, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	return session, nil
}

// GetByRefreshTokenHash gets session by refresh token hash
func (r *SessionRepo) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, access_token_id, device_name,
			device_type, ip_address, user_agent, expires_at, revoked_at,
			last_activity_at, created_at, updated_at
		FROM sessions WHERE refresh_token_hash = $1 AND (revoked_at IS NULL AND expires_at > NOW())
	`
	
	session := &domain.Session{}
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&session.ID, &session.UserID, &session.RefreshTokenHash, &session.AccessTokenID,
		&session.DeviceName, &session.DeviceType, &session.IPAddress,
		&session.UserAgent, &session.ExpiresAt, &session.RevokedAt,
		&session.LastActivityAt, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

// GetByUserID gets sessions for a user
func (r *SessionRepo) GetByUserID(ctx context.Context, userID uuid.UUID, filter domain.SessionFilter) ([]domain.Session, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, userID)
	argIdx++

	if filter.DeviceType != nil {
		conditions = append(conditions, fmt.Sprintf("device_type = $%d", argIdx))
		args = append(args, *filter.DeviceType)
		argIdx++
	}

	if filter.IsRevoked != nil {
		if *filter.IsRevoked {
			conditions = append(conditions, "(revoked_at IS NOT NULL OR expires_at <= NOW())")
		} else {
			conditions = append(conditions, "revoked_at IS NULL AND expires_at > NOW()")
		}
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) FROM sessions ` + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, refresh_token_hash, access_token_id, device_name,
			device_type, ip_address, user_agent, expires_at, revoked_at,
			last_activity_at, created_at, updated_at
		FROM sessions %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		var session domain.Session
		err := rows.Scan(
			&session.ID, &session.UserID, &session.RefreshTokenHash, &session.AccessTokenID,
			&session.DeviceName, &session.DeviceType, &session.IPAddress,
			&session.UserAgent, &session.ExpiresAt, &session.RevokedAt,
			&session.LastActivityAt, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, session)
	}

	return sessions, total, nil
}

// Revoke revokes a session
func (r *SessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sessions SET revoked_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *SessionRepo) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	var query string
	var args []interface{}
	
	if exceptSessionID != nil {
		query = `UPDATE sessions SET revoked_at = NOW(), updated_at = NOW() 
			WHERE user_id = $1 AND id != $2 AND revoked_at IS NULL`
		args = append(args, userID, *exceptSessionID)
	} else {
		query = `UPDATE sessions SET revoked_at = NOW(), updated_at = NOW() 
			WHERE user_id = $1 AND revoked_at IS NULL`
		args = append(args, userID)
	}
	
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

// UpdateActivity updates session activity
func (r *SessionRepo) UpdateActivity(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sessions SET last_activity_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Client operations

// CreateClient creates a new OAuth client
func (r *SessionRepo) CreateClient(ctx context.Context, client *domain.Client) error {
	query := `
		INSERT INTO clients (id, tenant_id, client_id, client_secret_hash, name, redirect_uris,
			grant_types, scope, is_active, access_token_ttl, refresh_token_ttl, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	
	_, err := r.db.Exec(ctx, query,
		client.ID, client.TenantID, client.ClientID, client.ClientSecretHash,
		client.Name, client.RedirectURIs, client.GrantTypes, client.Scope,
		client.IsActive, client.AccessTokenTTL, client.RefreshTokenTTL,
		client.CreatedAt, client.UpdatedAt,
	)
	return err
}

// GetClientByClientID gets client by client ID
func (r *SessionRepo) GetClientByClientID(ctx context.Context, clientID string) (*domain.Client, error) {
	query := `
		SELECT id, tenant_id, client_id, client_secret_hash, name, redirect_uris,
			grant_types, scope, is_active, access_token_ttl, refresh_token_ttl, created_at, updated_at
		FROM clients WHERE client_id = $1 AND is_active = true
	`
	
	client := &domain.Client{}
	err := r.db.QueryRow(ctx, query, clientID).Scan(
		&client.ID, &client.TenantID, &client.ClientID, &client.ClientSecretHash,
		&client.Name, &client.RedirectURIs, &client.GrantTypes, &client.Scope,
		&client.IsActive, &client.AccessTokenTTL, &client.RefreshTokenTTL,
		&client.CreatedAt, &client.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return client, nil
}

// Identity Provider operations

// CreateIdentityProvider creates a new identity provider
func (r *SessionRepo) CreateIdentityProvider(ctx context.Context, provider *domain.IdentityProvider) error {
	query := `
		INSERT INTO identity_providers (id, tenant_id, provider, client_id, client_secret,
			authorization_endpoint, token_endpoint, userinfo_endpoint, scopes, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err := r.db.Exec(ctx, query,
		provider.ID, provider.TenantID, provider.Provider, provider.ClientID,
		provider.ClientSecret, provider.AuthorizationEndpoint, provider.TokenEndpoint,
		provider.UserinfoEndpoint, provider.Scopes, provider.IsActive,
		provider.CreatedAt, provider.UpdatedAt,
	)
	return err
}

// GetIdentityProvidersByTenant gets identity providers for a tenant
func (r *SessionRepo) GetIdentityProvidersByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.IdentityProvider, error) {
	query := `
		SELECT id, tenant_id, provider, client_id, client_secret,
			authorization_endpoint, token_endpoint, userinfo_endpoint, scopes, is_active, created_at, updated_at
		FROM identity_providers WHERE tenant_id = $1 AND is_active = true
	`
	
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []domain.IdentityProvider
	for rows.Next() {
		var provider domain.IdentityProvider
		err := rows.Scan(
			&provider.ID, &provider.TenantID, &provider.Provider, &provider.ClientID,
			&provider.ClientSecret, &provider.AuthorizationEndpoint, &provider.TokenEndpoint,
			&provider.UserinfoEndpoint, &provider.Scopes, &provider.IsActive,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// GetIdentityProviderByTenantAndProvider gets identity provider by tenant and provider name
func (r *SessionRepo) GetIdentityProviderByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.IdentityProvider, error) {
	query := `
		SELECT id, tenant_id, provider, client_id, client_secret,
			authorization_endpoint, token_endpoint, userinfo_endpoint, scopes, is_active, created_at, updated_at
		FROM identity_providers WHERE tenant_id = $1 AND provider = $2 AND is_active = true
	`
	
	var ip domain.IdentityProvider
	err := r.db.QueryRow(ctx, query, tenantID, provider).Scan(
		&ip.ID, &ip.TenantID, &ip.Provider, &ip.ClientID,
		&ip.ClientSecret, &ip.AuthorizationEndpoint, &ip.TokenEndpoint,
		&ip.UserinfoEndpoint, &ip.Scopes, &ip.IsActive,
		&ip.CreatedAt, &ip.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ip, nil
}
