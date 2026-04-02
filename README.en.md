# Sociomile

| |
| :--: |
| ![Sociomile](docs/readme-mark.svg) |
| **Take-home fullstack for a multi-tenant omnichannel support workflow.** |
| [![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-0f766e.svg)](LICENSE) ![Go 1.26+](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white) ![Go Fiber v3](https://img.shields.io/badge/Fiber-v3-00B2A9?logo=go&logoColor=white) ![React 19](https://img.shields.io/badge/React-19-20232A?logo=react&logoColor=61DAFB) ![Vite 6](https://img.shields.io/badge/Vite-6-646CFF?logo=vite&logoColor=white) |
| ![MySQL 8.4](https://img.shields.io/badge/MySQL-8.4-4479A1?logo=mysql&logoColor=white) ![Redis 7.4](https://img.shields.io/badge/Redis-7.4-DC382D?logo=redis&logoColor=white) ![RabbitMQ 3.13](https://img.shields.io/badge/RabbitMQ-3.13-FF6600?logo=rabbitmq&logoColor=white) ![Podman Compose](https://img.shields.io/badge/Podman-Compose-892CA0?logo=podman&logoColor=white) |
| [Bahasa Indonesia](README.md) \| English |

Sociomile is a fullstack take-home implementation of a multi-tenant omnichannel support flow:

`Channel webhook -> Conversation -> Assignment -> Agent reply -> Ticket escalation`

The repository ships a Go Fiber v3 backend, a separate async worker, a React + Vite operator UI, MySQL, Redis, and RabbitMQ, all wrapped by a Podman-compatible local workflow.

## Stack And License

- Project license: Apache License 2.0.
- Core application stack: Go 1.26, Go Fiber v3, GORM, React 19, Vite 6, and TypeScript 5.
- Local infrastructure stack: MySQL 8.4, Redis 7.4, RabbitMQ 3.13, and Podman Compose.
- Selected upstream licenses: Go Fiber `MIT`, GORM `MIT`, React `MIT`, and Vite `MIT`. Container infrastructure services follow their own upstream licenses.

## Delivered Scope

- JWT login with role-aware access for `admin` and `agent`
- Public channel webhook that creates or reuses conversations per tenant and channel
- Conversation queue with server-side offset pagination and filtering
- Admin assignment flow and agent reply flow
- Ticket escalation with one-ticket-per-conversation enforcement
- Admin-only ticket status updates
- YAML-based locales with English default and Indonesian secondary locale
- Dark and light theme preference persistence
- Activity log plus outbox event persistence, published by a worker through RabbitMQ
- Swagger UI backed by `backend/docs/openapi.yaml`

## Workflow Diagram

```mermaid
flowchart LR
    A[Channel webhook] --> B{Active conversation exists?}
    B -- No --> C[Create new conversation]
    B -- Yes --> D[Reuse existing conversation]
    C --> E[Store customer message]
    D --> E
    E --> F[Admin assigns agent]
    F --> G[Agent replies]
    G --> H[Agent escalates to ticket]
    H --> I[Create ticket]
    I --> J[Admin updates ticket status]
```

## Architecture Diagram

```mermaid
flowchart TB
    EXT[Simulated channel webhook] --> API[Go Fiber API]
    FE[React + Vite Frontend] --> API
    API --> MYSQL[(MySQL)]
    API --> REDIS[(Redis)]
    API --> OUTBOX[Activity log + outbox]
    OUTBOX --> MYSQL
    WORKER[Go Worker] --> MYSQL
    WORKER --> RMQ[(RabbitMQ)]
```

## Assignment Audit

| Area                           | Status     | Notes                                                                                                           |
| ------------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------- |
| Authentication & authorization | `Complete` | JWT login, `admin` and `agent` roles, endpoint protection, and service-layer authorization checks are in place  |
| Channel -> conversation flow   | `Complete` | The webhook creates or reuses an active conversation and stores inbound customer messages                       |
| Conversation management        | `Complete` | List, detail, reply, assign, close, filtering, and pagination are implemented                                   |
| Escalation to ticket           | `Complete` | Only agents can escalate, one conversation can create only one ticket, and only admins can update ticket status |
| Multi-tenancy                  | `Complete` | Tenant isolation is enforced via JWT claims, services, repositories, and seeded data                            |
| Database & migrations          | `Complete` | SQL migrations include the relevant foreign keys and indexes                                                    |
| Async events & worker          | `Complete` | `activity_logs` and `outbox_events` are persisted and processed by the RabbitMQ worker                          |
| Redis usage                    | `Complete` | Redis is used for webhook rate limiting and conversation or ticket list caching                                 |
| Docker compose stack           | `Complete` | The stack includes backend, worker, MySQL, Redis, RabbitMQ, and frontend                                        |
| Testing                        | `Partial`  | Core tests exist, but the brief's full-coverage target is not met yet                                           |
| OpenAPI                        | `Complete` | Swagger now documents the full implemented application route surface                                            |

Additional note:

- The webhook intentionally requires `channel_key` so the multi-channel simulation is explicit. That is slightly stricter than the minimum payload in the brief, but it stays aligned with the channel-to-conversation flow.

## Prerequisites

- Go 1.26+
- Node.js 22+
- Podman 4.9+ with either `podman compose` or `podman-compose`

## Quick Start

```bash
git clone https://github.com/wecrazy/sociomile.git
cd sociomile
make env
make setup
make dev
```

Optional commands once the stack is up:

- `make dev-logs` to follow logs
- `make dev-down` to stop the stack
- `make migrate` and `make seed` if you want to rerun the database workflows manually from the host

Primary local URLs:

- Frontend: `http://localhost:5173`
- Backend health: `http://localhost:8080/health`
- Swagger UI: `http://localhost:8080/swagger`
- RabbitMQ management: `http://localhost:15672`

> Note: real local files `.env`, `.env.compose`, `backend/.env`, and `frontend/.env` are intended for local testing and are ignored by git.
>
> `make dev` starts the full stack in the background. The backend also applies migrations automatically during startup, and in `APP_ENV=development` it ensures the demo tenants, users, channels, and sample data without a separate seed workflow. After resetting the local MySQL volume, the demo accounts come back without needing routine `make seed` runs.

## Environment Summary

The local workflow uses two root env files:

- `.env` for shared values and local secrets
- `.env.compose` for internal container wiring in compose

Reviewer-facing summary of the most important variables:

| Variable            | Default                                                  | Purpose                                                                                                           |
| ------------------- | -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `APP_ENV`           | `development`                                            | Selects backend and worker runtime mode                                                                           |
| `BACKEND_PORT`      | `8080`                                                   | Host port for the API                                                                                             |
| `FRONTEND_PORT`     | `5173`                                                   | Host port for the UI                                                                                              |
| `MYSQL_DSN`         | `sociomile:sociomile@tcp(localhost:13306)/sociomile?...` | Host-side DSN for migrations, seeding, and direct backend runs                                                    |
| `REDIS_ADDR`        | `localhost:16379`                                        | Host-side Redis address for cache and rate limiting                                                               |
| `RABBITMQ_URL`      | `amqp://guest:guest@localhost:5672/`                     | Broker URL used by backend and worker                                                                             |
| `JWT_SECRET`        | `sociomile-local-dev-secret`                             | JWT signing secret                                                                                                |
| `ACCESS_TOKEN_TTL`  | `15m`                                                    | Access-token lifetime                                                                                             |
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1`                           | Browser API base URL                                                                                              |
| `VITE_APP_VERSION`  | auto-generated                                           | Optional frontend build version override; otherwise Vite uses the `package.json` version plus the build timestamp |
| `SWAGGER_FILE`      | `./docs/openapi.yaml`                                    | Static OpenAPI file served by the backend                                                                         |

The full environment reference lives in [docs/REFERENCE.en.md](docs/REFERENCE.en.md).

## Frontend Update Behavior

- Built JS and CSS assets already use Vite-hashed filenames.
- The frontend build also emits `version.json` and build metadata so the UI can detect newer deployments.
- When that manifest reports a newer version, operators get a toast that prompts a normal refresh to pick up the latest bundles without relying on a manual hard refresh.
- If you want explicit rollout versioning, set `VITE_APP_VERSION` during the frontend build or `COMPOSE_VITE_APP_VERSION` in the compose workflow.
- For production-style deploys, `/assets/*` can stay immutable, but `index.html` and `version.json` should use revalidation or `no-store` caching.
- In development mode, the update monitor is disabled and Vite HMR continues to handle local changes.

## Demo Accounts

The login page now shows quick-fill shortcuts for the Acme `admin` and `agent` roles. Every seeded demo user shares the password `Password123!`.

| Role    | Tenant         | Name          | Email                      |
| ------- | -------------- | ------------- | -------------------------- |
| `admin` | `Acme Support` | `Alice Admin` | `alice.admin@acme.local`   |
| `agent` | `Acme Support` | `Aaron Agent` | `aaron.agent@acme.local`   |
| `admin` | `Globex Care`  | `Grace Admin` | `grace.admin@globex.local` |
| `agent` | `Globex Care`  | `Gina Agent`  | `gina.agent@globex.local`  |

## API Summary

### Public Endpoints

| Method | Path                      | Purpose                           |
| ------ | ------------------------- | --------------------------------- |
| `GET`  | `/health`                 | Backend health probe              |
| `GET`  | `/swagger`                | Swagger UI                        |
| `POST` | `/api/v1/auth/login`      | Email and password login          |
| `POST` | `/api/v1/channel/webhook` | Simulated inbound channel message |

### Authenticated Endpoints

| Method  | Path                                 | Role             | Purpose                                       |
| ------- | ------------------------------------ | ---------------- | --------------------------------------------- |
| `GET`   | `/api/v1/auth/me`                    | `admin`, `agent` | Current user payload                          |
| `GET`   | `/api/v1/users/agents`               | `admin`, `agent` | Tenant-scoped active agent list               |
| `GET`   | `/api/v1/conversations`              | `admin`, `agent` | Conversation list with filters and pagination |
| `GET`   | `/api/v1/conversations/:id`          | `admin`, `agent` | Conversation detail and message thread        |
| `POST`  | `/api/v1/conversations/:id/messages` | `agent`          | Agent reply                                   |
| `PATCH` | `/api/v1/conversations/:id/assign`   | `admin`          | Assign a conversation to an agent             |
| `PATCH` | `/api/v1/conversations/:id/close`    | `admin`, `agent` | Close a conversation                          |
| `POST`  | `/api/v1/conversations/:id/escalate` | `agent`          | Escalate a conversation into a ticket         |
| `GET`   | `/api/v1/tickets`                    | `admin`, `agent` | Ticket list with filters and pagination       |
| `GET`   | `/api/v1/tickets/:id`                | `admin`, `agent` | Ticket detail                                 |
| `PATCH` | `/api/v1/tickets/:id/status`         | `admin`          | Update ticket status                          |

## Multi-Tenancy Model

- Every tenant-owned entity carries `tenant_id` in the database.
- Protected endpoints derive tenant context from the JWT claim instead of trusting caller-provided tenant IDs.
- Repository queries always scope by `tenant_id` to prevent cross-tenant reads and writes.
- The webhook endpoint is the intentional exception that accepts `tenant_id` because it simulates an external channel callback.
- Seed data includes two tenants so isolation can be verified quickly from the UI, API, and automated tests.

## Assumptions And Trade-Offs

- A monorepo keeps backend, frontend, worker, and infrastructure easy to review in one repository.
- The backend uses row-based multi-tenancy in a shared schema instead of separate databases per tenant.
- Swagger is served from a static OpenAPI file so the deliverable stays easy to inspect without extra generators.
- The frontend intentionally uses lightweight local state and request helpers so the take-home stays focused on product flow.
- The compose stack still uses the Vite development server to optimize local review speed, even though it is not a production-grade frontend serving setup.

## Additional Documentation

- [Architecture and flow](docs/ARCHITECTURE.en.md)
- [Operational reference](docs/REFERENCE.en.md)
- [Testing guide](docs/TESTING.en.md)
- [OpenAPI specification](backend/docs/openapi.yaml)

## Known Gaps

- The brief's full-coverage target is not met yet; the current coverage snapshot lives in [docs/TESTING.en.md](docs/TESTING.en.md)
- The latest verified clean snapshot is now backend `95.6%` statement coverage and frontend `97.88%` statement coverage with `86.79%` branch coverage and `85.07%` function coverage; the detailed breakdown lives in [docs/TESTING.en.md](docs/TESTING.en.md)
- Remaining backend gaps now mostly sit in seed-loading error branches, a subset of tenant-aware repository helpers, and a few service validation paths such as webhook transaction failures and ticket escalation edge cases
- The compose stack still uses the Vite development server rather than a production static web server
