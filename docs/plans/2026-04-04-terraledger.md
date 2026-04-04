# TerraLedger Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a two-layer Solana-based Agricultural Credit Intelligence platform combining satellite-verified productivity certificates (TerraToken) with on-chain lien registration (LCR), served through a Go backend and React frontend.

**Architecture:** Monorepo with four parallel workstreams — two Anchor v1 programs (terra_token + lien_registry) sharing a PDA namespace via CPI, a Go/Fiber backend (Clean Architecture), and a Vite/React frontend using @solana/kit + react-router-dom. PostgreSQL for off-chain persistence, Helius webhooks for event indexing.

**Tech Stack:**
- Smart Contracts: Anchor v1.0.0, Solana CLI 3.x, Rust 1.79+, Token-2022, `spl-token-interface 2.0`
- Backend: Go 1.23+, Fiber v2, PostgreSQL, zerolog, caarlos0/env, Anthropic Claude API (credit scoring)
- Frontend: React 19, Vite 8, react-router-dom 7, @solana/kit 6.x, @solana/react-hooks, CSS Modules
- Testing: LiteSVM 0.8.2 + anchor-litesvm 0.3, Surfpool, Playwright
- CI/CD: GitHub Actions, Yandex Container Registry, DigitalOcean VPS
- Indexing: Helius Enhanced Transactions webhooks + polling reconciler fallback

**Team:** 10 engineers across 4 workstreams. **Deadline:** April 7, 2026.

---

## Context

TerraLedger is a hackathon project for the Decentrathon (Case 1: Real World Assets). The concept scored 97/100 and describes an Agricultural Credit Intelligence system for Kazakhstan — combining satellite NDVI productivity certificates with on-chain lien tracking to prevent double-pledging of agricultural land. The full concept is at `judge-workspace/iteration-3/eval-11-terraledger/with_skill/outputs/concept.md`.

This plan was revised from an earlier draft based on user feedback:
1. Renamed ZherToken → **TerraToken** throughout
2. Upgraded from Anchor v0.31.1 → **Anchor v1.0**
3. Replaced tab-based SPA → **React Router**
4. Scaled from solo developer → **10-person team**
5. Replaced WebSocket indexer → **Helius webhooks**

---

## 1. Team Structure & Workstreams

| Workstream | People | Scope |
|------------|--------|-------|
| **Smart Contracts** (3) | SC Lead, SC-1, SC-2 | terra_token program, lien_registry program, TransferHook, CPI, LiteSVM/Surfpool tests, devnet deployment |
| **Backend** (3) | BE Lead, BE-1, BE-2 | Go/Fiber REST API, Helius webhook handler, NDVI pipeline, keeper bot, credit scoring, PostgreSQL |
| **Frontend** (2) | FE Lead, FE-1 | React app, router, lender dashboard, farmer portal, wallet integration, consent UI |
| **Infra/QA** (2) | Infra-1, Infra-2 | CI/CD, Docker, deployment, Playwright E2E, devnet seeding, demo prep |

---

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENTS                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐    │
│  │ Farmer Portal│  │Lender Dashbd │  │ @terraledger/sdk   │    │
│  │ (React SPA)  │  │ (React SPA)  │  │ (TS, thin REST)    │    │
│  └──────┬───────┘  └──────┬───────┘  └────────┬───────────┘    │
│         └────────┬─────────┴───────────────────┘                │
│                  │  @solana/kit + react-router-dom               │
└──────────────────┼──────────────────────────────────────────────┘
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                     GO BACKEND (Fiber)                           │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────┐ │
│  │  REST API   │  │  Webhook   │  │  NDVI      │  │  Keeper  │ │
│  │  Handlers   │  │  Handler   │  │  Pipeline  │  │  Bot     │ │
│  │  (Fiber)    │  │  (Helius)  │  │ (Sentinel) │  │  (cron)  │ │
│  └──────┬─────┘  └──────┬─────┘  └──────┬─────┘  └────┬─────┘ │
│         └──────┬─────────┴──────┬────────┴──────────────┘       │
│         USE CASES          RECONCILER (60s poll fallback)        │
│         PostgreSQL ◄───────────────────► Solana RPC (Helius)    │
└─────────────────────────────────┬───────────────────────────────┘
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SOLANA (Devnet via Helius)                   │
│  ┌───────────────────┐    CPI    ┌────────────────────────┐    │
│  │   terra_token      │◄────────►│   lien_registry         │    │
│  │   (Token-2022)     │          │                          │    │
│  │ - register_parcel  │          │ - register_encumbrance   │    │
│  │ - mint_certificate │          │ - release_encumbrance    │    │
│  │ - verify_parcel    │          │ - query_lien_status      │    │
│  │ - seasonal_check   │          │                          │    │
│  └───────────────────┘          └────────────────────────┘    │
│  Shared PDA: seeds = [b"parcel", cadastral_number.as_bytes()]   │
│                                                                 │
│  Helius Enhanced Transactions Webhook ──► POST /webhooks/helius │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Anchor v1.0 Toolchain & Key Patterns

