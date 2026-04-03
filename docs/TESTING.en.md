# Sociomile Testing Guide

[Bahasa Indonesia](TESTING.md) | English | [README](../README.en.md)

This document summarizes the automated test workflow, coverage, and the manual end-to-end validation path.

## Automated Checks

Run these from the repository root:

```bash
make backend-test
make frontend-test
make lint
make coverage
cd frontend && npm run build
```

When this document was last reviewed, the following commands completed successfully:

- `make backend-test`
- `make frontend-test`
- `make lint`
- `make coverage`
- `cd frontend && npm run build`

## Coverage Snapshot

- Backend: `95.6%` statement coverage from a clean `go test -count=1 -covermode=atomic -coverpkg=./... ./...` run
- Frontend: `97.88%` statement coverage, `86.79%` branch coverage, and `85.07%` function coverage from `make frontend-coverage`
- HTML reports: `coverage/backend.html` and `coverage/frontend/index.html`

## Current Test Coverage

### Backend

- Authentication success and failure
- Conversation lifecycle, ticket lifecycle, and tenant isolation in the service layer
- Fiber router integration for health, login, auth/me, users/agents, webhook intake, assignment, reply, close, escalation, ticket list/detail, and status updates
- Config loading, migration runner, `migrate` or `seed` CLI branches, API or worker startup seams, logger and Redis helpers, repository outbox helpers, worker retry cancellation or eventual success, and seeded data loading
- Redis cache helpers for JSON hit or miss, version invalidation, rate limiting, plus publisher setup, `Publish`, or `Close` seams and `OpenDatabase`

### Frontend

- Login success and failure
- App routing and protected-route redirects
- Dashboard metrics loading
- App layout locale switching and logout behavior
- Auth persistence on login, state clearing on logout, and invalid persisted-state fallback
- Locale switching, English bundle fallback, missing-key behavior, and theme persistence
- `useTenantAgents` hook success, no-token, and failure paths
- Conversation filter query updates plus pagination callbacks or offset resets
- Ticket filter query updates plus pagination callbacks or offset resets
- Admin assignment flow in conversation detail
- Agent reply flow, ticket escalation, and linked-ticket rendering in conversation detail
- Admin ticket status updates and role-based UI restriction for agents
- Theme and locale persistence in settings
- `DataTable` loading or empty states and pagination callbacks

## Manual End-to-End Test

1. Start the local stack, apply migrations, and load demo data.

```bash
make dev
make migrate
make seed
```

1. Open the local services.

- Frontend: `http://localhost:5173`
- Swagger UI: `http://localhost:8080/swagger`

1. Sign in as an admin.

- Email: `alice.admin@acme.local`
- Password: `Password123!`

1. Create a fresh conversation through the simulated webhook.

```bash
curl -X POST http://localhost:8080/api/v1/channel/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "11111111-1111-1111-1111-111111111111",
    "channel_key": "whatsapp",
    "customer_external_id": "cust-manual-001",
    "customer_name": "Manual QA",
    "message": "Hello, I need help"
  }'
```

1. Verify the admin flow in the UI.

- The new conversation appears in the conversation list
- The admin can assign the conversation to `Aaron Agent`

1. Sign in as an agent.

- Email: `aaron.agent@acme.local`
- Password: `Password123!`

1. Verify the agent flow in the UI.

- The agent can open the assigned conversation
- The agent can send a reply
- The agent can escalate the conversation into a ticket

1. Sign back in as an admin and verify the ticket flow.

- The new ticket appears in the ticket list
- The admin can move the ticket to `in_progress`, `resolved`, or `closed`

## Remaining Testing Gaps

- The brief's full-coverage target is not met yet
- Backend low-coverage areas now mostly cluster around seed-loading error branches, a subset of tenant-aware repository helpers, and a few service validation paths such as webhook transaction failures and ticket escalation or update edge cases
- Frontend low-coverage areas now mostly sit in edge branches inside the conversation or ticket detail pages, part of the list-page callback branches, and a small number of dashboard or layout UI branches
