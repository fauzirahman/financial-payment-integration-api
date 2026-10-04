# Financial Payment Integration API

A Go-based backend demonstration for financial domain modeling, payment orchestration, webhook processing, idempotency, and ledger-style accounting.

## Architecture

The project follows a layered design:

- HTTP handlers: transport layer
- Services: business logic and validation
- Repositories: PostgreSQL access layer
- Models: domain entities
- Migration files: schema evolution and persistence contracts

### Core flow

```mermaid
flowchart LR
    Client[Client / Merchant / Webhook] --> API[HTTP API]
    API --> Handler[Payment Handler]
    Handler --> Service[Payment Service]
    Service --> Repo[Postgres Repository]
    Repo --> Postgres[(PostgreSQL)]
```

## Domain model

The design models the key financial primitives needed for a real payment platform:

- customers
- accounts
- payments
- financial_transactions
- ledger_entries
- chart_of_accounts
- idempotency_keys
- payment_webhook_events

## ERD summary

```mermaid
erDiagram
    CUSTOMERS ||--o{ ACCOUNTS : owns
    CUSTOMERS ||--o{ PAYMENTS : creates
    ACCOUNTS ||--o{ PAYMENTS : funded_by
    PAYMENTS ||--o{ FINANCIAL_TRANSACTIONS : generates
    PAYMENTS ||--o{ LEDGER_ENTRIES : posts_to
    ACCOUNTS ||--o{ LEDGER_ENTRIES : records
    CHART_OF_ACCOUNTS ||--o{ LEDGER_ENTRIES : classifies
    PAYMENTS ||--o{ PAYMENT_WEBHOOK_EVENTS : emits

    CUSTOMERS {
        uuid id PK
        string customer_number
        string name
        string email
        string phone
        string status
        timestamp created_at
        timestamp updated_at
    }

    ACCOUNTS {
        uuid id PK
        uuid customer_id FK
        string account_number
        string currency
        numeric balance
        string status
    }

    PAYMENTS {
        uuid id PK
        uuid customer_id FK
        uuid account_id FK
        string reference
        numeric amount
        string currency
        string status
        string gateway
    }

    FINANCIAL_TRANSACTIONS {
        uuid id PK
        uuid payment_id FK
        uuid account_id FK
        string transaction_type
        string direction
        numeric amount
        string currency
        string reference
        string status
    }

    LEDGER_ENTRIES {
        uuid id PK
        uuid payment_id FK
        uuid account_id FK
        uuid chart_of_account_id FK
        string entry_type
        numeric amount
        string currency
        string description
    }

    CHART_OF_ACCOUNTS {
        uuid id PK
        string code
        string name
        string category
        string account_type
    }

    PAYMENT_WEBHOOK_EVENTS {
        uuid id PK
        uuid payment_id FK
        string event_id
        string event_type
        json payload
        string status
    }
```

## What the project demonstrates

- financial domain modeling
- payment lifecycle and status transitions
- atomic payment and idempotency-key persistence (duplicate keys are rejected; responses are not replayed)
- webhook HMAC validation and event-ID deduplication
- atomic webhook status changes and double-entry demo ledger posting
- in-memory retry queue and dead-letter simulation (not wired to the API server)
- PostgreSQL persistence with relational integrity

## Stack

- Go
- PostgreSQL
- pgx
- HTTP handlers
- SQL migrations

## Current limitations

- no real payment provider integration
- idempotency keys reject reuse but do not replay stored responses
- customer/account migrations (005/006) conflict with the active payment schema (001-004) and need reconciliation
- ledger uses fixed demo account codes and is not production accounting