**Toolchain:**
```
anchor-cli: 1.0.0
solana-cli: 3.1.10
rust: stable (1.79-1.85+)
platform-tools: v1.52
node: 20.x LTS
```

**Critical v1 differences from v0.31:**

| Area | v1 Pattern |
|------|-----------|
| CPI context | `CpiContext::new(terra_token::ID, accounts)` — Pubkey, not AccountInfo. Remove program account from `#[derive(Accounts)]` |
| Deps | `anchor-lang = "1.0.0"`, `anchor-spl = "1.0.0"`, all `solana-*` crates at `^3` |
| SPL Token | `spl-token-interface = "2.0"` (not `spl-token 7.x`) |
| TransferHook | `#[instruction(discriminator = spl_transfer_hook_interface::instruction::ExecuteInstruction::SPL_DISCRIMINATOR)]` |
| TS package | `@anchor-lang/core` (not `@coral-xyz/anchor`) |
| IDL | Stored via Program Metadata; publish with `anchor idl init` |
| Testing | `anchor test` defaults to surfpool; fallback: `[tooling] validator = "solana"` in Anchor.toml |
| Serialization | `borsh::to_vec(&value)` instead of `try_to_vec()` |
| Error blocks | One `#[error_code]` per program |
| Context | `Context<'info, T>` or `Context<T>` — no extra lifetimes |

**Edition2024 crate pins (must add to Cargo.lock):**
```
blake3 = 1.8.2
constant_time_eq = 0.3.1
base64ct = 1.7.3
indexmap = 2.11.4
```

---

## 4. Monorepo Structure

