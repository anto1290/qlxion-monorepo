-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- TENANTS (Multi-tenancy support)
-- ============================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE,
    config JSONB DEFAULT '{}',
    status INT DEFAULT 1, -- 1=ACTIVE, 0=INACTIVE, 2=SUSPENDED
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

COMMENT ON TABLE tenants IS 'Master table untuk multi-tenant support';
COMMENT ON COLUMN tenants.code IS 'Singkatan unik untuk tenant, misal: PTA untuk Perusahaan A';
COMMENT ON COLUMN tenants.domain IS 'Subdomain kustom, opsional';
COMMENT ON COLUMN tenants.config IS 'Pengaturan tenant dalam JSON, misal: session_timeout, login_attempts, dll';
COMMENT ON COLUMN tenants.status IS '1=ACTIVE, 0=INACTIVE, 2=SUSPENDED';

CREATE INDEX idx_tenants_code ON tenants(code);
CREATE INDEX idx_tenants_domain ON tenants(domain) WHERE domain IS NOT NULL;
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;

-- ============================================
-- USERS
-- ============================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    username VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    avatar_url TEXT,
    is_email_verified BOOLEAN DEFAULT FALSE,
    is_phone_verified BOOLEAN DEFAULT FALSE,
    status INT DEFAULT 1, -- 1=ACTIVE, 0=BLOCKED, 2=PENDING
    last_login_at TIMESTAMPTZ,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

COMMENT ON TABLE users IS 'Tabel utama untuk data pengguna';
COMMENT ON COLUMN users.status IS '1=ACTIVE, 0=BLOCKED, 2=PENDING';

CREATE INDEX idx_users_tenant_id ON users(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email_tenant ON users(email, tenant_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_username_unique ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;

-- ============================================
-- PROFILE
-- ============================================
CREATE TABLE profile (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    avatar VARCHAR(255),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(120),
    nik_number VARCHAR(32),
    place_birth VARCHAR(50),
    date_of_birth DATE,
    mobile VARCHAR(15),
    address VARCHAR(100),
    zip_code VARCHAR(10),
    extras JSONB DEFAULT '{}'
);

CREATE INDEX idx_profile_user_id ON profile(user_id);

-- ============================================
-- USER CREDENTIALS
-- ============================================
CREATE TABLE user_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- PASSWORD, OAUTH_GOOGLE, OAUTH_GITHUB, LDAP, dll
    credential_hash VARCHAR(255),
    provider_user_id VARCHAR(255),
    provider_data JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN user_credentials.type IS 'Enum: PASSWORD, OAUTH_GOOGLE, OAUTH_GITHUB, LDAP, dll';
COMMENT ON COLUMN user_credentials.credential_hash IS 'Hash password untuk tipe PASSWORD';
COMMENT ON COLUMN user_credentials.provider_user_id IS 'ID user dari provider OAuth/LDAP';
COMMENT ON COLUMN user_credentials.provider_data IS 'Data tambahan dari provider (token, scope, dll)';

CREATE INDEX idx_credentials_user_id ON user_credentials(user_id);
CREATE INDEX idx_credentials_user_type ON user_credentials(user_id, type);

-- ============================================
-- ROLES
-- ============================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_system_defined BOOLEAN DEFAULT FALSE,
    status INT DEFAULT 1, -- 1=ACTIVE, 0=INACTIVE
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, code)
);

COMMENT ON COLUMN roles.code IS 'Contoh: ROLE_SUPER_ADMIN, ROLE_CHIEF_ENGINEER';
COMMENT ON COLUMN roles.name IS 'Contoh: Super Admin, Chief Engineering';
COMMENT ON COLUMN roles.is_system_defined IS 'Jika true, role bawaan sistem dan tidak bisa dihapus';

CREATE INDEX idx_roles_tenant_id ON roles(tenant_id);
CREATE INDEX idx_roles_code ON roles(code);

-- ============================================
-- PERMISSIONS
-- ============================================
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    description TEXT,
    is_system_defined BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(resource, action)
);

COMMENT ON COLUMN permissions.resource IS 'Contoh: course, user, billing';
COMMENT ON COLUMN permissions.action IS 'Contoh: create, read, update, delete';

CREATE INDEX idx_permissions_resource ON permissions(resource);

-- ============================================
-- ROLE PERMISSIONS
-- ============================================
CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    assigned_by UUID,
    PRIMARY KEY (role_id, permission_id)
);

COMMENT ON COLUMN role_permissions.assigned_by IS 'User ID yang memberikan permission';

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

-- ============================================
-- USER ROLES
-- ============================================
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    assigned_by UUID,
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

COMMENT ON COLUMN user_roles.expires_at IS 'Jika tidak null, role hanya berlaku hingga tanggal tersebut';

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE INDEX idx_user_roles_expires ON user_roles(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================
-- SESSIONS
-- ============================================
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) UNIQUE NOT NULL,
    access_token_id VARCHAR(255) UNIQUE,
    device_name VARCHAR(255),
    device_type VARCHAR(50), -- mobile, desktop, web
    ip_address INET,
    user_agent TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_activity_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN sessions.refresh_token_hash IS 'Hash dari refresh token untuk keamanan';
COMMENT ON COLUMN sessions.access_token_id IS 'JTI dari JWT access token (opsional)';

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_hash ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_active ON sessions(user_id) WHERE revoked_at IS NULL AND expires_at > NOW();

