# api-gateway

**Port**: 8000 | **DB**: None | **Dependencies**: Redis (Rate Limiting)

## Deskripsi

Custom API Gateway (KrakenD-like) yang bertindak sebagai pintu masuk utama (Ingress point). Gateway ini bertugas membaca konfigurasi routing terpusat dan meneruskan request ke internal microservices.

## Core Features

- **Routing & Proxy**: Forwarding request eksternal ke internal microservices.
- **Middleware**: Rate limiting, CORS, dan request logging terpusat.
- **Auth Aggregation**: Validasi JWT Token secara terpusat sebelum diteruskan ke service di belakangnya.
- **Response Aggregation**: Menggabungkan response dari beberapa service menjadi satu response.
- **Load Balancing**: Round-robin dan health check untuk backend services.

## Architecture

```
Client Request
    |
    v
[Rate Limiter] --> Redis (Sliding Window)
    |
    v
[CORS Handler]
    |
    v
[Request Logger]
    |
    v
[Auth Middleware] --> JWT Validation (pkg/auth)
    |
    v
[Router] --> Match endpoint config
    |
    v
[Proxy] --> Forward to backend service
    |
    v
[Aggregator] (optional)
    |
    v
Backend Service
```

## Konfigurasi

### Environment Variables

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `GATEWAY_HOST` | `0.0.0.0` | Host binding |
| `GATEWAY_PORT` | `8000` | Port binding |
| `GATEWAY_DEBUG` | `false` | Enable debug mode |
| `JWT_SECRET` | - | Secret key untuk JWT validation |
| `REDIS_HOST` | `localhost` | Redis host untuk rate limiting |
| `REDIS_PORT` | `6379` | Redis port |
| `RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |
| `CORS_ENABLED` | `true` | Enable CORS |

### File Konfigurasi (YAML)

Gateway juga mendukung konfigurasi via file YAML. Set path dengan environment variable `GATEWAY_CONFIG_PATH`.

Lihat `gateway.yaml` untuk contoh konfigurasi lengkap.

## Endpoints

### Gateway Management

| Method | Path | Deskripsi | Auth |
|--------|------|-----------|------|
| GET | `/health` | Health check gateway | No |
| GET | `/gateway/info` | Info services terdaftar | No |

### Auth Service (Proxied)

| Method | Path | Deskripsi | Auth |
|--------|------|-----------|------|
| POST | `/auth/login` | Login user | No |
| POST | `/auth/register` | Register user | No |
| POST | `/auth/refresh` | Refresh token | No |
| POST | `/auth/logout` | Logout | Yes |
| GET | `/auth/me` | Get current user | Yes |
| POST | `/auth/password/change` | Change password | Yes |

### User Management (Proxied)

| Method | Path | Deskripsi | Auth | Role |
|--------|------|-----------|------|------|
| GET | `/users` | List users | Yes | ADMIN, SUPER_ADMIN |
| POST | `/users` | Create user | Yes | ADMIN, SUPER_ADMIN |
| GET | `/users/{id}` | Get user | Yes | - |
| PUT | `/users/{id}` | Update user | Yes | - |
| DELETE | `/users/{id}` | Delete user | Yes | ADMIN, SUPER_ADMIN |

### Role & Permission Management (Proxied)

| Method | Path | Deskripsi | Auth | Role |
|--------|------|-----------|------|------|
| GET | `/roles` | List roles | Yes | - |
| POST | `/roles` | Create role | Yes | SUPER_ADMIN |
| GET | `/roles/{id}` | Get role | Yes | - |
| PUT | `/roles/{id}` | Update role | Yes | SUPER_ADMIN |
| DELETE | `/roles/{id}` | Delete role | Yes | SUPER_ADMIN |
| GET | `/permissions` | List permissions | Yes | - |

### Tenant Management (Proxied)

| Method | Path | Deskripsi | Auth | Role |
|--------|------|-----------|------|------|
| GET | `/tenants` | List tenants | Yes | SUPER_ADMIN |
| POST | `/tenants` | Create tenant | Yes | SUPER_ADMIN |
| GET | `/tenants/{id}` | Get tenant | Yes | SUPER_ADMIN |
| PUT | `/tenants/{id}` | Update tenant | Yes | SUPER_ADMIN |

### Session & Audit (Proxied)

| Method | Path | Deskripsi | Auth | Role |
|--------|------|-----------|------|------|
| GET | `/sessions` | List sessions | Yes | - |
| POST | `/sessions/{id}/revoke` | Revoke session | Yes | - |
| GET | `/audit-logs` | View audit logs | Yes | ADMIN |

## Middleware Stack

1. **Recovery** - Panic recovery
2. **Rate Limiting** - Sliding window rate limiter (Redis-backed)
3. **CORS** - Cross-origin resource sharing
4. **Request Logger** - Structured logging dengan zerolog
5. **Auth** - JWT validation (optional per endpoint)
6. **Proxy** - Request forwarding ke backend

## Headers yang Diteruskan

Setelah validasi JWT, gateway akan menambahkan header berikut ke request backend:

| Header | Deskripsi |
|--------|-----------|
| `X-User-Id` | UUID user yang terautentikasi |
| `X-Tenant-Id` | UUID tenant user |
| `X-User-Email` | Email user |
| `X-User-Roles` | Comma-separated list of roles |

## Relasi ke Service Lain

- Menerima JWT dari Header `Authorization`, memvalidasinya menggunakan `pkg/auth`, dan meneruskan claims (seperti `X-User-Id`, `X-Tenant-Id`) ke backend services.
- Gateway tidak memiliki database sendiri, hanya menggunakan Redis untuk rate limiting.
- Semua autentikasi dan otorisasi business logic ditangani oleh `auth-service`.

## Running

### Local Development

```bash
# Set environment variables
export JWT_SECRET="your-secret-key"
export REDIS_HOST="localhost"

# Run gateway
cd api-gateway
go run cmd/gateway/main.go
```

### With Config File

```bash
export GATEWAY_CONFIG_PATH="./gateway.yaml"
go run cmd/gateway/main.go
```

### Docker

```bash
docker build -t qlxion-api-gateway .
docker run -p 8000:8000 -e JWT_SECRET=secret qlxion-api-gateway
```

## Development

### Menambah Service Baru

1. Tambahkan service ke `gateway.yaml`:

```yaml
services:
  - name: "new-service"
    host: "localhost"
    port: 8002
    endpoints:
      - path: "/api/resource"
        method: "GET"
        backend: "new-service"
        backend_path: "/v1/resource"
        requires_auth: true
```

2. Restart gateway untuk memuat konfigurasi baru.
