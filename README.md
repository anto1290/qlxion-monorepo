# Qlxion Monorepo

Monorepo untuk platform Qlxion - API Gateway dan Auth Service dengan arsitektur microservices.

## Architecture Overview

```
                    ┌─────────────────┐
                    │     Client      │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  API Gateway    │  Port: 8000
                    │  (KrakenD-like) │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼────────┐    │    ┌────────▼────────┐
     │  Auth Service   │    │    │  Other Services │
     │  Port: 8001     │    │    │  (Future)       │
     └─────────────────┘    │    └─────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────▼────────┐          ┌────────▼────────┐
     │   PostgreSQL    │          │     Redis       │
     │    (Auth DB)    │          │  (Cache/Sess)   │
     └─────────────────┘          └─────────────────┘
```

## Project Structure

```
qlxion-monorepo/
├── go.work                          # Go Workspace file
├── Makefile                         # Automation scripts
├── README.md                        # This file
│
├── api-gateway/                     # Custom API Gateway (KrakenD-like)
│   ├── cmd/gateway/main.go          # Entry point
│   ├── internal/
│   │   ├── config/                  # Routing config parser
│   │   ├── middleware/              # Rate limiter, CORS, logging
│   │   ├── proxy/                   # Request forwarding
│   │   └── aggregator/              # Response aggregation
│   ├── plugins/                     # Custom Go plugins
│   ├── gateway.yaml                 # Gateway configuration
│   ├── Dockerfile
│   ├── go.mod
│   └── README.md
│
├── services/
│   └── auth-service/                # SSO & IAM Service
│       ├── cmd/auth/main.go         # Entry point
│       ├── internal/
│       │   ├── delivery/            # HTTP handlers
│       │   ├── usecase/             # Business logic
│       │   ├── domain/              # Entities
│       │   └── repository/          # DB queries
│       ├── migrations/              # Database migrations
│       ├── Dockerfile
│       ├── go.mod
│       └── README.md
│
├── pkg/                             # Shared Libraries
│   ├── auth/                        # JWT validation & claims
│   ├── database/                    # DB connection pooling
│   ├── errors/                      # Custom errors
│   ├── logger/                      # Centralized logging
│   └── response/                    # JSON response format
│
├── deploy/                          # Infrastructure & Deployment
│   ├── docker-compose.yml           # Local development
│   └── k8s/                         # Kubernetes manifests
│       ├── api-gateway/
│       ├── auth-service/
│       └── shared/
│
└── api-docs/                        # OpenAPI/Swagger documentation
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- PostgreSQL 16+ (atau gunakan Docker)
- Redis 7+ (atau gunakan Docker)

### Run with Docker Compose (Recommended)

```bash
# Clone repository
git clone <repository-url>
cd qlxion-monorepo

# Copy environment file
cp .env.example .env
# Edit .env dengan konfigurasi yang sesuai

# Start all services
make docker-up

# Atau menggunakan docker-compose langsung
cd deploy && docker-compose up --build
```

Services akan tersedia di:
- API Gateway: http://localhost:8000
- Auth Service: http://localhost:8001
- PostgreSQL: localhost:5432
- Redis: localhost:6379

### Local Development

#### 1. Start Infrastructure

```bash
# Start PostgreSQL dan Redis saja
cd deploy && docker-compose up -d postgres redis
```

#### 2. Run API Gateway

```bash
# Terminal 1
make dev-gateway
# atau
cd api-gateway && JWT_SECRET=dev-secret go run cmd/gateway/main.go
```

#### 3. Run Auth Service

```bash
# Terminal 2
make dev-auth
# atau
cd services/auth-service && JWT_SECRET=dev-secret DB_PASSWORD=postgres go run cmd/auth/main.go
```

## API Documentation

### API Gateway

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/gateway/info` | GET | Gateway info |
| `/auth/login` | POST | Login |
| `/auth/register` | POST | Register |
| `/auth/refresh` | POST | Refresh token |
| `/auth/me` | GET | Current user |
| `/users` | GET/POST | User management |
| `/roles` | GET/POST | Role management |
| `/tenants` | GET/POST | Tenant management |

Lihat `api-gateway/README.md` dan `services/auth-service/README.md` untuk dokumentasi lengkap.

## Database Migrations

```bash
# Run migrations (otomatis saat pertama kali menjalankan PostgreSQL di Docker)
make migrate-up

# Rollback migrations
make migrate-down

# Create new migration
make migrate-create name=add_new_table
```

## Kubernetes Deployment

```bash
# Setup secrets
cp deploy/k8s/shared/secrets.example.yaml deploy/k8s/shared/secrets.yaml
# Edit secrets.yaml dengan nilai yang sesuai

# Deploy
make k8s-deploy

# Check status
make k8s-status

# View logs
make k8s-logs-gateway
make k8s-logs-auth

# Remove deployment
make k8s-delete
```

## Available Make Commands

```bash
make help              # Show all available commands
make build             # Build all services
make test              # Run all tests
make lint              # Run linter
make docker-build      # Build Docker images
make docker-up         # Start with docker-compose
make docker-down       # Stop docker-compose
make k8s-deploy        # Deploy to Kubernetes
make migrate-up        # Run DB migrations
make clean             # Clean build artifacts
make ci                # Run CI pipeline locally
```

## Technology Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.23 |
| API Gateway | Custom (KrakenD-like) |
| Auth Service | Clean Architecture |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| JWT | golang-jwt/jwt/v5 |
| DB Driver | jackc/pgx/v5 |
| Logging | rs/zerolog |
| Container | Docker |
| Orchestration | Kubernetes |
| Migration | SQL |

## Security

- JWT tokens dengan HS256 signing
- Password hashing dengan bcrypt
- Sliding window rate limiting
- CORS protection
- Audit logging untuk semua operasi penting
- Session management dengan refresh token rotation
- Multi-tenant data isolation

## Contributing

1. Fork repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

This project is proprietary and confidential.