```
terraledger/
├── .github/workflows/
│   ├── contracts.yml              # anchor build + surfpool test
│   ├── backend.yml                # go lint + test
│   ├── frontend.yml               # vite build + vitest
│   └── e2e.yml                    # Playwright (devnet)
├── contracts/
│   ├── Anchor.toml
│   ├── Cargo.toml                 # workspace, resolver = "2"
│   ├── rust-toolchain.toml        # channel = "stable"
│   ├── programs/
│   │   ├── terra_token/
│   │   │   ├── Cargo.toml         # anchor-lang 1.0.0, anchor-spl 1.0.0
│   │   │   └── src/
│   │   │       ├── lib.rs
│   │   │       ├── constants.rs
│   │   │       ├── errors.rs
│   │   │       ├── events.rs
│   │   │       ├── state/
│   │   │       │   ├── mod.rs
│   │   │       │   └── parcel.rs          # ParcelConfig
│   │   │       └── instructions/
│   │   │           ├── mod.rs
│   │   │           ├── register_parcel.rs
│   │   │           ├── mint_certificate.rs
│   │   │           ├── verify_parcel.rs   # CPI entry for lien_registry
│   │   │           └── seasonal_check.rs
│   │   └── lien_registry/
│   │       ├── Cargo.toml
│   │       └── src/
│   │           ├── lib.rs
│   │           ├── constants.rs
│   │           ├── errors.rs
│   │           ├── events.rs
│   │           ├── state/
│   │           │   ├── mod.rs
│   │           │   ├── encumbrance.rs     # EncumbranceAccount
│   │           │   └── lien_index.rs      # LienIndex
│   │           └── instructions/
│   │               ├── mod.rs
│   │               ├── register_encumbrance.rs  # CPI: CpiContext::new(terra_token::ID, ...)
│   │               ├── release_encumbrance.rs
│   │               └── query_lien_status.rs
│   ├── tests/
│   │   ├── terra_token.test.ts
│   │   ├── lien_registry.test.ts
│   │   └── integration.test.ts
│   └── scripts/seed.ts
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── entity/                        # Domain models
│   │   │   ├── parcel.go
│   │   │   ├── certificate.go
│   │   │   ├── lien.go
│   │   │   ├── credit.go
│   │   │   ├── lender.go
│   │   │   └── errors.go
│   │   ├── usecase/
│   │   │   ├── repository/repository.go   # Interfaces
│   │   │   ├── parcel/usecase.go
│   │   │   ├── certificate/usecase.go
│   │   │   ├── lien/usecase.go
│   │   │   ├── credit/usecase.go
│   │   │   ├── ndvi/usecase.go
│   │   │   └── auth/usecase.go
│   │   ├── adapter/
│   │   │   ├── controller/http/
│   │   │   │   ├── router.go
│   │   │   │   ├── middleware.go          # API key auth
│   │   │   │   ├── parcel_handler.go
│   │   │   │   ├── lien_handler.go
│   │   │   │   ├── credit_handler.go
│   │   │   │   ├── certificate_handler.go
│   │   │   │   └── webhook_handler.go     # Helius webhook
│   │   │   └── repository/
│   │   │       ├── parcel_pg.go
│   │   │       ├── certificate_pg.go
│   │   │       ├── lien_pg.go
│   │   │       ├── lender_pg.go
│   │   │       ├── solana_rpc.go
│   │   │       ├── copernicus.go          # Sentinel-2 NDVI
│   │   │       ├── claude_scorer.go       # Anthropic API credit scoring
│   │   │       └── egiss_mock.go          # Mock EGISS oracle
│   │   └── infrastructure/
│   │       ├── app/app.go                 # DI wiring
│   │       ├── config/config.go           # caarlos0/env
│   │       ├── service/
│   │       │   ├── postgres.go
│   │       │   ├── keeper.go              # Seasonal check worker
│   │       │   └── reconciler.go          # Polling fallback (60s)
│   │       └── migration/migrations/      # SQL files
│   ├── go.mod
│   ├── Makefile
│   └── Dockerfile
├── web/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx                        # RouterProvider
│   │   ├── layouts/
│   │   │   ├── AppLayout.tsx              # SolanaProvider + TopBar + <Outlet />
│   │   │   └── AppLayout.module.css
│   │   ├── api/client.ts                  # fetch wrapper with API key
│   │   ├── solana/
│   │   │   ├── program.ts                 # PDA derivation, instruction builders (@solana/kit)
│   │   │   └── accounts.ts               # Account deserialization
│   │   ├── hooks/
│   │   │   ├── useParcel.ts
│   │   │   ├── useCreditProfile.ts
│   │   │   ├── useLien.ts
│   │   │   ├── useCertificates.ts
│   │   │   ├── useWalletAddress.ts
│   │   │   └── useConsent.ts
│   │   ├── components/                    # Card, Button, Input, Badge, MetricCard, Skeleton,
│   │   │                                  # NDVIChart, LienStatusBadge, CreditGauge, WalletButton
│   │   ├── pages/
│   │   │   ├── LenderDashboard/
│   │   │   ├── ParcelDetail/
│   │   │   ├── LienManagement/
│   │   │   ├── FarmerPortal/
│   │   │   └── ConsentDashboard/
│   │   └── styles/
│   │       ├── global.css
│   │       └── tokens.css
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile
├── sdk/                                   # @terraledger/sdk (thin REST wrapper)
│   ├── src/
│   │   ├── index.ts
│   │   ├── client.ts
│   │   └── types.ts
│   └── package.json
├── e2e/
│   ├── demo-flow.spec.ts
│   └── playwright.config.ts
├── deployments/
│   ├── docker-compose.yml
│   └── docker-compose.dev.yml
├── .env.example
└── Makefile
```

---

## 5. Smart Contract Design

### Account Structures

**ParcelConfig** (terra_token) — PDA: `[b"parcel", cadastral_str.as_bytes()]`

```rust
#[account]
#[derive(InitSpace)]
pub struct ParcelConfig {
    pub owner: Pubkey,
    #[max_len(20)]
    pub cadastral_str: String,         // "KZ19-0374-012"
    pub area_ha: u32,                  // hectares × 100
    pub land_class: u8,                // 1-8 per KZ Land Code
    pub kyc_verified: bool,            // mock: true on register
    pub kyc_method: u8,                // 0=None, 1=EGISS_NCA
    pub last_cert_epoch: u64,
    pub cert_count: u16,
    pub ndvi_submissions_this_season: u8,
    pub dormant_seasons: u8,
    pub has_active_lien: bool,
    pub egiss_snapshot_hash: [u8; 32],
    pub risk_flag: u8,                 // set by off-chain AI, checked by TransferHook
    pub registered_at: i64,
    pub bump: u8,
}
```

