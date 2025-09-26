# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the **42Tokyo Tuning the backend Contest 2025** codebase - a performance optimization competition focused on a warehouse robot delivery management system. The application consists of a React/Next.js frontend, Go backend, MySQL database, and Nginx reverse proxy, all containerized with Docker.

## Architecture

```
.
├── webapp/           # Main application
│   ├── frontend/     # Next.js React application (TypeScript)
│   ├── backend/      # Go API server with Chi router
│   ├── mysql/        # Database configuration and data
│   ├── nginx/        # Reverse proxy configuration
│   └── e2e/          # Playwright E2E tests
├── benchmarker/      # Load testing and scoring tools
└── documents/        # Competition documentation (Japanese)
```

### Backend Architecture (Go)
- **Framework**: Chi router (github.com/go-chi/chi/v5)
- **Database**: MySQL with sqlx ORM (github.com/jmoiron/sqlx)
- **Observability**: OpenTelemetry tracing with Jaeger
- **Structure**: Clean architecture pattern
  - `internal/handler/` - HTTP request handlers
  - `internal/service/` - Business logic layer
  - `internal/repository/` - Data access layer
  - `internal/model/` - Domain models
  - `internal/middleware/` - HTTP middleware
  - `internal/db/` - Database connection management
  - `internal/telemetry/` - Tracing configuration

### Frontend Architecture (Next.js)
- **Framework**: Next.js 15.4.5 with React 19
- **Styling**: Material-UI (@mui/material) + Tailwind CSS
- **Language**: TypeScript
- **Package Manager**: Yarn

## Common Development Commands

### Environment Setup
```bash
# First-time setup (VM environment)
bash init.sh

# First-time setup (local environment)
bash init.sh {VM_PUBLIC_IP} {PRIVATE_KEY_PATH}
```

### Running the Complete Evaluation Pipeline
```bash
# Full pipeline: restore data, run E2E tests, load test, scoring
bash run.sh
```

### Individual Development Commands

#### Backend Development (Go)
```bash
cd webapp/backend

# Build Go binary
go build -o bin/backend cmd/main.go

# Run tests
go test ./...

# Run with live reload (development)
go run cmd/main.go
```

#### Frontend Development (Next.js)
```bash
cd webapp/frontend

# Install dependencies
yarn install

# Development server with turbopack
yarn dev

# Build for production
yarn build

# Start production server
yarn start

# Lint code
yarn lint
```

#### Container Management
```bash
cd webapp

# Restart containers with current code
bash restart_container.sh

# Use docker-compose directly
docker-compose up -d
docker-compose down
```

#### Database Operations
```bash
# Restore database and run migrations
bash restore_and_migration.sh [sql_file_name]

# Direct MySQL access
docker exec -it tuning-mysql mysql -u root -p
```

#### Testing
```bash
# Run E2E tests
cd webapp/e2e
bash run_e2e_test.sh [DATA_INDEX]

# Run load tests and scoring
cd benchmarker
bash run_k6_and_score.sh
```

### Environment-Specific Commands

#### Local Development
- Uses `webapp/docker-compose.local.yml`
- Frontend runs on http://localhost:3000
- Backend API on http://localhost:8080
- Database on localhost:3306

#### VM Environment  
- Uses `webapp/docker-compose.yml`
- Accessible via https://{hostname}.ftt2508.dabaas.net
- Includes SSL certificates and external networking

## Key Services and Ports

- **Frontend**: Port 3000 (Next.js)
- **Backend**: Port 8080 (Go API server)
- **Database**: Port 3306 (MySQL)
- **Nginx**: Port 443 (HTTPS reverse proxy)
- **Jaeger**: Port 16686 (Tracing UI)

## Application Domain

The system manages warehouse robot deliveries with these core entities:
- **Orders**: Store requests for products with quantities
- **Products**: Items stored in warehouses
- **Robots**: Transport robots that carry orders
- **Routes**: Delivery paths between warehouses and stores

Key use cases (as per documents/usecases.md):
1. Order registration (stores request products)
2. Transport planning (robots receive delivery instructions)
3. Delivery completion notifications (warehouse and store arrivals)
4. Delivery status tracking

## Performance Considerations

This is a performance tuning contest, so focus on:
- Database query optimization (MySQL indexes, query structure)
- Go backend performance (goroutine management, memory allocation)
- Frontend rendering optimization (React.memo, useMemo, virtualization)
- Caching strategies
- Database connection pooling
- Proper HTTP caching headers

## Testing and Evaluation

The competition uses a strict evaluation pipeline:
1. **Database restore**: Fresh data loaded from SQL files
2. **E2E testing**: Playwright tests verify functionality
3. **Load testing**: 240-second load test via benchmarker
4. **Scoring**: Performance metrics calculated

Always run the full pipeline (`bash run.sh`) before submissions to ensure functionality is preserved during optimization.