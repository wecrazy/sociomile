# Sociomile Operational Reference

[Bahasa Indonesia](REFERENCE.md) | English | [README](../README.en.md)

This document summarizes the repository layout, primary commands, environment variables, and the API surface.

## Repository Layout

- `backend/` backend service, worker, migrations, seeds, and the OpenAPI file
- `frontend/` React operator UI
- `docker-compose.yml` local stack definition for Podman-compatible compose
- `Makefile` single entry point for setup, run, build, test, migrate, seed, and coverage
- `.env.example` shared root env template for local workflows
- `.env.compose.example` compose-only env template

## Reviewer Quick Reference

### Fast Review Path

1. `git clone https://github.com/wecrazy/sociomile.git`
1. `cd sociomile`
1. `make env`
1. `make setup`
1. `make dev`
1. `make migrate && make seed`
1. Open `http://localhost:5173`
1. Sign in with `alice.admin@acme.local` / `Password123!`

### Key URLs

| Surface | URL |
| --- | --- |
| Frontend | `http://localhost:5173` |
| Backend health | `http://localhost:8080/health` |
| Swagger UI | `http://localhost:8080/swagger` |
| RabbitMQ management | `http://localhost:15672` |

## Local Defaults

- Frontend: `5173`
- Backend: `8080`
- MySQL: `13306`
- Redis: `16379`
- RabbitMQ AMQP: `5672`
- RabbitMQ management: `15672`

## Demo Accounts

The seed command creates these users with password `Password123!`:

- `alice.admin@acme.local`
- `aaron.agent@acme.local`
- `grace.admin@globex.local`
- `gina.agent@globex.local`

## Makefile Commands

| Command | Purpose |
| --- | --- |
| `make help` | Show all available targets |
| `make env` | Create local env files without overwriting existing ones |
| `make setup` | Install backend and frontend dependencies |
| `make dev` | Build and run the full local stack |
| `make dev-down` | Stop and remove the local stack |
| `make dev-logs` | Stream compose logs |
| `make config` | Print the resolved compose configuration from `.env` and `.env.compose` |
| `make migrate` | Apply backend SQL migrations |
| `make seed` | Load demo tenants, users, and channels |
| `make fmt` | Format backend Go sources and frontend source or config files |
| `make backend-fmt` | Format backend Go files with `gofmt` |
| `make frontend-fmt` | Format frontend source and config files with `Prettier` |
| `make backend-lint` | Run backend `gofmt` check, `go vet`, and `revive` |
| `make backend-test` | Run backend tests |
| `make frontend-test` | Run frontend tests |
| `make backend-coverage` | Run backend coverage and generate `coverage/backend.html` |
| `make frontend-coverage` | Run frontend coverage and generate reports in `coverage/frontend/` |
| `make coverage` | Run the full backend and frontend coverage workflow |
| `make lint` | Run backend Go linting, frontend formatting checks, and frontend TypeScript linting |
| `make build` | Build container images |
| `make swagger` | Print the Swagger UI hint |

## Environment Variables

The local workflow uses two root env files:

- `.env` for shared values and local secrets
- `.env.compose` for compose-only internal container wiring

`backend/.env` and `frontend/.env` remain the host-run configuration files.

### Root Shared Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `sociomile` | Compose project namespace |
| `MYSQL_DATABASE` | `sociomile` | App database name |
| `MYSQL_USER` | `sociomile` | App MySQL user |
| `MYSQL_PASSWORD` | `sociomile` | App MySQL password |
| `MYSQL_ROOT_PASSWORD` | `root` | MySQL root password |
| `MYSQL_PORT` | `13306` | Host port mapped to MySQL |
| `REDIS_PORT` | `16379` | Host port mapped to Redis |
| `RABBITMQ_PORT` | `5672` | Host port mapped to RabbitMQ AMQP |
| `RABBITMQ_MANAGEMENT_PORT` | `15672` | Host port mapped to RabbitMQ UI |
| `RABBITMQ_DEFAULT_USER` | `guest` | RabbitMQ login |
| `RABBITMQ_DEFAULT_PASS` | `guest` | RabbitMQ password |
| `BACKEND_PORT` | `8080` | Host port mapped to backend |
| `FRONTEND_PORT` | `5173` | Host port mapped to frontend |
| `JWT_SECRET` | `sociomile-local-dev-secret` | JWT signing secret for compose runtime |
| `ACCESS_TOKEN_TTL` | `15m` | JWT expiration duration |
| `APP_ENV` | `development` | Runtime environment flag |
| `LOG_LEVEL` | `debug` | Backend and worker log verbosity |