**EncumbranceAccount** (lien_registry) — PDA: `[b"encumbrance", parcel_pda, lender]`

```rust
#[account]
#[derive(InitSpace)]
pub struct EncumbranceAccount {
    pub parcel_pda: Pubkey,
    pub lender: Pubkey,
    pub amount: u64,
    pub notary_sig_hash: [u8; 32],
    pub notary_cert_hash: [u8; 32],
    pub egiss_snapshot_hash: [u8; 32],
    pub registered_at: i64,
    pub released_at: i64,              // 0 if active
    pub status: u8,                    // 0=Active, 1=Released, 2=Disputed
    pub bump: u8,
}
```

**LienIndex** (lien_registry) — PDA: `[b"lien_index", parcel_pda]`

```rust
#[account]
#[derive(InitSpace)]
pub struct LienIndex {
    pub parcel_pda: Pubkey,
    pub active_lien_count: u8,
    pub total_lien_count: u16,
    pub bump: u8,
}
```

### Instructions

| Program | Instruction | Key Behavior |
|---------|------------|-------------|
| terra_token | `register_parcel` | Init ParcelConfig PDA, mock KYC (kyc_verified=true), store EGISS fixture hash |
| terra_token | `mint_certificate` | Mint Token-2022 NonTransferable with NDVI metadata. TransferHook validates risk_flag |
| terra_token | `verify_parcel` | CPI entry point — returns kyc_verified, last_cert_epoch, cert_count |
| terra_token | `seasonal_check` | Keeper calls: check ndvi_submissions, increment dormant_seasons, emit ParcelDormant |
| lien_registry | `register_encumbrance` | CPI → terra_token::verify_parcel → check active_lien_count == 0 → store → emit |
| lien_registry | `release_encumbrance` | Set status=Released, decrement active_lien_count → emit |
| lien_registry | `query_lien_status` | Read-only: returns active count + total |

### CPI Pattern (Anchor v1)

```rust
// In lien_registry — v1 pattern: Pubkey, not AccountInfo
use terra_token::program::TerraToken;

pub fn register_encumbrance(ctx: Context<RegisterEncumbrance>, ...) -> Result<()> {
    let cpi_accounts = terra_token::cpi::accounts::VerifyParcel {
        parcel_config: ctx.accounts.parcel_config.to_account_info(),
    };
    // v1: pass program ID as Pubkey
    let cpi_ctx = CpiContext::new(TerraToken::id(), cpi_accounts);
    let verification = terra_token::cpi::verify_parcel(cpi_ctx, cadastral_number.clone())?;
    // ... double-pledge check, store encumbrance, emit event
}
```

### Events

```rust
// terra_token
ParcelRegistered { cadastral_number: String, owner: Pubkey, area_ha: u32 }
CertificateMinted { cadastral_number: String, season: String, ndvi_score: u16, cert_address: Pubkey }
ParcelDormant { cadastral_number: String, seasons_dormant: u8, has_active_lien: bool }

// lien_registry
EncumbranceRegistered { cadastral_number: String, lender: Pubkey, amount: u64, parcel_pda: Pubkey }
EncumbranceReleased { cadastral_number: String, lender: Pubkey, parcel_pda: Pubkey }
```

---

## 6. Backend Design

### REST API Routes

```go
// Authenticated (API key middleware)
GET  /api/v1/parcels/:cadastral           // Get parcel info
GET  /api/v1/parcels/:cadastral/profile   // Full credit profile
POST /api/v1/parcels                      // Register parcel
GET  /api/v1/parcels/:cadastral/certificates
POST /api/v1/parcels/:cadastral/certificates  // Mint NDVI cert
POST /api/v1/liens                        // Register lien
POST /api/v1/liens/:id/release            // Release lien
GET  /api/v1/parcels/:cadastral/liens

// Internal (webhook auth)
POST /webhooks/helius                     // Helius Enhanced Transactions

// Public
GET  /health
GET  /ready
```

### Helius Webhook Handler

```go
// POST /webhooks/helius
// 1. Verify Authorization header against HELIUS_WEBHOOK_SECRET
// 2. Parse []HeliusEnhancedTransaction payload
// 3. For each tx: check if ProgramID matches terra_token or lien_registry
// 4. Parse Anchor event from program logs (prefix "Program data:")
// 5. Route to use case: ParcelRegistered → INSERT parcels, EncumbranceRegistered → INSERT liens, etc.
// 6. Return 200 immediately
```

### Polling Reconciler (fallback)

