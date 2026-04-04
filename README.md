# TerraLedger

Agricultural Credit Intelligence platform on Solana for Kazakhstan.

## The Problem

Kazakhstan's agricultural sector produces 4.5% of GDP but receives only 2.3% of total bank lending — a structural credit gap of **1.2 trillion tenge (~$2.7B)**. Banks reject **67% of agricultural loan applications** because they cannot verify land collateral quality and cleanliness:

- **2,400+ double-pledging fraud cases per year** — the same land registered as collateral at multiple banks simultaneously
- **14–21 working days** to verify a single parcel through manual eGov lookups, notary requests, and akimat records
- **No standardized productivity data** — banks still rely on 1982 Soviet-era soil grades
- **18 million hectares** of arable land with fewer than 3% having any modern assessment

TerraLedger replaces this entire workflow with a single API call that returns a cryptographically verified credit profile in under 400ms: satellite-verified NDVI productivity certificates (TerraToken layer) + on-chain lien registry with atomic double-pledge prevention (LCR layer) + AI-computed credit risk score. No government portal, no phone call, no branch visit.

## Architecture

```
Frontend (React)  ──►  Backend (Go/Fiber)  ──►  Solana (Anchor v1)
                            │                       │
                        PostgreSQL              terra_token (NDVI certs)
                        Helius webhooks         lien_registry (liens)
                        Claude API (scoring)
```

**Two Anchor programs** share a PDA namespace (`cadastral_number` as seed):
- **terra_token** — parcel registration, NDVI certificate minting (Token-2022), seasonal checks
- **lien_registry** — encumbrance registration with double-pledge prevention, release with account closure

## Quick Start

```bash
# Prerequisites: Anchor 1.0, Solana CLI 3.x, Go 1.23+, Node 20+, Docker

# Start PostgreSQL
docker compose -f deployments/docker-compose.dev.yml up -d

# Build contracts
cd contracts && NO_DNA=1 anchor build

# Run contract tests (15 tests, ~200ms)
cd contracts && yarn install && yarn test

# Build backend
cd backend && go build ./...

# Run backend (needs .env with DATABASE_URL)
cd backend && go run ./cmd/server

# Build frontend
cd web && npm install && npm run build

# Dev frontend
cd web && npm run dev
```

## Project Structure

```
terra-ledger/
├── contracts/     # Anchor v1 programs (terra_token + lien_registry)
├── backend/       # Go/Fiber REST API (Clean Architecture)
├── web/           # React + Vite + @solana/kit frontend
├── sdk/           # @terraledger/sdk (TypeScript REST wrapper)
├── e2e/           # Playwright E2E tests
└── deployments/   # Docker Compose files
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/parcels/:cadastral/profile` | Full credit profile |
| POST | `/api/v1/parcels` | Register parcel |
| POST | `/api/v1/liens` | Register lien (double-pledge check) |
| POST | `/api/v1/liens/:id/release` | Release lien |
| POST | `/webhooks/helius` | Helius event indexer |

## Tech Stack

- **Contracts**: Anchor v1.0, Solana CLI 3.x, Token-2022
- **Backend**: Go, Fiber, PostgreSQL, zerolog, Claude API (scoring)
- **Frontend**: React 19, Vite, react-router-dom, @solana/kit, CSS Modules
- **Infra**: GitHub Actions, Docker, DigitalOcean

## Environment

Copy `.env.example` to `.env` and fill in your keys.