-- ============================================
-- CLIENTS (OAuth2)
-- ============================================
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id VARCHAR(255) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    redirect_uris TEXT[] NOT NULL,
    grant_types VARCHAR[] NOT NULL,
    scope VARCHAR[],
    is_active BOOLEAN DEFAULT TRUE,
    access_token_ttl INT DEFAULT 3600, -- detik
    refresh_token_ttl INT DEFAULT 86400,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN clients.client_secret_hash IS 'Hash client secret';
COMMENT ON COLUMN clients.grant_types IS 'authorization_code, password, client_credentials, dll';

CREATE INDEX idx_clients_tenant ON clients(tenant_id);
CREATE INDEX idx_clients_client_id ON clients(client_id);

-- ============================================
-- IDENTITY PROVIDERS (OAuth/SAML)
-- ============================================
CREATE TABLE identity_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- google, github, facebook, dll
    client_id VARCHAR(255) NOT NULL,
    client_secret VARCHAR(255) NOT NULL,
    authorization_endpoint TEXT,
    token_endpoint TEXT,
    userinfo_endpoint TEXT,
    scopes VARCHAR[],
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN identity_providers.provider IS 'google, github, facebook, dll';
COMMENT ON COLUMN identity_providers.client_secret IS 'Dienkripsi';

CREATE INDEX idx_idp_tenant ON identity_providers(tenant_id);
CREATE INDEX idx_idp_provider ON identity_providers(tenant_id, provider);

-- ============================================
-- AUDIT LOGS
-- ============================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL, -- LOGIN, LOGOUT, ROLE_ASSIGN, PERMISSION_CHANGE, dll
    resource VARCHAR(100),
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

COMMENT ON COLUMN audit_logs.action IS 'LOGIN, LOGOUT, ROLE_ASSIGN, PERMISSION_CHANGE, dll';

CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource, resource_id);

-- Partition untuk audit_logs (opsional, untuk performa)
-- CREATE TABLE audit_logs_y2024m01 PARTITION OF audit_logs
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ============================================
-- USER ATTRIBUTES
-- ============================================
CREATE TABLE user_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    value JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, key)
);

CREATE INDEX idx_user_attributes_user ON user_attributes(user_id);

-- ============================================
-- SEED DATA
-- ============================================

-- Insert default tenant
INSERT INTO tenants (id, code, name, status, config) VALUES
    ('00000000-0000-0000-0000-000000000001', 'SYSTEM', 'System Tenant', 1, 
     '{"session_timeout": 30, "max_login_attempts": 5, "password_min_length": 8, "require_strong_password": true}');

-- Insert system roles
INSERT INTO roles (id, tenant_id, code, name, description, is_system_defined, status) VALUES
    ('00000000-0000-0000-0000-000000000010', NULL, 'ROLE_SUPER_ADMIN', 'Super Admin', 'Full system access', true, 1),
    ('00000000-0000-0000-0000-000000000020', NULL, 'ROLE_ADMIN', 'Administrator', 'Tenant administrator', true, 1),
    ('00000000-0000-0000-0000-000000000030', NULL, 'ROLE_USER', 'User', 'Standard user', true, 1),
    ('00000000-0000-0000-0000-000000000040', NULL, 'ROLE_TENANT_ADMIN', 'Tenant Admin', 'Tenant-specific admin', true, 1),
    ('00000000-0000-0000-0000-000000000050', NULL, 'ROLE_TENANT_MANAGER', 'Tenant Manager', 'Tenant manager', true, 1);

-- Insert system permissions
INSERT INTO permissions (id, resource, action, description, is_system_defined) VALUES
    -- User permissions
    ('00000000-0000-0000-0000-000000000100', 'user', 'create', 'Create users', true),
    ('00000000-0000-0000-0000-000000000101', 'user', 'read', 'Read users', true),
    ('00000000-0000-0000-0000-000000000102', 'user', 'update', 'Update users', true),
    ('00000000-0000-0000-0000-000000000103', 'user', 'delete', 'Delete users', true),
    -- Role permissions
    ('00000000-0000-0000-0000-000000000200', 'role', 'create', 'Create roles', true),
    ('00000000-0000-0000-0000-000000000201', 'role', 'read', 'Read roles', true),
    ('00000000-0000-0000-0000-000000000202', 'role', 'update', 'Update roles', true),
    ('00000000-0000-0000-0000-000000000203', 'role', 'delete', 'Delete roles', true),
    -- Tenant permissions
    ('00000000-0000-0000-0000-000000000300', 'tenant', 'create', 'Create tenants', true),
    ('00000000-0000-0000-0000-000000000301', 'tenant', 'read', 'Read tenants', true),
    ('00000000-0000-0000-0000-000000000302', 'tenant', 'update', 'Update tenants', true),
    ('00000000-0000-0000-0000-000000000303', 'tenant', 'delete', 'Delete tenants', true),
    -- Session permissions
    ('00000000-0000-0000-0000-000000000400', 'session', 'read', 'Read sessions', true),
    ('00000000-0000-0000-0000-000000000401', 'session', 'revoke', 'Revoke sessions', true),
    -- Audit permissions
    ('00000000-0000-0000-0000-000000000500', 'audit', 'read', 'Read audit logs', true);

-- Assign all permissions to SUPER_ADMIN
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', id FROM permissions;

-- Assign user/role read to ADMIN
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000100'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000101'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000102'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000200'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000201'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000202'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000400'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000401'),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000500');

-- Assign basic read to USER
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000030', '00000000-0000-0000-0000-000000000101'),
    ('00000000-0000-0000-0000-000000000030', '00000000-0000-0000-0000-000000000201'),
    ('00000000-0000-0000-0000-000000000030', '00000000-0000-0000-0000-000000000400');