```go
// Runs every 60s. Fetches getSignaturesForAddress for both programs.
// Compares against known tx signatures in PostgreSQL.
// Processes any missing transactions.
```

### Background Workers

| Worker | Trigger | Pattern |
|--------|---------|---------|
| Reconciler | Ticker (60s) | Single goroutine |
| NDVI Pipeline | On parcel registration + daily | Worker pool (3 goroutines max, channel-based) |
| Keeper Bot | Ticker (6h) | Single goroutine, builds + sends seasonal_check txns |
| Credit Scorer | On new cert/lien change | Claude API call (with debounce + PG cache, 1h TTL) |

### Credit Scoring — Claude API (Anthropic)

Credit scoring uses **Claude API** directly from the Go backend. No custom ML model, no Python service, no training pipeline. Claude analyzes parcel data as a structured prompt and returns a JSON credit assessment.

**Architecture:**
```
Go Backend (credit use case) ──HTTP POST──► Anthropic API (Claude)
                                             │
                                             ├── Structured prompt with parcel data
                                             ├── NDVI history, lien status, land class
                                             └── Returns: JSON { score, grade, ltv, confidence, explanation }
```

**Go integration:**
```go
// internal/adapter/repository/claude_scorer.go
type ClaudeScorer struct {
    apiKey string
    model  string  // "claude-haiku-4-5-20251001" (fast + cheap for scoring)
    client *http.Client
}

func (s *ClaudeScorer) ComputeScore(ctx context.Context, input *entity.ScoringInput) (*entity.CreditScore, error) {
    // Build structured prompt with parcel data
    // Call Anthropic Messages API
    // Parse JSON response into CreditScore
}
```

**Prompt design:**
```
You are an agricultural credit risk analyst for Kazakhstan. Analyze this parcel data and return a JSON credit assessment.

Parcel: {cadastral_number}, {area_ha} ha, land class {land_class}, oblast {oblast}
NDVI History (last 6 seasons): [{season: "2026-Q1", ndvi: 0.76, crop: "winter_wheat", yield: 2.9}, ...]
Lien History: {active_liens} active, {total_liens} historical, {disputes} disputes
Dormancy: {dormant_seasons} consecutive dormant seasons

Return ONLY this JSON:
{
  "score": <0-100>,
  "grade": <"A"|"B"|"C"|"D">,
  "recommended_ltv": <0.0-0.80>,
  "confidence": <0.0-1.0>,
  "risk_factors": ["<factor1>", ...],
  "explanation": "<2-3 sentence explanation>"
}
```

**Model choice:** `claude-haiku-4-5-20251001` — fast (~200ms), cheap ($0.25/M input), sufficient for structured scoring. Upgrade to `claude-sonnet-4-6` if richer explanations needed.

**Caching:** Cache scores in PostgreSQL `credit_scores` table with 1-hour TTL. Only re-score when new certificate is minted or lien status changes. At MVP volumes (< 100 queries/day), API cost is negligible (~$0.01/day).

**Fallback:** If Anthropic API is unreachable, return a hardcoded formula-based score (base 50 +/- adjustments for NDVI trend, cert count, land class). This ensures the demo never breaks.

### Config (caarlos0/env)

```env
APP_NAME=terraledger
PORT=3000
LOG_LEVEL=info
DATABASE_URL=postgres://terraledger:terraledger@localhost:5432/terraledger?sslmode=disable
SOLANA_RPC_URL=https://devnet.helius-rpc.com/?api-key=KEY
TERRA_TOKEN_PROGRAM_ID=<deployed>
LIEN_REGISTRY_PROGRAM_ID=<deployed>
RELAY_KEYPAIR_PATH=~/.config/solana/id.json
HELIUS_API_KEY=KEY
HELIUS_WEBHOOK_SECRET=SECRET
COPERNICUS_CLIENT_ID=ID
COPERNICUS_CLIENT_SECRET=SECRET
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_MODEL=claude-haiku-4-5-20251001
KEEPER_INTERVAL=6h
```

---

## 7. Frontend Design

### React Router Setup

