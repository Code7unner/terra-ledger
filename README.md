# TerraLedger

Agricultural Credit Intelligence platform on Solana for Kazakhstan.

## The Problem

Kazakhstan's agricultural sector produces 4.5% of GDP but receives only 2.3% of total bank lending — a structural credit gap of **1.2 trillion tenge (~$2.7B)**. Banks reject **67% of agricultural loan applications** because they cannot verify land collateral quality and cleanliness:

- **2,400+ double-pledging fraud cases per year** — the same land registered as collateral at multiple banks simultaneously
- **14-21 working days** to verify a single parcel through manual eGov lookups, notary requests, and akimat records
- **No standardized productivity data** — banks still rely on 1982 Soviet-era soil grades
- **18 million hectares** of arable land with fewer than 3% having any modern assessment

TerraLedger replaces this entire workflow with a single API call that returns a cryptographically verified credit profile in under 400ms: satellite-verified NDVI productivity certificates (TerraToken layer) + on-chain lien registry with atomic double-pledge prevention (LCR layer) + AI-computed credit risk score.

## Architecture

```
Frontend (React/Vite)  ──>  Backend (Go/Fiber)  ──>  Solana Devnet (Anchor v1)
     |                            |                       |
     |                        PostgreSQL              terra_token
     |                        Helius webhooks           - parcel registration
     |                        Claude API (scoring)      - Token-2022 NDVI certs
     |                        Copernicus (satellite)    - seasonal checks
     |                                                lien_registry
     |                                                  - encumbrance CRUD
     |                                                  - CPI to terra_token
     |                                                transfer_hook
     |                                                  - fraud/expiry guard
     +-- @solana/kit (direct on-chain reads + signing)
```

**Three Anchor programs** share a PDA namespace (`cadastral_number` as seed):
- **terra_token** — parcel registration, NDVI certificate minting (Token-2022 NonTransferable + MetadataPointer), seasonal checks with dormancy tracking
- **lien_registry** — encumbrance registration with CPI to terra_token::verify_parcel, double-pledge prevention, release with account closure
- **transfer_hook** — guards Token-2022 transfers by checking fraud flags and certificate expiry

## Satellite Intelligence Pipeline

Real Sentinel-2 satellite data from Copernicus Data Space:

```
Copernicus Statistical API
    |
    v
Multi-index evalscript (NDVI + NDWI + EVI + SCL cloud mask)
    |
    v
12-month time series per parcel ──> PostgreSQL certificates
    |
    v
Credit scoring (Claude AI + multi-index enrichment)
    |
    v
Agricultural Health Index (AHI composite score)
```

- **NDVI** — vegetation health (Normalized Difference Vegetation Index)
- **NDWI** — water stress detection (Normalized Difference Water Index)
- **EVI** — enhanced vegetation index for dense canopy
- **LAI** — leaf area index (estimated from NDVI via Beer-Lambert)
- **SCL cloud masking** — excludes cloud, shadow, snow pixels for clean data

## Quick Start

```bash
# Prerequisites: Anchor 1.0, Solana CLI 3.x, Go 1.23+, Node 20+, Docker

# Start PostgreSQL
docker compose -f deployments/docker-compose.dev.yml up -d

# Build contracts
cd contracts && NO_DNA=1 anchor build

# Run contract tests (20 tests)
cd contracts && yarn install && yarn test

# Build and run backend (needs .env with DATABASE_URL)
cd backend && go build ./... && go run ./cmd/server

# Run backend tests
cd backend && go test ./... -count=1

# Build and run frontend
cd web && npm install && npm run dev
```

## Project Structure

```
terra-ledger/
├── contracts/           # Anchor v1 programs (terra_token + lien_registry + transfer_hook)
│   ├── programs/        # Rust source
│   ├── tests/           # TypeScript integration tests
│   └── scripts/         # seed.ts devnet seeder
├── backend/             # Go/Fiber REST API (Clean Architecture)
│   ├── cmd/server/      # Entry point
│   ├── internal/
│   │   ├── entity/          # Domain types
│   │   ├── usecase/         # Business logic (NDVI, repository interfaces)
│   │   ├── adapter/
│   │   │   ├── controller/  # HTTP handlers
│   │   │   └── repository/  # PG repos, Copernicus, Claude scorer, Solana RPC
│   │   └── infrastructure/  # App wiring, keeper bot, reconciler, migrations
│   └── Dockerfile
├── web/                 # React 19 + Vite + @solana/kit
│   ├── src/solana/      # On-chain client (PDA, instructions, accounts)
│   └── src/pages/       # WizardFlow, ParcelDetail, LienManagement, etc.
├── sdk/                 # @terraledger/sdk (on-chain @solana/kit client)
├── e2e/                 # Playwright E2E tests
└── deployments/         # Docker Compose, nginx, deploy scripts
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/parcels` | Register parcel |
| GET | `/api/v1/parcels/:cadastral` | Get parcel |
| GET | `/api/v1/parcels` | List all parcels |
| GET | `/api/v1/parcels/:cadastral/profile` | Full credit profile (AI-scored) |
| GET | `/api/v1/parcels/:cadastral/ndvi` | Current NDVI from Sentinel-2 |
| GET | `/api/v1/parcels/:cadastral/satellite` | 12-month multi-index time series |
| GET | `/api/v1/parcels/:cadastral/indices` | Agricultural Health Index (AHI) |
| POST | `/api/v1/liens` | Register lien (double-pledge check) |
| POST | `/api/v1/liens/:id/release` | Release lien |
| GET | `/api/v1/parcels/:cadastral/liens` | List liens |
| GET | `/api/v1/parcels/:cadastral/certificates` | List NDVI certificates |
| POST | `/api/v1/consent/grant` | Grant PDPA consent |
| POST | `/api/v1/consent/revoke` | Revoke consent |
| POST | `/webhooks/helius` | Helius event indexer |

## On-Chain Programs (Devnet)

| Program | Address |
|---------|---------|
| terra_token | `2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX` |
| lien_registry | `3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4` |

## Tech Stack

- **Contracts**: Anchor v1.0, Solana CLI 3.x, Token-2022 (NonTransferable + MetadataPointer), TransferHook
- **Backend**: Go 1.23, Fiber, PostgreSQL, zerolog, Claude API (credit scoring), Copernicus Sentinel Hub (satellite data)
- **Frontend**: React 19, Vite, react-router-dom, @solana/kit, CSS Modules, Wallet Standard
- **SDK**: @solana/kit (PDA derivation, account deserialization, instruction building)
- **Infra**: GitHub Actions CI/CD, Docker, Yandex Container Registry, DigitalOcean, Helius webhooks, Cloudflare

## Deployment

Backend + frontend deploy automatically on push to `master` via `.github/workflows/deploy.yml`:
1. Build Go backend Docker image -> push to Yandex Container Registry
2. Build Vite frontend -> upload artifact
3. SSH deploy to DigitalOcean VPS + nginx reload

Production: https://terra-ledger.com

## Environment

Copy `.env.example` to `.env` and fill in:

```
DATABASE_URL=postgres://terraledger:password@localhost:5432/terraledger?sslmode=disable
SOLANA_RPC_URL=https://devnet.helius-rpc.com/?api-key=YOUR_KEY
HELIUS_API_KEY=your_key
HELIUS_WEBHOOK_SECRET=your_secret
ANTHROPIC_API_KEY=your_key
COPERNICUS_CLIENT_ID=your_id
COPERNICUS_CLIENT_SECRET=your_secret
```

## Team

Built at Decentrathon 5 National Solana Hackathon (Kazakhstan, 2026).
