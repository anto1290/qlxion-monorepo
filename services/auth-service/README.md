# auth-service

**Port**: 8001 | **DB**: `auth_db` | **Dependencies**: Redis (Session/Cache)

## Deskripsi

Pusat Single Sign-On (SSO) dan Identity Access Management. Service ini menjadi Single Source of Truth untuk autentikasi.

## Authentication & Authorization

- `POST /auth/login` -> Validasi kredensial, generate JWT.
- Ketika user login, service ini meng-generate JWT yang berisi claims standar (User ID, Role, dan Scope/Permission).
- `POST /auth/refresh` -> Refresh JWT token menggunakan valid refresh_token.

## User & Role Management (Multi-Tenant)

- `POST /users` -> Registrasi user baru di bawah tenant tertentu.
- `POST /roles` -> Membuat role spesifik per tenant.
- `POST /permissions` -> Mapping permission ke role.

## Relasi ke Service Lain

- Berperan sebagai penerbit (Issuer) token JWT yang akan dikonsumsi oleh `api-gateway`.
- Service lain tidak menembak langsung ke database auth, melainkan percaya pada claims JWT yang sudah divalidasi oleh gateway.

## Architecture

```
HTTP Request
    |
    v
[Delivery/Handler] --> HTTP REST API
    |
    v
[Usecase] --> Business Logic (Login, Register, RBAC)
    |
    v
[Repository] --> Database queries
    |
    v
[PostgreSQL / Redis]
```

## Database Schema

### Tables

| Table | Description |
|-------|-------------|
| `tenants` | Multi-tenant master data |
| `users` | User accounts |
| `profile` | Extended user profiles |
| `user_credentials` | Authentication credentials (password, OAuth, LDAP) |
| `roles` | Role definitions |
| `permissions` | Permission definitions |
| `role_permissions` | Role-permission mapping |
| `user_roles` | User-role mapping |
| `sessions` | Active session tracking |
| `clients` | OAuth2 client applications |
| `identity_providers` | External OAuth/SAML providers |
| `audit_logs` | Audit trail |
| `user_attributes` | Dynamic user attributes |

### Entity Relationship

```
tenants ||--o{ users : contains
users ||--o{ user_credentials : has
users ||--o{ profile : has
users ||--o{ user_roles : assigned
users ||--o{ sessions : creates
users ||--o{ user_attributes : has
users ||--o{ audit_logs : generates
roles ||--o{ role_permissions : has
roles ||--o{ user_roles : assigned_to
permissions ||--o{ role_permissions : mapped_to
tenants ||--o{ roles : defines
tenants ||--o{ clients : owns
tenants ||--o{ identity_providers : configures
```

## API Endpoints

### Authentication

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/v1/auth/login` | Login with email/password | No |
| POST | `/v1/auth/register` | Register new user | No |
| POST | `/v1/auth/refresh` | Refresh access token | No |
| POST | `/v1/auth/logout` | Logout (revoke session) | Yes |
| GET | `/v1/auth/me` | Get current user info | Yes |
| POST | `/v1/auth/password/change` | Change password | Yes |

### Users

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/users` | List users | Yes |
| POST | `/v1/users` | Create user | Yes |
| GET | `/v1/users/{id}` | Get user | Yes |
| PUT | `/v1/users/{id}` | Update user | Yes |
| DELETE | `/v1/users/{id}` | Delete user | Yes |
| GET | `/v1/users/{id}/roles` | Get user roles | Yes |
| POST | `/v1/users/{id}/roles` | Assign role | Yes |
| DELETE | `/v1/users/{id}/roles/{roleId}` | Remove role | Yes |

### Roles

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/roles` | List roles | Yes |
| POST | `/v1/roles` | Create role | Yes |
| GET | `/v1/roles/{id}` | Get role | Yes |
| PUT | `/v1/roles/{id}` | Update role | Yes |
| DELETE | `/v1/roles/{id}` | Delete role | Yes |
| GET | `/v1/roles/{id}/permissions` | Get role permissions | Yes |
| POST | `/v1/roles/{id}/permissions` | Assign permission | Yes |
| DELETE | `/v1/roles/{id}/permissions/{permId}` | Remove permission | Yes |

### Permissions

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/permissions` | List permissions | Yes |
| POST | `/v1/permissions` | Create permission | Yes |

### Tenants

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/tenants` | List tenants | Yes |
| POST | `/v1/tenants` | Create tenant | Yes |
| GET | `/v1/tenants/{id}` | Get tenant | Yes |
| PUT | `/v1/tenants/{id}` | Update tenant | Yes |
| DELETE | `/v1/tenants/{id}` | Delete tenant | Yes |

### Sessions

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/sessions` | List sessions | Yes |
| POST | `/v1/sessions/{id}/revoke` | Revoke session | Yes |

### Audit Logs

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/audit-logs` | List audit logs | Yes |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_SERVICE_HOST` | `0.0.0.0` | Service host |
| `AUTH_SERVICE_PORT` | `8001` | Service port |
| `AUTH_SERVICE_DEBUG` | `false` | Debug mode |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `auth_db` | PostgreSQL database |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | - | Redis password |
| `JWT_SECRET` | - | JWT signing secret |

## Running

### Local Development

```bash
# Set environment variables
export JWT_SECRET="your-secret-key"
export DB_PASSWORD="postgres"

# Run migrations first
psql -h localhost -U postgres -d auth_db -f migrations/001_init.up.sql

# Run service
cd services/auth-service
go run cmd/auth/main.go
```

### Docker

```bash
docker build -t qlxion-auth-service -f services/auth-service/Dockerfile .
docker run -p 8001:8001 \
  -e JWT_SECRET=secret \
  -e DB_HOST=host.docker.internal \
  qlxion-auth-service
```

## Predefined Roles

| Role Code | Name | Description |
|-----------|------|-------------|
| `ROLE_SUPER_ADMIN` | Super Admin | Full system access |
| `ROLE_ADMIN` | Administrator | Tenant administrator |
| `ROLE_USER` | User | Standard user |
| `ROLE_TENANT_ADMIN` | Tenant Admin | Tenant-specific admin |
| `ROLE_TENANT_MANAGER` | Tenant Manager | Tenant manager |

## Predefined Permissions

| Resource | Actions |
|----------|---------|
| `user` | create, read, update, delete |
| `role` | create, read, update, delete |
| `tenant` | create, read, update, delete |
| `session` | read, revoke |
| `audit` | read |