```tsx
// src/App.tsx
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { lazy, Suspense } from 'react'

const ParcelDetail = lazy(() => import('./pages/ParcelDetail/ParcelDetail'))
const LienManagement = lazy(() => import('./pages/LienManagement/LienManagement'))
const FarmerPortal = lazy(() => import('./pages/FarmerPortal/FarmerPortal'))
const ConsentDashboard = lazy(() => import('./pages/ConsentDashboard/ConsentDashboard'))

const router = createBrowserRouter([
  {
    element: <AppLayout />,  // SolanaProvider + TopBar + <Outlet />
    children: [
      { index: true, element: <LenderDashboard /> },
      { path: 'parcel/:cadastral', element: <Suspense><ParcelDetail /></Suspense> },
      { path: 'liens', element: <Suspense><LienManagement /></Suspense> },
      { path: 'farmer', element: <Suspense><FarmerPortal /></Suspense> },
      { path: 'farmer/consent', element: <Suspense><ConsentDashboard /></Suspense> },
    ],
  },
])
```

### Routes

| Path | Page | Role |
|------|------|------|
| `/` | LenderDashboard | Search by cadastral, view credit profiles |
| `/parcel/:cadastral` | ParcelDetail | Full credit profile (shareable URL) |
| `/liens` | LienManagement | Register/release liens, active liens list |
| `/farmer` | FarmerPortal | Parcel registration, land passport, cert history |
| `/farmer/consent` | ConsentDashboard | PDPA consent tracking, access log |

### Solana Integration (same pattern as decentrathon/web)

```tsx
// AppLayout wraps SolanaProvider
import { SolanaProvider } from '@solana/react-hooks'
import { createClient, autoDiscover } from '@solana/client'

const solanaClient = createClient({
  endpoint: import.meta.env.VITE_SOLANA_RPC_URL,
  websocketEndpoint: wsEndpoint,
  walletConnectors: autoDiscover(),  // Wallet Standard, Phantom-first
})
```

- PDA derivation, instruction building, account deserialization: `src/solana/program.ts` and `src/solana/accounts.ts` using `@solana/kit` (Address, getProgramDerivedAddress, getAddressEncoder)
- State per page via custom hooks (`useCreditProfile`, `useLien`, etc.)
- Mobile-adaptive: CSS Modules with CSS custom properties, mobile-first grid

---

## 8. PostgreSQL Schema

```sql
-- parcels: cadastral_number (unique), owner_wallet, on_chain_address, area_ha, land_class, kyc_verified, oblast, rayon, holder_iin_hash, egiss_snapshot (JSONB)
-- certificates: parcel_id (FK), cadastral_number, season, ndvi_score, crop_type, yield_t_ha, sentinel_scene_id, on_chain_address, tx_signature
-- liens: parcel_id (FK), cadastral_number, lender_wallet, lender_name, amount_tenge, notary_cert_hash, on_chain_address, tx_signature, status (active/released/disputed)
-- lenders: name, bin, api_key (unique), tier, queries_this_month, query_limit
-- credit_scores: parcel_id (FK, unique), ai_score, recommended_ltv, collateral_grade, estimated_value_tenge, model_version
```

Key indexes: `idx_parcels_cadastral`, `idx_liens_cadastral_active` (partial WHERE status='active')

---

## 9. Testing Strategy

| Layer | Tool | Scope |
|-------|------|-------|
| Contract unit | LiteSVM (0.8.2) + anchor-litesvm (0.3) | Each instruction in isolation, PDA correctness, constraint violations |
| Contract integration | Surfpool (default for `anchor test`) | Full CPI flow, double-pledge prevention, Token-2022 TransferHook |
| Backend unit | Go table-driven tests + mock repos | Use case logic, handler response codes |
| Backend integration | Go test + testcontainers (PG) | API → handler → use case → real PostgreSQL |
| Frontend unit | vitest + testing-library | Hook tests with mock API, component rendering |
| E2E | Playwright (against devnet) | Demo flow only: query → lien → re-query → block duplicate |

CI note: Surfpool must be installed in GitHub Actions: `curl -sL https://run.surfpool.run/ | bash`

---

## 10. Delivery Schedule (3 Days, 10 People)

### Day 1 (April 5): Foundation + Core Build

**Morning:**
| Workstream | Tasks |
|-----------|-------|
| SC Lead | Monorepo scaffold, Anchor.toml, toolchain validation (Anchor 1.0, Solana CLI 3.x, surfpool), edition2024 crate pins |
| SC-1 | terra_token: ParcelConfig state + register_parcel + verify_parcel + LiteSVM tests |
| SC-2 | lien_registry: EncumbranceAccount + LienIndex state + register_encumbrance skeleton |
| BE Lead | Go skeleton, config (caarlos0/env), docker-compose (PG), migrations, `GET /health` |
| BE-1 | Entity models + repository interfaces + PostgreSQL implementations |
| BE-2 | Helius webhook handler skeleton + auth verification |
| FE Lead | Vite scaffold, react-router-dom, AppLayout, TopBar with NavLink, SolanaProvider, CSS tokens |
| FE-1 | Component library: Card, Button, Input, Badge, MetricCard, Skeleton, WalletButton |
| Infra-1 | GitHub Actions workflows (contracts, backend, frontend), surfpool install step |
| Infra-2 | Docker setup, docker-compose.dev.yml, devnet seeding script skeleton |

