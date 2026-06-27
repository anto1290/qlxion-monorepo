package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/qlxion/qlxion-monorepo/pkg/auth"
	appErrors "github.com/qlxion/qlxion-monorepo/pkg/errors"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthUsecase handles authentication business logic
type AuthUsecase struct {
	userRepo    repository.UserRepository
	roleRepo    repository.RoleRepository
	tenantRepo  repository.TenantRepository
	sessionRepo repository.SessionRepository
	auditRepo   repository.AuditRepository
	jwtConfig   auth.JWTConfig
}

// NewAuthUsecase creates a new AuthUsecase
func NewAuthUsecase(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	tenantRepo repository.TenantRepository,
	sessionRepo repository.SessionRepository,
	auditRepo repository.AuditRepository,
	jwtConfig auth.JWTConfig,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		tenantRepo:  tenantRepo,
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
		jwtConfig:   jwtConfig,
	}
}

// Login authenticates a user and returns tokens
func (u *AuthUsecase) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	// Find user
	var user *domain.User
	var err error

	if req.TenantID != nil {
		user, err = u.userRepo.GetByEmailAndTenant(ctx, req.Email, *req.TenantID)
	} else {
		user, err = u.userRepo.GetByEmail(ctx, req.Email)
	}

	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInvalidCredentials, "Invalid email or password", err)
	}

	if user == nil {
		return nil, appErrors.New(appErrors.ErrInvalidCredentials, "Invalid email or password")
	}

	// Check user status
	if !user.IsActive() {
		u.logAudit(ctx, domain.AuditActionLogin, user, nil, map[string]interface{}{
			"error": "user account is not active",
		})
		return nil, appErrors.New(appErrors.ErrUserInactive, "User account is not active")
	}

	// Get tenant and check status
	tenant, err := u.tenantRepo.GetByID(ctx, user.TenantID)
	if err != nil || tenant == nil || !tenant.IsActive() {
		return nil, appErrors.New(appErrors.ErrTenantInactive, "Tenant is not active")
	}

	// Verify password
	cred, err := u.userRepo.GetActiveCredentialByType(ctx, user.ID, domain.CredentialTypePassword)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to verify credentials", err)
	}

	if cred == nil || cred.CredentialHash == nil {
		return nil, appErrors.New(appErrors.ErrInvalidCredentials, "No password set for this account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*cred.CredentialHash), []byte(req.Password)); err != nil {
		u.logAudit(ctx, domain.AuditActionLogin, user, nil, map[string]interface{}{
			"error": "invalid password",
		})
		return nil, appErrors.New(appErrors.ErrInvalidCredentials, "Invalid email or password")
	}

	// Get user roles
	roles, err := u.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to get user roles", err)
	}

	// Get user permissions
	permissions, err := u.roleRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to get user permissions", err)
	}

	// Generate tokens
	tokenPair, err := u.generateTokenPair(ctx, user, roles, permissions, req)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to generate tokens", err)
	}

	// Update last login
	if err := u.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Update credential last used
	now := time.Now()
	cred.LastUsedAt = &now
	u.userRepo.UpdateCredential(ctx, cred)

	// Log audit
	u.logAudit(ctx, domain.AuditActionLogin, user, nil, map[string]interface{}{
		"device_type": req.DeviceType,
		"success":     true,
	})

	return &domain.AuthResponse{
		User:  user,
		Token: tokenPair,
	}, nil
}

// Register creates a new user account
func (u *AuthUsecase) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	// Check tenant
	tenant, err := u.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Tenant not found", err)
	}

	if !tenant.IsActive() {
		return nil, appErrors.New(appErrors.ErrTenantInactive, "Tenant is not active")
	}

	// Check if email already exists
	existing, err := u.userRepo.GetByEmailAndTenant(ctx, req.Email, req.TenantID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check email", err)
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.ErrConflict, "Email already registered")
	}

	// Check if username already exists
	existing, err = u.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check username", err)
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.ErrConflict, "Username already taken")
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		FullName:        req.FullName,
		Email:           req.Email,
		Username:        req.Username,
		Phone:           req.Phone,
		IsEmailVerified: false,
		IsPhoneVerified: false,
		Status:          domain.UserStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create user", err)
	}

	// Hash password and create credential
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to hash password", err)
	}

	hashStr := string(hashedPassword)
	cred := &domain.Credential{
		ID:             uuid.New(),
		UserID:         user.ID,
		Type:           domain.CredentialTypePassword,
		CredentialHash: &hashStr,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := u.userRepo.CreateCredential(ctx, cred); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create credentials", err)
	}

	// Assign default role (ROLE_USER)
	defaultRole, err := u.roleRepo.GetByCode(ctx, domain.RoleUser, &req.TenantID)
	if err != nil {
		// Log but don't fail registration
		fmt.Printf("Failed to get default role: %v\n", err)
	} else if defaultRole != nil {
		u.roleRepo.AssignRoleToUser(ctx, user.ID, defaultRole.ID, nil, nil)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionRegister, user, &req.TenantID, map[string]interface{}{
		"email": req.Email,
	})

	// Return user without tokens (need to verify email first)
	return &domain.AuthResponse{
		User: user,
	}, nil
}