### Root Compose Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `COMPOSE_MYSQL_DSN` | `sociomile:sociomile@tcp(mysql:3306)/sociomile?...` | Internal DSN used by backend and worker on the compose network |
| `COMPOSE_REDIS_ADDR` | `redis:6379` | Internal Redis address on the compose network |
| `COMPOSE_RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | Internal RabbitMQ URL used by backend and worker |
| `COMPOSE_VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | Browser-reachable API base URL used by the frontend in the compose stack |
| `COMPOSE_VITE_APP_NAME` | `Sociomile` | Frontend app name inside compose |
| `COMPOSE_SWAGGER_FILE` | `./docs/openapi.yaml` | OpenAPI file path used by the backend container |

If any compose-required variable is missing, `make dev`, `make build`, and `podman-compose config` fail immediately with an error that names the missing variable.

Note for local MySQL: the MySQL container only provisions the database and user from env on the first startup of an empty data volume. If `MYSQL_DATABASE`, `MYSQL_USER`, or the password changes after `sociomile_mysql_data` already exists, remove that volume and restart the stack so provisioning can run again.

### Running Compose Directly

If you want to bypass the Makefile, source `.env` first and then pass `.env.compose` to the compose command:

```bash
set -a
. ./.env
set +a
podman compose --env-file ./.env.compose config
podman compose --env-file ./.env.compose up --build
```

If your environment uses `podman-compose` directly, the same `--env-file` flag applies:

```bash
set -a
. ./.env
set +a
podman-compose --env-file ./.env.compose config
```

### Backend Runtime Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `APP_ENV` | `development` | Runtime environment |
| `BACKEND_PORT` | `8080` | API listen port |
| `MYSQL_DSN` | `sociomile:sociomile@tcp(localhost:13306)/sociomile?...` | Host-side DSN for migrations, seed, and local runs |
| `REDIS_ADDR` | `localhost:16379` | Host-side Redis address |
| `REDIS_PASSWORD` | empty | Redis password |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | Host-side RabbitMQ connection URL |
| `JWT_SECRET` | `sociomile-local-dev-secret` | JWT signing secret |
| `ACCESS_TOKEN_TTL` | `15m` | JWT expiration |
| `LOG_LEVEL` | `debug` | Log verbosity |
| `SWAGGER_FILE` | `./docs/openapi.yaml` | Static OpenAPI source served by Swagger UI |

### Frontend Runtime Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | Browser API base URL |
| `VITE_APP_NAME` | `Sociomile` | UI app title |

## API Summary

### Public Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Health probe |
| `GET` | `/swagger` | Swagger UI |
| `POST` | `/api/v1/auth/login` | Email and password login |
| `POST` | `/api/v1/channel/webhook` | Simulated inbound channel message |

### Authenticated Endpoints

| Method | Path | Role | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/auth/me` | admin, agent | Current user payload |
| `GET` | `/api/v1/users/agents` | admin, agent | Tenant-scoped active agent list |
| `GET` | `/api/v1/conversations` | admin, agent | List conversations with server-side filters |
| `GET` | `/api/v1/conversations/:id` | admin, agent | Conversation detail with message thread |
| `POST` | `/api/v1/conversations/:id/messages` | agent | Agent reply |
| `PATCH` | `/api/v1/conversations/:id/assign` | admin | Assign conversation to agent |
| `PATCH` | `/api/v1/conversations/:id/close` | admin, agent | Close conversation |
| `POST` | `/api/v1/conversations/:id/escalate` | agent | Escalate conversation into ticket |
| `GET` | `/api/v1/tickets` | admin, agent | List tickets with server-side filters |
| `GET` | `/api/v1/tickets/:id` | admin, agent | Ticket detail |
| `PATCH` | `/api/v1/tickets/:id/status` | admin | Update ticket status |