**Afternoon:**
| Workstream | Tasks |
|-----------|-------|
| SC-1 | mint_certificate with Token-2022 NonTransferable + MetadataPointer |
| SC-2 | register_encumbrance CPI to terra_token::verify_parcel (v1 pattern), release_encumbrance |
| SC Lead | TransferHook program (v1 discriminator pattern), LiteSVM unit tests |
| BE Lead | REST API handlers (parcel CRUD, credit profile, lien endpoints) + API key middleware |
| BE-1 | Solana RPC adapter (GetAccountInfo, SendTransaction, SimulateTransaction) |
| BE-2 | Webhook payload parsing + event routing to use cases |
| FE Lead | LenderDashboard: cadastral search, credit profile display with mock API data |
| FE-1 | NDVIChart, LienStatusBadge, CreditGauge components |
| Infra-1 | Finish CI pipelines |
| Infra-2 | Devnet seeding script (5 Akmola parcels) |

**Day 1 Checkpoint:**
- [x] `anchor build` succeeds for both programs with v1 deps
- [x] LiteSVM tests pass: register_parcel, verify_parcel, mint_certificate
- [x] Go backend starts, `GET /health` returns 200, migrations run
- [x] Frontend renders with router, TopBar navigation works, mock data displayed
- [x] CI pipeline runs

### Day 2 (April 6): Integration + Feature Complete

**Morning:**
| Workstream | Tasks |
|-----------|-------|
| SC-2 | Double-pledge prevention tests, edge cases |
| SC Lead | Integration tests (CPI: register → mint → encumber → release) |
| SC-1 | seasonal_check instruction + keeper-callable design |
| BE Lead | Wire handlers to real Solana RPC (read parcel PDA, submit lien tx) |
| BE-1 | Copernicus/NDVI pipeline (OAuth, scene search, NDVI calc) |
| BE-2 | Polling reconciler (60s fallback). Claude API scorer: prompt design, Anthropic HTTP client, PG caching |
| FE Lead | ParcelDetail page (useParams for :cadastral, fetch real API), wire search → navigate |
| FE-1 | LienManagement page: register lien form, release button, wallet signing |
| Infra-1 | Devnet deployment script |
| Infra-2 | Playwright E2E skeleton |

**Afternoon:**
| Workstream | Tasks |
|-----------|-------|
| SC Lead | Deploy both programs to devnet, `anchor idl init` (Program Metadata) |
| SC-1 + SC-2 | Fix integration test failures, seed test data on devnet |
| BE Lead | Keeper bot goroutine + seasonal_check tx building |
| BE-1 | Complete webhook → PostgreSQL pipeline, test with curl |
| BE-2 | TypeScript SDK (thin REST wrapper) |
| FE Lead | FarmerPortal page (registration, land passport, cert history) |
| FE-1 | ConsentDashboard page (consent status, access log) |
| Infra-1 | Deploy backend to DigitalOcean, nginx, SSL |
| Infra-2 | Register Helius webhook for devnet programs |

**Day 2 Checkpoint:**
- [x] Both programs deployed to devnet with 5 seeded parcels
- [x] CPI flow works on devnet
- [x] Backend serves real on-chain data via REST API
- [x] Helius webhook receives events → PostgreSQL updated
- [x] All 5 routes render with real data
- [x] Wallet connect + lien registration works E2E in browser

### Day 3 (April 7): Polish + Demo + Submit

**Morning:**
| Workstream | Tasks |
|-----------|-------|
| SC team | Bug fixes, README, architecture docs |
| BE team | Integration bugs, credit scoring for demo parcels, webhook reliability |
| FE team | UI polish, loading states, error handling, tx status indicators, Explorer links |
| Infra-2 | Playwright E2E test (demo flow), final deployment |
| Infra-1 | Smoke test all environments |

**Afternoon (all hands):**
- Seed demo data (5 parcels, 3 lenders)
- Run through 4:45 demo script
- Fix last issues
- Record backup video
- Submit

---

## 11. Interface Contracts (Define Day 1 Morning)