// RefreshToken refreshes the access token
func (u *AuthUsecase) RefreshToken(ctx context.Context, req domain.RefreshTokenRequest) (*domain.TokenPair, error) {
	// Hash the refresh token to find the session
	hash := sha256.Sum256([]byte(req.RefreshToken))
	hashStr := hex.EncodeToString(hash[:])

	session, err := u.sessionRepo.GetByRefreshTokenHash(ctx, hashStr)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrTokenInvalid, "Invalid refresh token", err)
	}

	if session == nil || session.IsRevoked() {
		return nil, appErrors.New(appErrors.ErrSessionRevoked, "Session has been revoked or expired")
	}

	// Get user
	user, err := u.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to get user", err)
	}

	if !user.IsActive() {
		return nil, appErrors.New(appErrors.ErrUserInactive, "User account is not active")
	}

	// Get roles and permissions
	roles, err := u.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to get roles", err)
	}

	permissions, err := u.roleRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to get permissions", err)
	}

	// Generate new token pair
	claims := auth.Claims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Roles:    roleCodes(roles),
		Permissions: permissionStrings(permissions),
		SessionID: session.ID,
	}

	accessToken, err := auth.GenerateAccessToken(claims, u.jwtConfig.AccessTokenSecret, u.jwtConfig.AccessTokenTTL)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to generate access token", err)
	}

	newRefreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to generate refresh token", err)
	}

	// Update session with new refresh token
	newHash := sha256.Sum256([]byte(newRefreshToken))
	session.RefreshTokenHash = hex.EncodeToString(newHash[:])
	session.AccessTokenID = &claims.ID
	session.UpdateActivity()
	
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to update session", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionTokenRefresh, user, &user.TenantID, nil)

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(u.jwtConfig.AccessTokenTTL.Seconds()),
	}, nil
}

// Logout revokes the current session
func (u *AuthUsecase) Logout(ctx context.Context, sessionID uuid.UUID) error {
	session, err := u.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrNotFound, "Session not found", err)
	}

	if err := u.sessionRepo.Revoke(ctx, sessionID); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to revoke session", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionLogout, &domain.User{ID: session.UserID}, nil, map[string]interface{}{
		"session_id": sessionID,
	})

	return nil
}

// ChangePassword changes user password
func (u *AuthUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, req domain.ChangePasswordRequest) error {
	// Get user's password credential
	cred, err := u.userRepo.GetActiveCredentialByType(ctx, userID, domain.CredentialTypePassword)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to get credentials", err)
	}

	if cred == nil || cred.CredentialHash == nil {
		return appErrors.New(appErrors.ErrBadRequest, "No password set for this account")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(*cred.CredentialHash), []byte(req.OldPassword)); err != nil {
		return appErrors.New(appErrors.ErrInvalidCredentials, "Old password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to hash password", err)
	}

	hashStr := string(hashedPassword)
	cred.CredentialHash = &hashStr
	now := time.Now()
	cred.LastUsedAt = &now
	cred.UpdatedAt = now

	if err := u.userRepo.UpdateCredential(ctx, cred); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to update password", err)
	}

	// Revoke all other sessions
	if err := u.sessionRepo.RevokeAllUserSessions(ctx, userID, nil); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to revoke sessions: %v\n", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionPasswordChange, &domain.User{ID: userID}, nil, nil)

	return nil
}

// GetMe gets current user info with roles
func (u *AuthUsecase) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "User not found", err)
	}

	// Get roles
	roles, err := u.roleRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to get roles", err)
	}
	user.Roles = roles

	// Get profile
	profile, err := u.userRepo.GetProfileByUserID(ctx, userID)
	if err == nil && profile != nil {
		user.Profile = profile
	}

	return user, nil
}

// Helper functions

func (u *AuthUsecase) generateTokenPair(
	ctx context.Context,
	user *domain.User,
	roles []domain.Role,
	permissions []domain.Permission,
	req domain.LoginRequest,
) (*domain.TokenPair, error) {
	// Create claims
	claims := auth.Claims{
		UserID:      user.ID,
		TenantID:    user.TenantID,
		Email:       user.Email,
		Roles:       roleCodes(roles),
		Permissions: permissionStrings(permissions),
	}

	// Generate access token
	accessToken, err := auth.GenerateAccessToken(claims, u.jwtConfig.AccessTokenSecret, u.jwtConfig.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Hash refresh token
	hash := sha256.Sum256([]byte(refreshToken))
	hashStr := hex.EncodeToString(hash[:])

	// Create session
	session := &domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: hashStr,
		AccessTokenID:    &claims.ID,
		DeviceName:       req.DeviceName,
		DeviceType:       req.DeviceType,
		UserAgent:        "", // Will be set from HTTP request
		ExpiresAt:        time.Now().Add(u.jwtConfig.RefreshTokenTTL),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Extract IP if present in context
	if ip, ok := ctx.Value("client_ip").(string); ok {
		session.IPAddress = &ip
	}

	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	// Set session ID in claims
	claims.SessionID = session.ID

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(u.jwtConfig.AccessTokenTTL.Seconds()),
	}, nil
}

func (u *AuthUsecase) logAudit(ctx context.Context, action domain.AuditAction, user *domain.User, tenantID *uuid.UUID, details map[string]interface{}) {
	ip := ""
	if clientIP, ok := ctx.Value("client_ip").(string); ok {
		ip = clientIP
	}

	// Parse IP
	parsedIP := net.ParseIP(ip)
	
	log := &domain.AuditLog{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    &user.ID,
		Action:    action,
		Details:   details,
		IPAddress: func() *string { if parsedIP != nil { s := parsedIP.String(); return &s }; return nil }(),
		CreatedAt: time.Now(),
	}

	if err := u.auditRepo.Create(ctx, log); err != nil {
		fmt.Printf("Failed to create audit log: %v\n", err)
	}
}

func roleCodes(roles []domain.Role) []string {
	codes := make([]string, len(roles))
	for i, role := range roles {
		codes[i] = role.Code
	}
	return codes
}

func permissionStrings(perms []domain.Permission) []string {
	strs := make([]string, len(perms))
	for i, perm := range perms {
		strs[i] = fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
	}
	return strs
}
