# <img src="web/favicon.svg" width="32" height="32" alt="TerraLedger" /> TerraLedger

**Two-layer Solana-based Agricultural Credit Intelligence for Kazakhstan.**

Satellite-verified productivity certificates + on-chain lien registry + AI credit scoring = instant, trustworthy agricultural lending.

> **Hackathon:** National Solana Hackathon by Decentrathon (April 2026, Kazakhstan)

> **Case:** Case 1 (RWA Tokenization)

> **Live demo:** [terra-ledger.com](https://terra-ledger.com)

> **Devnet programs:** [terra_token](https://explorer.solana.com/address/2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX?cluster=devnet) | [lien_registry](https://explorer.solana.com/address/3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4?cluster=devnet)

---

## Screenshots

### Credit Profile Dashboard
NDVI chart, credit gauge, Agricultural Health Index radar, and lien history for a registered parcel.

![Dashboard](docs/screenshots/dashboard-credit-profile.png)

[//]: # (### AI Risk Assessment On-Chain &#40;Solana Explorer&#41;)

[//]: # (The keeper bot writes AI credit scores directly on-chain via `update_risk_assessment`. Each transaction is verifiable on Solana Explorer.)

[//]: # ()
[//]: # (![Solana Explorer]&#40;docs/screenshots/solana-explorer-risk-assessment.png&#41;)

### Double-Pledge Prevention (Transaction Error)
Attempting to register a second lien on an already-encumbered parcel fails atomically on-chain with `ActiveLienExists` error.

![Double Pledge Error](docs/screenshots/solana-explorer-double-pledge.png)

---

## The Problem

Kazakhstan's agricultural sector produces 4.5% of GDP but receives only 2.3% of total bank lending, a structural credit gap of **1.2 trillion tenge (~$2.7B)**. Banks reject **67% of agricultural loan applications** because they cannot verify land collateral quality and cleanliness:

- **2,400+ double-pledging fraud cases/year** — the same land registered as collateral at multiple banks simultaneously
- **14-21 working days** to verify a single parcel through manual eGov lookups, notary requests, and akimat records
- **No standardized productivity data** — banks still rely on 1982 Soviet-era soil grades
- **18 million hectares** of arable land with fewer than 3% having any modern assessment

## The Solution

TerraLedger replaces the entire manual workflow with a single API call that returns a cryptographically verified credit profile in under 400ms:

1. **TerraToken Layer** — Satellite-verified NDVI productivity certificates minted as non-transferable Token-2022 tokens
2. **Lien Registry Layer** — On-chain encumbrance CRUD with atomic double-pledge prevention via CPI
3. **AI Credit Scoring** — Claude API analyzes multi-index satellite data and outputs a 0-100 credit score written on-chain

---

## User Journey

### Farmer Flow

```
Connect Wallet → Register Parcel → Satellite NDVI Check → AI Credit Score → View Profile
```

1. **Connect Wallet** — Phantom wallet via Wallet Standard
2. **Register Parcel** — Enter cadastral number, area, land class. On-chain PDA created with EGISS snapshot hash
3. **Satellite Check** — Backend fetches real Sentinel-2 satellite data from Copernicus, computes NDVI/NDWI/EVI/LAI indices
4. **AI Credit Score** — Claude API analyzes satellite time series + land metadata, outputs score (0-100), grade (A-D), recommended LTV, and risk factors. Score is written on-chain via `update_risk_assessment` instruction
5. **View Profile** — Full credit profile with NDVI charts, radar chart, Agricultural Health Index, and lien history

### Lender Flow

```
Search Parcel → View Credit Profile → Register Lien → Monitor
```

1. **Search Parcel** — Find parcels by cadastral number
2. **View Credit Profile** — See AI score, satellite data, NDVI history, existing liens
3. **Register Lien** — On-chain encumbrance with double-pledge prevention (CPI to terra_token::verify_parcel). Atomic: if parcel already has a lien from another lender, transaction fails
4. **Monitor** — Keeper bot runs seasonal checks, updates scores, flags dormant parcels

---

## How Solana Is Used

This is not "blockchain for a checkbox." Every Solana feature is chosen for a specific reason:

| Feature | Why | Where |
|---------|-----|-------|
| **PDAs (Program Derived Addresses)** | Deterministic parcel accounts from cadastral numbers. Any party can derive the address and verify state without a centralized lookup | `seeds = ["parcel", cadastral_bytes]` |
| **CPI (Cross-Program Invocation)** | Lien registry calls terra_token::verify_parcel before registering encumbrance. Ensures parcel exists and is not fraud-flagged, atomically in one transaction | `lien_registry → terra_token` |
| **Token-2022 NonTransferable** | NDVI certificates cannot be transferred or sold. They are proof of satellite measurement, not a tradable asset. Soulbound by design | `NonTransferable` extension |
| **Token-2022 MetadataPointer** | Each NDVI certificate stores its measurement data (score, season, timestamp) as on-chain metadata without needing external storage | `MetadataPointer` extension |
| **TransferHook** | Secondary defense: if somehow a transfer is attempted, the hook checks `risk_flag` and `dormant_seasons` on the parcel and rejects if flagged or expired | `transfer_hook` program |
| **Helius Webhooks** | Real-time indexing of on-chain events into PostgreSQL. Backend stays in sync without polling or WebSocket subscriptions | Parcel registration, cert minting, lien events |
| **On-chain AI Score** | `ParcelConfig.ai_score` (u8, 0-100) lives on-chain. Any program can read it via CPI. Lenders don't need our API to check creditworthiness | `update_risk_assessment` instruction |

### On-Chain Programs (Devnet)

| Program | Address | Instructions |
|---------|---------|-------------|
| terra_token | `2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX` | register_parcel, mint_certificate, seasonal_check, verify_parcel, update_risk_assessment |
| lien_registry | `3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4` | register_encumbrance, release_encumbrance |
| transfer_hook | `CpvRLN1XUqpjPw1uHAQcPHQjgbvSF7jnaftwEghta964` | execute (called by Token-2022 runtime) |

---

## AI Credit Intelligence Pipeline

The AI component is not cosmetic. It drives on-chain state changes:

```
                        ┌─────────────────────────┐
                        │   Copernicus Sentinel-2  │
                        │   Statistical API        │
                        └────────┬────────────────┘
                                 │
                    12-month multi-index evalscript
                    (NDVI + NDWI + EVI + SCL cloud mask)
                                 │
                                 v
                        ┌─────────────────────────┐
                        │   Agricultural Health    │
                        │   Index (AHI) Composite  │
                        │   NDVI·0.4 + NDWI·0.2   │
                        │   + EVI·0.25 + LAI·0.15  │
                        └────────┬────────────────┘
                                 │
                                 v
                        ┌─────────────────────────┐
                        │   Claude API (Anthropic) │
                        │   Structured JSON output │
                        │   score, grade, LTV,     │
                        │   risk_factors[]         │
                        └────────┬────────────────┘
                                 │
                                 v
                        ┌─────────────────────────┐
                        │   Keeper Bot writes      │
                        │   ai_score on-chain via  │
                        │   update_risk_assessment │
                        │   (Solana transaction)   │
                        └─────────────────────────┘
```

**Decision chain:** Satellite data → AI analysis → on-chain transaction → smart contract state change

The keeper bot runs on a configurable interval, re-evaluates all parcels, and submits `update_risk_assessment` transactions to Solana. Each decision is logged as an `AgentDecision` record with the transaction signature, enabling full audit trail.

### Satellite Indices

| Index | What It Measures | Source |
|-------|-----------------|--------|
| **NDVI** | Vegetation health (chlorophyll density) | Sentinel-2 B8/B4 |
| **NDWI** | Water stress detection | Sentinel-2 B3/B8 |
| **EVI** | Enhanced vegetation (dense canopy correction) | Sentinel-2 B8/B4/B2 |
| **LAI** | Leaf area index (Beer-Lambert estimation) | Derived from NDVI |
| **SCL** | Scene Classification Layer (cloud mask) | Sentinel-2 L2A |

---

## Architecture

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────────────────┐
│   React/Vite     │     │   Go/Fiber API   │     │   Solana Devnet (Anchor v1)  │
│   Frontend       │────>│   Backend        │────>│                              │
│                  │     │                  │     │   terra_token                │
│   @solana/kit    │     │   PostgreSQL     │     │     register_parcel          │
│   Wallet Standard│     │   Helius webhooks│     │     mint_certificate         │
│   CSS Modules    │     │   Claude API     │     │     seasonal_check           │
│                  │     │   Copernicus API │     │     update_risk_assessment    │
│                  │     │   Keeper Bot     │     │                              │
│                  │     │                  │     │   lien_registry              │
│   Direct on-chain│─────│──────────────────│────>│     register_encumbrance     │
│   reads + signing│     │                  │     │     release_encumbrance      │
└──────────────────┘     └──────────────────┘     │                              │
                                                  │   transfer_hook              │
                                                  │     fraud/expiry guard       │
                                                  └──────────────────────────────┘
```

The frontend reads on-chain state directly via `@solana/kit` (not through the backend). The backend handles satellite data fetching, AI scoring, and on-chain write transactions via the keeper bot. Helius webhooks keep the PostgreSQL index in sync with on-chain events.

---

## Innovation

**What makes TerraLedger different from existing solutions:**

1. **Real satellite data, not oracles** — We use Copernicus Sentinel-2 Statistical API directly, not third-party oracles. The evalscript runs on ESA infrastructure with cloud masking and multi-band index computation. The data is independently reproducible.

2. **AI score lives on-chain** — The `ParcelConfig.ai_score` field is a u8 stored directly in the Solana account. Any program can CPI into terra_token and read the credit score without going through our API. This makes TerraLedger composable with other DeFi protocols.

3. **Atomic double-pledge prevention** — The lien registry uses CPI to verify parcel state before creating an encumbrance. If a parcel already has a lien from another lender, the transaction fails atomically. This is impossible to circumvent without program authority.

4. **Non-transferable productivity certificates** — Token-2022 NonTransferable extension ensures NDVI certificates are soulbound to the parcel owner. They prove satellite measurement happened, not that a score was "bought" or transferred.

5. **Full audit trail** — Every AI decision is logged with the Solana transaction signature. Regulators can verify exactly what score was assigned, when, and based on what data.

---

## Scalability

- **18M hectares** of Kazakhstan's arable land can be onboarded. Each parcel is a single PDA (~300 bytes). Solana handles this without sharding.
- **Seasonal checks** run via keeper bot every configurable interval. With Copernicus's free tier (10,000 requests/month), the system can monitor ~2,500 parcels with quarterly checks.
- **Composability** — Other Solana programs can read `ai_score` from `ParcelConfig` via CPI, enabling DeFi lending protocols to use TerraLedger as a credit oracle without any API dependency.
- **Multi-country expansion** — The cadastral number is a generic string seed. The system works for any country's land registry format.

---

## Quick Start

```bash
# Prerequisites: Anchor 1.0, Solana CLI 3.x, Go 1.23+, Node 20+, Docker

# Start PostgreSQL
docker compose -f deployments/docker-compose.dev.yml up -d

# Build and test contracts
cd contracts && NO_DNA=1 anchor build
cd contracts && yarn install && yarn test

# Run backend (needs .env, see Environment section)
cd backend && go build ./... && go run ./cmd/server

# Run backend tests
cd backend && go test ./... -count=1

# Run frontend
cd web && npm install && npm run dev

# E2E tests
cd e2e && npx playwright test
```

## Project Structure

```
terra-ledger/
├── contracts/              # 3 Anchor v1 programs
│   ├── programs/
│   │   ├── terra_token/    # Parcel registration, NDVI certs, seasonal checks
│   │   ├── lien_registry/  # Encumbrance CRUD with CPI to terra_token
│   │   └── transfer_hook/  # Token-2022 transfer guard (fraud + expiry)
│   ├── tests/              # TypeScript integration tests
│   └── scripts/            # seed.ts devnet seeder
├── backend/                # Go/Fiber REST API (Clean Architecture)
│   ├── cmd/server/         # Entry point
│   ├── internal/
│   │   ├── entity/         # Domain types (Parcel, Certificate, CreditScore, AgentDecision)
│   │   ├── usecase/        # Business logic (NDVI pipeline, repository interfaces)
│   │   ├── adapter/
│   │   │   ├── controller/ # HTTP handlers + Helius webhook
│   │   │   └── repository/ # PostgreSQL, Copernicus, Claude scorer, Solana RPC
│   │   └── infrastructure/ # App wiring, keeper bot, migrations (11 migrations)
│   └── Dockerfile
├── web/                    # React 19 + Vite + @solana/kit
│   ├── src/components/     # Card, Badge, MetricCard, NDVIChart, CreditGauge, RadarChart
│   ├── src/pages/          # WizardFlow (farmer+lender), ParcelDetail
│   ├── src/hooks/          # useCreditProfile, useSatelliteIndices, useMapParcels
│   └── src/solana/         # On-chain client (PDA derivation, instruction builders)
├── sdk/                    # @terraledger/sdk (TypeScript, @solana/kit)
├── e2e/                    # Playwright E2E tests
└── deployments/            # Docker Compose, nginx, deploy scripts
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/parcels` | Register parcel |
| GET | `/api/v1/parcels` | List all parcels |
| GET | `/api/v1/parcels/:cadastral` | Get parcel |
| GET | `/api/v1/parcels/:cadastral/profile` | Full credit profile (AI-scored) |
| GET | `/api/v1/parcels/:cadastral/ndvi` | Current NDVI from Sentinel-2 |
| GET | `/api/v1/parcels/:cadastral/satellite` | 12-month multi-index time series |
| GET | `/api/v1/parcels/:cadastral/indices` | Agricultural Health Index (AHI) |
| POST | `/api/v1/liens` | Register lien (double-pledge check) |
| POST | `/api/v1/liens/:id/release` | Release lien |
| GET | `/api/v1/parcels/:cadastral/liens` | List liens for parcel |
| GET | `/api/v1/parcels/:cadastral/certificates` | List NDVI certificates |
| POST | `/api/v1/consent/grant` | Grant PDPA consent |
| POST | `/api/v1/consent/revoke` | Revoke consent |
| POST | `/webhooks/helius` | Helius event indexer |
| GET | `/health` | Health check |

## TypeScript SDK

On-chain client built with `@solana/kit`. Direct Solana RPC interaction without the backend as middleman.

```typescript
import { getParcelPda, buildRegisterParcelInstruction } from '@terraledger/sdk'

const [parcelPda] = await getParcelPda('KZ11-0032-001')
const ix = await buildRegisterParcelInstruction(walletAddress, 'KZ11-0032-001', 150, 2, egissHash)
```

**Instruction builders:** `buildRegisterParcelInstruction`, `buildMintCertificateInstruction`, `buildVerifyParcelInstruction`, `buildSeasonalCheckInstruction`, `buildUpdateRiskAssessmentInstruction`, `buildRegisterEncumbranceInstruction`, `buildReleaseEncumbranceInstruction`

**PDA derivation:** `getParcelPda(cadastral)`, `getEncumbrancePda(parcelPda, lender)`, `getLienIndexPda(parcelPda)`

**Account deserialization:** `fetchParcelConfig()`, `fetchEncumbrance()`, `fetchLienIndex()`, `fetchAllParcels()`, `fetchAllEncumbrances()`

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Smart Contracts | Anchor v1.0, Rust, Token-2022 (NonTransferable + MetadataPointer), TransferHook |
| Backend | Go 1.23, Fiber, PostgreSQL, Clean Architecture, zerolog |
| AI | Claude API (Anthropic) for credit scoring |
| Satellite | Copernicus Sentinel-2 Statistical API, multi-band evalscript |
| Frontend | React 19, Vite, react-router-dom, @solana/kit, CSS Modules, Wallet Standard |
| SDK | TypeScript, @solana/kit (PDA derivation, instruction builders, account deserialization) |
| Infrastructure | GitHub Actions CI/CD, Docker, Yandex Container Registry, DigitalOcean, Helius webhooks, Cloudflare |

## Deployment

Backend + frontend deploy automatically on push to `master` via `.github/workflows/deploy.yml`:

1. Build Go backend Docker image -> push to Yandex Container Registry
2. Build Vite frontend -> upload artifact
3. SSH deploy to DigitalOcean VPS + nginx reload

Production: [terra-ledger.com](https://terra-ledger.com)

## Environment

Copy `.env.example` to `.env`:

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

Built at Decentrathon 5 National Solana Hackathon (Kazakhstan, April 2026).