Teams must agree on these before working in parallel:

1. **Program IDs** — generated by `anchor keys list` after scaffold
2. **PDA seeds** — `[b"parcel", cadastral]`, `[b"encumbrance", parcel_pda, lender]`, `[b"lien_index", parcel_pda]`
3. **REST API contract** — endpoint signatures + JSON shapes (section 6)
4. **Event → PostgreSQL mapping** — ParcelRegistered → INSERT parcels, EncumbranceRegistered → INSERT liens, etc.
5. **Environment variables** — shared `.env.example` (section 6)

---

## 12. Risks

| Risk | Likelihood | Mitigation |
|------|-----------|-----------|
| Anchor v1 + surfpool unstable on macOS | Medium | `[tooling] validator = "solana"` fallback in Anchor.toml |
| Edition2024 crate breaks build | High | Pin blake3=1.8.2, constant_time_eq=0.3.1, base64ct=1.7.3, indexmap=2.11.4 |
| Light Protocol incompatible with Anchor v1 | High | **Deferred to phase 2** — use standard accounts for MVP |
| Helius devnet webhook delivery unreliable | Medium | Polling reconciler fills gaps (60s); worst case reconciler-only |
| Token-2022 TransferHook CU budget exceeded | Medium | Simplify hook to single risk_flag check |
| Copernicus API latency | Medium | Fall back to synthetic NDVI fixtures |
| Anthropic API latency/downtime | Low | Haiku is ~200ms; cache scores in PG (1h TTL); fallback: hardcoded Go formula |
| Team coordination / merge conflicts | Medium | Each workstream owns distinct dirs; only Anchor.toml + docker-compose shared |

---

## 13. Validation Checkpoints

| # | When | Validate |
|---|------|----------|
| CP1 | Day 1 morning | `anchor build` succeeds with v1 deps, surfpool starts, edition2024 pins work |
| CP2 | Day 1 end | CPI between programs works in LiteSVM. Double-pledge blocked. Events emitted. |
| CP3 | Day 2 morning | curl POST /webhooks/helius → PG updated → GET /profile returns new data |
| CP4 | Day 2 afternoon | Lender searches parcel in browser → navigates to `/parcel/:cadastral` → sees real on-chain data |
| CP5 | Day 3 morning | Playwright demo flow passes E2E against devnet |

---

## 14. Key Reference Files

| Reference | Path | Use |
|-----------|------|-----|
| TerraLedger concept (97/100) | `judge-workspace/iteration-3/eval-11-terraledger/with_skill/outputs/concept.md` | Domain knowledge, business rules |
| Anchor v1 migration guide | `.claude/skills/solana-dev/references/anchor/migrating-v0.32-to-v1.md` | CPI patterns, deps, TransferHook discriminator |
| Compatibility matrix | `.claude/skills/solana-dev/references/compatibility-matrix.md` | Toolchain versions, edition2024 pins |
| Anchor program reference | `.claude/skills/solana-dev/references/programs/anchor.md` | Account types, constraints, macros |
| Kit reference | `.claude/skills/solana-dev/references/kit/overview.md` | @solana/kit patterns |
| Frontend framework-kit | `.claude/skills/solana-dev/references/frontend-framework-kit.md` | SolanaProvider, wallet hooks |
| creditai Anchor program | `../decentrathon/contracts/creditai/programs/creditai/src/` | Instruction/state layout reference |
| creditai frontend | `../decentrathon/web/src/` | @solana/kit usage, hooks, solana/program.ts patterns |
| Storage service | `gitlab.com/shelterzoom/storage/internal/` | Clean Architecture layers reference |
| Go code standards | `../interview-knowladgebase/CLAUDE.md` | 85-line limit, naming, error handling, testing |
| Existing detailed plan | `docs/plans/2026-04-04-terraledger.md` | Full account structures, API contracts, SQL schema |

## 15. Verification

After implementation:
1. `cd contracts && NO_DNA=1 anchor build` — both programs compile
2. `cd contracts && NO_DNA=1 anchor test` — all LiteSVM + surfpool tests pass
3. `cd backend && make test` — all Go tests pass
4. `cd web && npm run build` — frontend builds without errors
5. `docker compose up` — backend + postgres start
6. `cd e2e && npx playwright test` — demo flow passes against devnet
7. Manual: open browser → search parcel → see profile → register lien → re-query → duplicate blocked
8. Verify Solana Explorer shows transactions for all on-chain operations
9. Verify credit score returned in profile response (Claude API scoring with explanation)
