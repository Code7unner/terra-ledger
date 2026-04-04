# TerraLedger Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a two-layer Solana-based Agricultural Credit Intelligence platform combining satellite-verified productivity certificates (ZherToken) with on-chain lien registration (LCR), served through a Go backend and React frontend.

**Architecture:** Monorepo with three workstreams — two Anchor programs (zher_token + lien_registry) sharing a PDA namespace via CPI, a Go/Fiber backend following Clean Architecture (inspired by the storage-service and interview-knowledge-base references), and a Vite/React frontend using @solana/kit + @solana/react-hooks (following the decentrathon/web patterns). PostgreSQL for off-chain persistence, WebSocket-based Solana event indexing, Helius RPC.

**Tech Stack:**
- Smart Contracts: Anchor v0.31.1, Solana CLI 2.1.x, Rust, Token-2022, Light Protocol SDK
- Backend: Go 1.23+, Fiber v2, PostgreSQL, zerolog, caarlos0/env
- Frontend: React 19, Vite 8, @solana/kit 6.x, @solana/react-hooks, CSS Modules
- Testing: LiteSVM, solana-test-validator, Go table-driven tests, Playwright
- CI/CD: GitHub Actions, Yandex Container Registry, DigitalOcean VPS

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Architecture](#2-system-architecture)
3. [Smart Contract Planning](#3-smart-contract-planning)
4. [Backend Planning](#4-backend-planning)
5. [Frontend Planning](#5-frontend-planning)
6. [Data and Storage Planning](#6-data-and-storage-planning)
7. [Testing Strategy](#7-testing-strategy)
8. [Delivery Roadmap](#8-delivery-roadmap)
9. [Risks and Open Questions](#9-risks-and-open-questions)

---

## 1. Executive Summary

### What TerraLedger Is

TerraLedger is an Agricultural Credit Intelligence platform for Kazakhstan that combines:
- **ZherToken layer**: Satellite-verified (Sentinel-2 NDVI) productivity certificates as non-transferable Token-2022 tokens
- **LCR layer**: On-chain lien/encumbrance registry with notary EDS co-signature support
- **Unified SDK/API**: Single query returns complete credit profile (productivity + encumbrance + AI risk score) in <400ms

The two Anchor programs share a PDA namespace (cadastral number as seed) and compose atomically via CPI — lien registration verifies parcel existence in the same transaction.

### Core Assumptions

| # | Assumption | Impact if Wrong |
|---|-----------|----------------|
| A1 | Copernicus API credentials are valid and quota is sufficient for MVP (5-50 parcels) | Fall back to synthetic NDVI fixture data |
| A2 | Anchor v0.31.1 is stable on macOS for development + devnet deploy | Upgrade to v1.0 (migration guide available) |
| A3 | Light Protocol SDK works with Anchor v0.31.1 for ZK compressed accounts | Defer ZK compression to phase 2, use standard accounts |
| A4 | Solo developer can deliver MVP in ~7-10 days with focused execution | Cut scope: drop farmer portal, keep lender flow only |
| A5 | XGBoost model can run as a Go subprocess or embedded Python | Use simplified Go-native scoring formula |
| A6 | Helius supports devnet WebSocket subscriptions for event indexing | Fall back to polling-based sync |

### Key Technical Risks

1. **Light Protocol + Anchor v0.31.1 compatibility** — Light SDK may target v0.30 or v1.0. Validate in Task 2 before committing.
2. **Token-2022 TransferHook CU budget** — Deterministic checks in the hook consume compute units. Must profile early.
3. **Sentinel-2 data latency** — Copernicus API queries for specific parcels may have multi-hour delivery. Need async pipeline.
4. **XGBoost in Go** — No native Go XGBoost. Options: Python subprocess, ONNX runtime, or simplified Go-native model.
5. **Solo developer parallelization** — Smart contract work blocks frontend integration. Plan serial with contract-first.

### Recommended Implementation Order

```
Phase 1: Foundation (Tasks 1-6)
  ├── Monorepo scaffold + toolchain
  ├── Anchor programs (zher_token + lien_registry) with LiteSVM tests
  └── Database schema + Go skeleton

Phase 2: Backend Core (Tasks 7-13)
  ├── Clean architecture domains (parcel, certificate, lien, credit)
  ├── Fiber REST API
  ├── Solana event indexer (WebSocket)
  ├── Sentinel-2 NDVI pipeline
  └── Keeper bot (seasonal check worker)

Phase 3: Frontend + SDK (Tasks 14-19)
  ├── React app scaffold with @solana/kit
  ├── Lender dashboard
  ├── Farmer portal
  ├── TypeScript SDK (thin REST wrapper)
  └── PDPA consent UI

Phase 4: Integration + Polish (Tasks 20-23)
  ├── Devnet deployment
  ├── Playwright E2E (demo flow)
  ├── CI/CD pipeline
  └── Demo preparation
```

---

## 2. System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENTS                                  │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐    │
│  │ Farmer Portal│  │Lender Dashbd │  │ @terraledger/sdk   │    │
│  │ (React SPA)  │  │ (React SPA)  │  │ (TypeScript, REST) │    │
│  └──────┬───────┘  └──────┬───────┘  └────────┬───────────┘    │
│         │                  │                    │                │
│         │    ┌─────────────┴────────────────────┘               │
│         │    │  @solana/kit (wallet signing, tx building)       │
│         │    │  @solana/react-hooks (SolanaProvider)            │
└─────────┼────┼──────────────────────────────────────────────────┘
          │    │
          ▼    ▼
┌─────────────────────────────────────────────────────────────────┐
│                     GO BACKEND (Fiber)                           │
│                                                                 │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────┐ │
│  │  REST API   │  │  Solana    │  │  NDVI      │  │  Keeper  │ │
│  │  Handlers   │  │  Indexer   │  │  Pipeline  │  │  Bot     │ │
│  │  (Fiber)    │  │  (WS sub)  │  │ (Sentinel) │  │  (cron)  │ │
│  └──────┬─────┘  └──────┬─────┘  └──────┬─────┘  └────┬─────┘ │
│         │               │               │              │        │
│  ┌──────┴───────────────┴───────────────┴──────────────┴─────┐ │
│  │              USE CASES (Business Logic)                     │ │
│  │  parcel | certificate | lien | credit | ndvi | auth        │ │
│  └──────┬─────────────────────────────────────────────────────┘ │
│         │                                                       │
│  ┌──────┴─────────────────────────────────────────────────────┐ │
│  │              REPOSITORIES (Interfaces)                      │ │
│  │  ParcelRepo | CertRepo | LienRepo | SolanaClient           │ │
│  └──────┬─────────────────────────────────────────────────────┘ │
│         │                                                       │
│  ┌──────┴──────────┐  ┌──────────────┐  ┌───────────────────┐ │
│  │   PostgreSQL    │  │ Solana RPC   │  │ Copernicus API    │ │
│  │   (off-chain)   │  │ (Helius)     │  │ (Sentinel-2)      │ │
│  └─────────────────┘  └──────┬───────┘  └───────────────────┘ │
└──────────────────────────────┼──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SOLANA (Devnet)                              │
│                                                                 │
│  ┌───────────────────┐    CPI    ┌────────────────────────┐    │
│  │   zher_token       │◄────────►│   lien_registry         │    │
│  │   (Token-2022)     │          │   (ZK Compressed)       │    │
│  │                    │          │                          │    │
│  │ - register_parcel  │          │ - register_encumbrance   │    │
│  │ - mint_certificate │          │ - release_encumbrance    │    │
│  │ - verify_parcel    │          │ - query_lien_status      │    │
│  │ - seasonal_check   │          │ - init_merkle_tree       │    │
│  └───────────────────┘          └────────────────────────┘    │
│                                                                 │
│  Shared PDA: seeds = [b"parcel", cadastral_number.as_bytes()]   │
└─────────────────────────────────────────────────────────────────┘
```

### Main Subsystems and Responsibilities

| Subsystem | Responsibility | Owner Layer |
|-----------|---------------|-------------|
| zher_token program | Parcel registration, KYC mock, NDVI certificate minting (Token-2022 NonTransferable), TransferHook validation | Solana |
| lien_registry program | Encumbrance CRUD, CPI parcel verification, notary hash verification, ZK compressed state tree, double-pledge prevention | Solana |
| REST API | Lender queries, farmer registration, credit profile aggregation, API key auth | Go Backend |
| Solana Indexer | WebSocket log subscription, event parsing, off-chain state sync | Go Backend |
| NDVI Pipeline | Copernicus API integration, NDVI calculation, certificate data preparation | Go Backend |
| Keeper Bot | Seasonal dormancy checks, lien expiry monitoring, webhook dispatch | Go Backend |
| Farmer Portal | Parcel registration UI, land passport view, consent dashboard | React Frontend |
| Lender Dashboard | Credit profile query, lien management, portfolio overview | React Frontend |
| TypeScript SDK | Thin REST wrapper for lender integration, typed responses | NPM Package |

### Data Flow: Lender Credit Profile Query

```
1. Lender calls GET /api/v1/parcels/:cadastral/profile (API key header)
2. Backend checks PostgreSQL cache (TTL: 5 min)
3. If stale → parallel fetch:
   a. Solana RPC: fetch parcel PDA account data (zher_token)
   b. Solana RPC: fetch lien index PDA (lien_registry)
   c. PostgreSQL: fetch NDVI certificate history
   d. PostgreSQL: fetch AI credit score (precomputed)
4. Assemble CreditProfile response
5. Return JSON (< 400ms target)
```

### Transaction Flow: Lien Registration

```
1. Lender submits POST /api/v1/liens with cadastral_number, amount, notary_hash
2. Backend builds Solana transaction:
   a. lien_registry::register_encumbrance instruction
   b. Includes CPI to zher_token::verify_parcel_exists
   c. Includes notary_cert_hash in instruction data
3. Backend signs with relay keypair (gas subsidy)
4. Backend submits to Solana via Helius RPC
5. Wait for confirmation (< 1.2s)
6. WebSocket indexer catches EncumbranceRegistered event
7. Backend updates PostgreSQL lien cache
8. Return lien_address + tx_signature to lender
```

---

## 3. Smart Contract Planning

### Anchor Version Decision: v0.31.1

**Why v0.31.1 over v1.0:**
- The creditai reference project uses pre-v1 Anchor patterns — same instruction/state file layout we'll follow
- v0.31.1 has `declare_program!`, dynamic discriminators, `LazyAccount` — all needed features
- v1.0 changes TS package to `@anchor-lang/core`, changes IDL storage location, defaults to Surfpool — adds migration risk for zero benefit at MVP
- v0.31.1 is confirmed stable on macOS (no GLIBC issues)
- Solana CLI 2.1.x compatibility is confirmed

**Toolchain:**
```
anchor-cli: 0.31.1
solana-cli: 2.1.x
rust: 1.79+ (stable)
platform-tools: v1.47
```

### Program Structure

```
contracts/
├── Anchor.toml
├── Cargo.toml                    # workspace
├── rust-toolchain.toml           # channel = "1.79.0"
├── programs/
│   ├── zher_token/
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs            # declare_id!, #[program] mod
│   │       ├── constants.rs      # seeds, epochs, thresholds
│   │       ├── errors.rs         # TerraLedgerError enum
│   │       ├── events.rs         # ParcelRegistered, CertificateMinted, ParcelDormant
│   │       ├── state/
│   │       │   ├── mod.rs
│   │       │   ├── parcel.rs     # ParcelConfig account
│   │       │   └── certificate.rs # not needed — cert data lives in Token-2022 metadata
│   │       └── instructions/
│   │           ├── mod.rs
│   │           ├── register_parcel.rs
│   │           ├── mint_certificate.rs
│   │           ├── verify_parcel.rs      # CPI entry point for lien_registry
│   │           └── seasonal_check.rs     # keeper bot calls this
│   │
│   └── lien_registry/
│       ├── Cargo.toml
│       └── src/
│           ├── lib.rs
│           ├── constants.rs
│           ├── errors.rs
│           ├── events.rs         # EncumbranceRegistered, EncumbranceReleased
│           ├── state/
│           │   ├── mod.rs
│           │   ├── encumbrance.rs    # EncumbranceAccount
│           │   └── lien_index.rs     # LienIndex (per-parcel lien counter)
│           └── instructions/
│               ├── mod.rs
│               ├── register_encumbrance.rs   # CPI to zher_token::verify_parcel
│               ├── release_encumbrance.rs
│               ├── query_lien_status.rs      # read-only, returns lien count
│               └── init_merkle_tree.rs       # Light Protocol tree init
├── tests/
│   ├── zher_token.test.ts
│   ├── lien_registry.test.ts
│   └── integration.test.ts       # cross-program CPI flow
└── scripts/
    └── seed.ts                   # devnet seeding with test parcels
```

### Account Designs

#### ParcelConfig (zher_token)

```rust
#[account]
#[derive(InitSpace)]
pub struct ParcelConfig {
    pub owner: Pubkey,                    // farmer wallet
    pub cadastral_number: [u8; 32],       // padded cadastral string hash
    pub cadastral_raw: String,            // "KZ19-0374-012" (max 20 chars)
    #[max_len(20)]
    pub cadastral_str: String,
    pub area_ha: u32,                     // hectares × 100 (fixed point)
    pub land_class: u8,                   // 1-8 per KZ Land Code
    pub kyc_verified: bool,               // mock: always true after register
    pub kyc_method: u8,                   // 0=None, 1=EGISS_NCA
    pub last_cert_epoch: u64,             // slot of last certificate mint
    pub cert_count: u16,                  // total certificates issued
    pub ndvi_submissions_this_season: u8, // reset each season by keeper
    pub dormant_seasons: u8,             // incremented by keeper if no NDVI
    pub has_active_lien: bool,           // updated via CPI callback
    pub egiss_snapshot_hash: [u8; 32],   // hash of EGISS fixture data
    pub registered_at: i64,              // Clock unix timestamp
    pub bump: u8,
}
```

**PDA:** `seeds = [b"parcel", cadastral_str.as_bytes()], bump`
**Space:** 8 (discriminator) + ParcelConfig::INIT_SPACE

#### EncumbranceAccount (lien_registry)

```rust
#[account]
#[derive(InitSpace)]
pub struct EncumbranceAccount {
    pub parcel_pda: Pubkey,              // zher_token parcel PDA address
    pub lender: Pubkey,                   // lender wallet
    pub amount: u64,                      // loan amount in lamports (or tenge × 100)
    pub notary_sig_hash: [u8; 32],       // SHA256(notary EDS signature)
    pub notary_cert_hash: [u8; 32],      // notary certificate hash
    pub egiss_snapshot_hash: [u8; 32],   // eGov state at registration time
    pub registered_at: i64,              // timestamp
    pub released_at: i64,               // 0 if active
    pub status: u8,                      // 0=Active, 1=Released, 2=Disputed
    pub bump: u8,
}
```

**PDA:** `seeds = [b"encumbrance", parcel_pda.as_ref(), lender.as_ref()], bump`

#### LienIndex (lien_registry)

```rust
#[account]
#[derive(InitSpace)]
pub struct LienIndex {
    pub parcel_pda: Pubkey,
    pub active_lien_count: u8,           // 0 or 1 for MVP (no multi-lien)
    pub total_lien_count: u16,           // historical total
    pub bump: u8,
}
```

**PDA:** `seeds = [b"lien_index", parcel_pda.as_ref()], bump`

### Instructions Summary

#### zher_token

| Instruction | Signers | Key Accounts | Notes |
|------------|---------|-------------|-------|
| `register_parcel` | farmer (payer) | parcel_config (init) | Creates ParcelConfig PDA. Mock KYC: sets kyc_verified=true. Stores EGISS fixture hash. |
| `mint_certificate` | authority (backend relay) | parcel_config (mut), mint, token_account | Mints Token-2022 NonTransferable token with NDVI metadata. TransferHook validates risk flags. |
| `verify_parcel` | (CPI caller) | parcel_config (read) | Returns kyc_verified, last_cert_epoch, cert_count. Called by lien_registry via CPI. |
| `seasonal_check` | keeper (signer) | parcel_config (mut) | Checks ndvi_submissions_this_season. If 0, increments dormant_seasons, emits ParcelDormant. |

#### lien_registry

| Instruction | Signers | Key Accounts | Notes |
|------------|---------|-------------|-------|
| `register_encumbrance` | lender (payer) | encumbrance (init), lien_index (mut), zher_token program (CPI) | Atomic: CPI verify_parcel → check active_lien_count == 0 → store encumbrance → emit event |
| `release_encumbrance` | lender (signer) | encumbrance (mut), lien_index (mut) | Sets status=Released, released_at=now, decrements active_lien_count |
| `query_lien_status` | none | lien_index (read) | View instruction, returns active count + total |

### Events

```rust
// zher_token events
#[event]
pub struct ParcelRegistered {
    pub cadastral_number: String,
    pub owner: Pubkey,
    pub area_ha: u32,
}

#[event]
pub struct CertificateMinted {
    pub cadastral_number: String,
    pub season: String,       // "2026-Q1"
    pub ndvi_score: u16,      // × 1000 (0.760 → 760)
    pub cert_address: Pubkey,
}

#[event]
pub struct ParcelDormant {
    pub cadastral_number: String,
    pub seasons_dormant: u8,
    pub has_active_lien: bool,
}

// lien_registry events
#[event]
pub struct EncumbranceRegistered {
    pub cadastral_number: String,
    pub lender: Pubkey,
    pub amount: u64,
    pub parcel_pda: Pubkey,
}

#[event]
pub struct EncumbranceReleased {
    pub cadastral_number: String,
    pub lender: Pubkey,
    pub parcel_pda: Pubkey,
}
```

### Token-2022 TransferHook Strategy

The TransferHook program enforces deterministic on-chain checks only:

```rust
// transfer_hook/src/lib.rs
// This is a separate small program that Token-2022 invokes
pub fn transfer_hook(ctx: Context<TransferHookCtx>) -> Result<()> {
    let cert_metadata = &ctx.accounts.cert_metadata;

    // Check 1: Certificate not flagged as fraudulent
    require!(!cert_metadata.fraud_flagged, TerraLedgerError::FraudFlagged);

    // Check 2: Certificate not expired (older than 4 seasons)
    let current_slot = Clock::get()?.slot;
    let slots_per_season = 216_000 * 90; // ~90 days
    require!(
        current_slot - cert_metadata.minted_slot < slots_per_season * 4,
        TerraLedgerError::CertificateExpired
    );

    // Check 3: Risk flag not set by off-chain AI pipeline
    require!(
        cert_metadata.risk_flag == 0,
        TerraLedgerError::RiskFlagSet
    );

    Ok(())
}
```

AI anomaly detection (NDVI jump > 0.25, impossible yield, coordinate mismatch) runs off-chain in the Go backend. If anomaly detected, backend calls an `update_risk_flag` instruction on the certificate metadata before any mint/burn/reissue.

### ZK Compression (Light Protocol)

For the LCR layer, lien state is stored in Light Protocol compressed accounts:

```rust
// In lien_registry — uses light-sdk Anchor macros
use light_sdk::compressed_account::*;

// Compressed lien record (stored in merkle tree, not as regular account)
#[derive(LightAccount)]
pub struct CompressedLienRecord {
    pub parcel_hash: [u8; 32],    // SHA256(cadastral_number)
    pub lender: Pubkey,
    pub amount: u64,
    pub status: u8,
    pub registered_at: i64,
}
```

**Fallback plan:** If Light SDK doesn't integrate cleanly with Anchor v0.31.1 during Task 2 validation, we use standard (non-compressed) accounts for MVP. The account structure is identical; only the storage mechanism changes. The interface stays the same.

### @solana/kit Usage (Client/Backend)

Frontend and TypeScript SDK use `@solana/kit` exclusively (no `@solana/web3.js`):

```typescript
// Client-side: creating the Solana client (same pattern as creditai/web)
import { createClient, autoDiscover } from '@solana/client'
import { address, getProgramDerivedAddress, getAddressEncoder } from '@solana/kit'

const solanaClient = createClient({
  endpoint: import.meta.env.VITE_SOLANA_RPC_URL,
  websocketEndpoint: wsEndpoint,
  walletConnectors: autoDiscover(),
})

// PDA derivation (same pattern as creditai reference)
const addressEncoder = getAddressEncoder()
export async function getParcelPda(cadastralNumber: string): Promise<[Address, number]> {
  return getProgramDerivedAddress({
    programAddress: ZHER_TOKEN_PROGRAM_ID,
    seeds: ['parcel', new TextEncoder().encode(cadastralNumber)],
  })
}
```

Go backend uses raw Solana JSON-RPC calls (no Go SDK dependency):
```go
// internal/adapter/repository/solana_rpc.go
type SolanaRPC struct {
    endpoint string
    client   *http.Client
}

func (s *SolanaRPC) GetAccountInfo(ctx context.Context, address string) (*AccountInfo, error) {
    // Direct JSON-RPC call to Helius
}

func (s *SolanaRPC) SendTransaction(ctx context.Context, tx []byte) (string, error) {
    // Submit signed transaction
}

func (s *SolanaRPC) SubscribeLogs(ctx context.Context, programID string) (<-chan LogEvent, error) {
    // WebSocket subscription for event indexing
}
```

---

## 4. Backend Planning

### Clean Architecture Module Breakdown

Following the patterns from both the storage-service and interview-knowledge-base references:

```
backend/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point, config load, app.Start()
├── internal/
│   ├── entity/                        # Domain models (no external deps)
│   │   ├── parcel.go                  # Parcel, ParcelConfig
│   │   ├── certificate.go             # NDVICertificate
│   │   ├── lien.go                    # Encumbrance, LienStatus
│   │   ├── credit.go                  # CreditProfile, CreditScore
│   │   ├── lender.go                  # Lender, APIKey
│   │   └── errors.go                  # ErrNotFound, ErrAlreadyExists, ErrDoublePledge
│   │
│   ├── usecase/                       # Business logic
│   │   ├── repository/               # Repository interfaces (defined by use cases)
│   │   │   └── repository.go         # ParcelRepo, CertRepo, LienRepo, SolanaClient, NDVIProvider
│   │   ├── parcel/                   # Parcel registration, lookup
│   │   │   └── usecase.go
│   │   ├── certificate/              # NDVI certificate management
│   │   │   └── usecase.go
│   │   ├── lien/                     # Encumbrance CRUD, double-pledge check
│   │   │   └── usecase.go
│   │   ├── credit/                   # Credit profile aggregation, AI scoring
│   │   │   └── usecase.go
│   │   ├── ndvi/                     # Sentinel-2 data pipeline
│   │   │   └── usecase.go
│   │   └── auth/                     # API key validation
│   │       └── usecase.go
│   │
│   ├── adapter/
│   │   ├── controller/
│   │   │   └── http/                 # Fiber handlers
│   │   │       ├── parcel_handler.go
│   │   │       ├── lien_handler.go
│   │   │       ├── credit_handler.go
│   │   │       ├── certificate_handler.go
│   │   │       ├── middleware.go      # API key auth, logging, recovery
│   │   │       └── router.go         # Route registration
│   │   └── repository/
│   │       ├── parcel_pg.go          # PostgreSQL ParcelRepo impl
│   │       ├── certificate_pg.go
│   │       ├── lien_pg.go
│   │       ├── lender_pg.go
│   │       ├── solana_rpc.go         # Solana JSON-RPC client
│   │       ├── copernicus.go         # Sentinel-2 NDVI API client
│   │       └── egiss_mock.go         # Mock EGISS oracle (fixture-based)
│   │
│   └── infrastructure/
│       ├── app/
│       │   └── app.go                # DI wiring: repos → use cases → handlers → server
│       ├── config/
│       │   └── config.go             # caarlos0/env-based config
│       ├── service/
│       │   ├── postgres.go           # Connection pool, migrations
│       │   ├── solana_ws.go          # WebSocket event subscription
│       │   └── keeper.go             # Keeper bot goroutine (seasonal checks)
│       └── migration/
│           └── migrations/           # SQL migration files
│               ├── 001_create_parcels.up.sql
│               ├── 001_create_parcels.down.sql
│               ├── 002_create_certificates.up.sql
│               ├── 003_create_liens.up.sql
│               ├── 004_create_lenders.up.sql
│               └── 005_create_credit_scores.up.sql
│
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── .env.example
```

### Dependency Flow

```
infrastructure → adapter → usecase → entity
     ↓              ↓         ↓
  (config,      (postgres,  (business    (pure domain
   postgres,     solana,     logic,       models,
   ws, keeper)   http)       interfaces)  errors)
```

### Repository Interfaces

```go
// internal/usecase/repository/repository.go
package repository

type ParcelRepo interface {
    Create(ctx context.Context, p *entity.Parcel) error
    GetByCadastral(ctx context.Context, cadastral string) (*entity.Parcel, error)
    UpdateOnChainState(ctx context.Context, cadastral string, state entity.OnChainState) error
}

type CertificateRepo interface {
    Create(ctx context.Context, cert *entity.NDVICertificate) error
    ListByParcel(ctx context.Context, cadastral string) ([]entity.NDVICertificate, error)
    GetLatest(ctx context.Context, cadastral string) (*entity.NDVICertificate, error)
}

type LienRepo interface {
    Create(ctx context.Context, lien *entity.Encumbrance) error
    GetActive(ctx context.Context, cadastral string) (*entity.Encumbrance, error)
    ListByParcel(ctx context.Context, cadastral string) ([]entity.Encumbrance, error)
    UpdateStatus(ctx context.Context, id string, status entity.LienStatus) error
}

type LenderRepo interface {
    GetByAPIKey(ctx context.Context, key string) (*entity.Lender, error)
}

type SolanaClient interface {
    GetAccountInfo(ctx context.Context, address string) ([]byte, error)
    SendTransaction(ctx context.Context, txBytes []byte) (string, error)
    SimulateTransaction(ctx context.Context, txBytes []byte) (*SimResult, error)
    SubscribeLogs(ctx context.Context, programID string, ch chan<- LogEvent) error
}

type NDVIProvider interface {
    FetchNDVI(ctx context.Context, bbox BoundingBox, dateRange DateRange) (*NDVIResult, error)
}

type EGISSOracle interface {
    GetParcelSnapshot(ctx context.Context, cadastral string) (*entity.EGISSSnapshot, error)
}
```

### Fiber Route Structure

```go
// internal/adapter/controller/http/router.go
func RegisterRoutes(app *fiber.App, h *Handlers) {
    api := app.Group("/api/v1")

    // Auth middleware
    api.Use(h.APIKeyAuth)

    // Parcels
    api.Post("/parcels", h.RegisterParcel)                     // farmer registers parcel
    api.Get("/parcels/:cadastral", h.GetParcel)                // get parcel info
    api.Get("/parcels/:cadastral/profile", h.GetCreditProfile) // full credit profile

    // Certificates
    api.Post("/parcels/:cadastral/certificates", h.MintCertificate)  // mint NDVI cert
    api.Get("/parcels/:cadastral/certificates", h.ListCertificates)

    // Liens
    api.Post("/liens", h.RegisterLien)             // register encumbrance
    api.Post("/liens/:id/release", h.ReleaseLien)  // release encumbrance
    api.Get("/parcels/:cadastral/liens", h.ListLiens)

    // Health
    app.Get("/health", h.Health)
    app.Get("/ready", h.Ready)
}
```

### API Contract Outline

#### GET /api/v1/parcels/:cadastral/profile

**Response (200):**
```json
{
  "cadastral_number": "KZ19-0374-012",
  "query_timestamp": 1775197375000,
  "parcel": {
    "area_ha": 45.3,
    "land_class": 2,
    "kyc_verified": true,
    "holder_iin_masked": "***4521",
    "oblast": "Akmola",
    "on_chain_address": "Bx7z...nYf2"
  },
  "productivity": {
    "certificates": [...],
    "ndvi_trend": "improving",
    "ndvi_trend_slope": 0.008,
    "dormancy_risk": "low"
  },
  "encumbrances": {
    "active_liens": [],
    "lien_count_historical": 1,
    "double_pledge_risk": false
  },
  "credit_intelligence": {
    "ai_credit_score": 82,
    "recommended_ltv": 0.65,
    "collateral_quality_grade": "A",
    "estimated_land_value_tenge": 23000000
  }
}
```

#### POST /api/v1/liens

**Request:**
```json
{
  "cadastral_number": "KZ19-0374-012",
  "lender_wallet": "HkBN...FKH",
  "amount_tenge": 15000000,
  "notary_cert_hash": "a1b2c3..."
}
```

**Response (201):**
```json
{
  "lien_id": "uuid",
  "on_chain_address": "Lien7x...",
  "tx_signature": "5KtR...",
  "status": "active"
}
```

**Error (409 — double pledge):**
```json
{
  "error": "ACTIVE_LIEN_EXISTS",
  "message": "Parcel KZ19-0374-012 has an active lien from Halyk Bank",
  "existing_lien": { "lender": "Halyk Bank", "registered_at": "2026-03-15" }
}
```

### Background Jobs / Async Processing

| Worker | Trigger | Concurrency Pattern | Description |
|--------|---------|-------------------|-------------|
| Solana Indexer | Continuous WebSocket | Single goroutine + channel fan-out | Subscribes to both program logs, parses events, updates PostgreSQL |
| NDVI Pipeline | On parcel registration + periodic (daily) | Worker pool (3 goroutines max) | Fetches Sentinel-2 data from Copernicus, calculates NDVI, prepares cert data |
| Keeper Bot | Ticker (every 6 hours) | Single goroutine | Iterates parcels needing seasonal check, builds + sends transactions |
| Credit Scorer | On new certificate or lien change | Single goroutine with debounce | Recomputes AI credit score, updates PostgreSQL |

**Concurrency patterns applied (from go-concurrency-patterns skill):**

```go
// Worker pool for NDVI pipeline
func (p *NDVIPipeline) Start(ctx context.Context) {
    jobs := make(chan string, 1) // cadastral numbers to process
    var wg sync.WaitGroup

    for range 3 { // 3 workers
        wg.Add(1)
        go func() {
            defer wg.Done()
            for cadastral := range jobs {
                if err := p.processNDVI(ctx, cadastral); err != nil {
                    p.logger.Error().Err(err).Str("cadastral", cadastral).Msg("ndvi processing failed")
                }
            }
        }()
    }

    // ... feed jobs channel from registration events
    // cleanup:
    close(jobs)
    wg.Wait()
}

// Keeper bot with ticker
func (k *KeeperBot) Run(ctx context.Context) {
    ticker := time.NewTicker(6 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := k.runSeasonalChecks(ctx); err != nil {
                k.logger.Error().Err(err).Msg("seasonal check failed")
            }
        }
    }
}
```

### Configuration Strategy

```go
// internal/infrastructure/config/config.go
type Config struct {
    AppName  string `env:"APP_NAME" envDefault:"terraledger"`
    Port     int    `env:"PORT" envDefault:"3000"`
    LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

    // PostgreSQL
    DatabaseURL string `env:"DATABASE_URL,required"`

    // Solana
    SolanaRPCURL       string `env:"SOLANA_RPC_URL" envDefault:"https://devnet.helius-rpc.com/?api-key="`
    SolanaWSURL        string `env:"SOLANA_WS_URL"`
    ZherTokenProgramID string `env:"ZHER_TOKEN_PROGRAM_ID,required"`
    LienRegistryID     string `env:"LIEN_REGISTRY_PROGRAM_ID,required"`
    RelayKeypairPath   string `env:"RELAY_KEYPAIR_PATH" envDefault:"~/.config/solana/id.json"`
    HeliusAPIKey       string `env:"HELIUS_API_KEY,required"`

    // Copernicus (Sentinel-2)
    CopernicusClientID     string `env:"COPERNICUS_CLIENT_ID,required"`
    CopernicusClientSecret string `env:"COPERNICUS_CLIENT_SECRET,required"`

    // Keeper
    KeeperInterval string `env:"KEEPER_INTERVAL" envDefault:"6h"`
}
```

### Error Handling Strategy

Following the interview-knowledge-base CLAUDE.md patterns:

```go
// internal/entity/errors.go
var (
    ErrNotFound     = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrDoublePledge = errors.New("active lien exists on parcel")
    ErrNotVerified  = errors.New("parcel not KYC verified")
    ErrUnauthorized = errors.New("invalid or missing API key")
)

// Use cases wrap with context:
// return fmt.Errorf("fetching parcel %s: %w", cadastral, err)

// Handlers map to HTTP status:
// entity.ErrNotFound → 404
// entity.ErrAlreadyExists → 409
// entity.ErrDoublePledge → 409
// entity.ErrUnauthorized → 401
```

### Observability / Logging

```go
// zerolog with structured context
logger.Info().
    Str("cadastral", cadastral).
    Str("lender", lenderID).
    Dur("latency", time.Since(start)).
    Msg("credit profile served")

logger.Error().
    Err(err).
    Str("tx_signature", sig).
    Msg("solana transaction failed")
```

Fiber middleware logs every request with method, path, status, latency.

---

## 5. Frontend Planning

### App Structure

```
web/
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json
├── eslint.config.js
├── public/
│   └── favicon.svg
├── src/
│   ├── main.tsx                      # React root
│   ├── App.tsx                       # SolanaProvider + Router
│   ├── App.module.css
│   │
│   ├── api/
│   │   └── client.ts                # fetch wrapper with API key header
│   │
│   ├── solana/
│   │   ├── program.ts               # PDA derivation, instruction builders (Kit)
│   │   └── accounts.ts              # Account deserialization helpers
│   │
│   ├── hooks/
│   │   ├── useParcel.ts             # register, fetch parcel
│   │   ├── useCreditProfile.ts      # fetch full credit profile
│   │   ├── useLien.ts               # register, release lien
│   │   ├── useCertificates.ts       # list certificates
│   │   ├── useWalletAddress.ts      # from @solana/react-hooks
│   │   └── useConsent.ts            # PDPA consent tracking
│   │
│   ├── components/
│   │   ├── TopBar/
│   │   │   ├── TopBar.tsx
│   │   │   └── TopBar.module.css
│   │   ├── WalletButton/
│   │   │   ├── WalletButton.tsx
│   │   │   └── WalletButton.module.css
│   │   ├── Card/
│   │   ├── Button/
│   │   ├── Input/
│   │   ├── Badge/
│   │   ├── MetricCard/
│   │   ├── Skeleton/
│   │   ├── NDVIChart/                # NDVI trend sparkline
│   │   │   ├── NDVIChart.tsx
│   │   │   └── NDVIChart.module.css
│   │   ├── LienStatusBadge/
│   │   ├── CreditGauge/              # A/B/C/D grade visualization
│   │   └── ParcelMap/                # cadastral number display (not a real map)
│   │
│   ├── pages/
│   │   ├── LenderDashboard/
│   │   │   ├── LenderDashboard.tsx
│   │   │   └── LenderDashboard.module.css
│   │   ├── FarmerPortal/
│   │   │   ├── FarmerPortal.tsx
│   │   │   └── FarmerPortal.module.css
│   │   ├── ParcelDetail/
│   │   │   ├── ParcelDetail.tsx
│   │   │   └── ParcelDetail.module.css
│   │   ├── LienManagement/
│   │   │   ├── LienManagement.tsx
│   │   │   └── LienManagement.module.css
│   │   └── ConsentDashboard/
│   │       ├── ConsentDashboard.tsx
│   │       └── ConsentDashboard.module.css
│   │
│   └── styles/
│       ├── global.css                # CSS reset, CSS variables, typography
│       └── tokens.css                # Design tokens (colors, spacing, breakpoints)
```

### Route / Page Map

| Route | Page | Role | Description |
|-------|------|------|-------------|
| `/` | LenderDashboard | Lender | Search by cadastral number, view credit profiles |
| `/parcel/:cadastral` | ParcelDetail | Both | Full credit profile view |
| `/liens` | LienManagement | Lender | Register/release liens, view active liens |
| `/farmer` | FarmerPortal | Farmer | Register parcel, view land passport, NDVI history |
| `/farmer/consent` | ConsentDashboard | Farmer | PDPA consent tracking, access log |

Tab-based navigation (same pattern as creditai/web reference — no react-router needed for MVP):

```tsx
function App() {
  const [activeTab, setActiveTab] = useState('lender')
  return (
    <SolanaProvider client={solanaClient}>
      <div className={styles.app}>
        <TopBar activeTab={activeTab} onTabChange={setActiveTab} />
        <main className={styles.content}>
          {activeTab === 'lender' && <LenderDashboard />}
          {activeTab === 'parcel' && <ParcelDetail />}
          {activeTab === 'liens' && <LienManagement />}
          {activeTab === 'farmer' && <FarmerPortal />}
          {activeTab === 'consent' && <ConsentDashboard />}
        </main>
      </div>
    </SolanaProvider>
  )
}
```

### State Management Approach

No external state library. Each page owns its state via custom hooks (same pattern as creditai/web):

```tsx
// hooks/useCreditProfile.ts
export function useCreditProfile() {
  const [data, setData] = useState<CreditProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchProfile = useCallback(async (cadastral: string) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await get<CreditProfile>(`/api/v1/parcels/${cadastral}/profile`)
      setData(resp)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch profile')
    } finally {
      setLoading(false)
    }
  }, [])

  return { data, loading, error, fetchProfile }
}
```

### Wallet and Solana Interaction Strategy

```tsx
// App.tsx — same pattern as creditai/web
import { SolanaProvider } from '@solana/react-hooks'
import { createClient, autoDiscover } from '@solana/client'

const endpoint = import.meta.env.VITE_SOLANA_RPC_URL || 'https://devnet.helius-rpc.com/?api-key=...'
const wsEndpoint = endpoint.replace('https://', 'wss://').replace('http://', 'ws://')

const solanaClient = createClient({
  endpoint,
  websocketEndpoint: wsEndpoint,
  walletConnectors: autoDiscover(), // Wallet Standard — Phantom first
})
```

For transaction signing (lien registration from lender wallet):
```tsx
// hooks/useLien.ts
import { useSignAndSendTransaction } from '@solana/react-hooks'
import { buildRegisterEncumbranceInstruction } from '../solana/program'

export function useLien() {
  const signAndSend = useSignAndSendTransaction()

  const registerLien = useCallback(async (params: RegisterLienParams) => {
    // Build instruction using @solana/kit
    const ix = await buildRegisterEncumbranceInstruction(params)
    // Sign and send via wallet
    const sig = await signAndSend([ix])
    // Notify backend
    await post('/api/v1/liens', { ...params, tx_signature: sig })
  }, [signAndSend])

  return { registerLien }
}
```

### Mobile-Adaptive UI Strategy

CSS Modules with CSS custom properties for breakpoints:

```css
/* styles/tokens.css */
:root {
  --breakpoint-sm: 640px;
  --breakpoint-md: 768px;
  --breakpoint-lg: 1024px;

  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;

  --color-primary: #2563eb;
  --color-success: #16a34a;
  --color-danger: #dc2626;
  --color-warning: #d97706;
  --color-surface: #ffffff;
  --color-background: #f8fafc;
  --color-text: #1e293b;
  --color-text-secondary: #64748b;

  --radius-sm: 6px;
  --radius-md: 8px;
  --radius-lg: 12px;
}

/* Mobile-first responsive pattern */
/* styles/global.css */
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: 'Inter', -apple-system, sans-serif; }
```

Component-level responsive:
```css
/* pages/LenderDashboard/LenderDashboard.module.css */
.grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--spacing-md);
}

@media (min-width: 768px) {
  .grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (min-width: 1024px) {
  .grid {
    grid-template-columns: 1fr 1fr 1fr;
  }
}
```

### UX Considerations for Blockchain Flows

1. **Transaction states**: Every on-chain action shows: Preparing → Signing (wallet popup) → Confirming → Done/Error
2. **Confirmation indicator**: Show Solana Explorer link after tx confirmation
3. **Double-pledge block**: When lien registration fails due to active lien, show clear error with existing lien details
4. **Loading skeletons**: Use Skeleton component (from creditai/web) during RPC calls
5. **Offline indicator**: If Solana RPC unreachable, show banner, degrade to cached data

---

## 6. Data and Storage Planning

### On-Chain vs Off-Chain Data Split

| Data | Storage | Reason |
|------|---------|--------|
| ParcelConfig (KYC status, cadastral, land class) | On-chain (Solana) | Source of truth, CPI-verifiable |
| NDVI Certificate (Token-2022 + metadata) | On-chain (Solana) | Immutable attestation, soulbound |
| Encumbrance (lien status, notary hash) | On-chain (Solana, ZK compressed) | Double-pledge prevention requires on-chain atomicity |
| LienIndex (active count per parcel) | On-chain (Solana) | Real-time CPI check |
| NDVI raw data (Sentinel-2 scenes, pixel values) | Off-chain (PostgreSQL) | Too large for on-chain; on-chain stores hash |
| Credit score (AI-computed) | Off-chain (PostgreSQL) | Recomputed frequently, read-heavy |
| Lender profiles + API keys | Off-chain (PostgreSQL) | Auth concern, no blockchain need |
| EGISS snapshots (fixture data) | Off-chain (PostgreSQL) | Mock oracle, stored at registration time |
| NDVI trend analytics | Off-chain (PostgreSQL) | Derived from certificate history |
| Webhook/notification state | Off-chain (PostgreSQL) | Operational concern |

### PostgreSQL Schema

```sql
-- 001_create_parcels.up.sql
CREATE TABLE parcels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cadastral_number VARCHAR(20) UNIQUE NOT NULL,
    owner_wallet VARCHAR(44) NOT NULL,
    on_chain_address VARCHAR(44),
    area_ha NUMERIC(10,2) NOT NULL,
    land_class SMALLINT NOT NULL CHECK (land_class BETWEEN 1 AND 8),
    kyc_verified BOOLEAN DEFAULT FALSE,
    oblast VARCHAR(50),
    rayon VARCHAR(50),
    holder_name VARCHAR(100),
    holder_iin_hash VARCHAR(64),          -- SHA256(IIN + cadastral)
    egiss_snapshot JSONB,                  -- fixture data
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_parcels_cadastral ON parcels(cadastral_number);
CREATE INDEX idx_parcels_wallet ON parcels(owner_wallet);

-- 002_create_certificates.up.sql
CREATE TABLE certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id),
    cadastral_number VARCHAR(20) NOT NULL,
    season VARCHAR(10) NOT NULL,           -- "2026-Q1"
    ndvi_score NUMERIC(5,3) NOT NULL,      -- 0.000 - 1.000
    crop_type VARCHAR(50),
    yield_t_ha NUMERIC(6,2),
    sentinel_scene_id VARCHAR(100),
    on_chain_address VARCHAR(44),
    tx_signature VARCHAR(88),
    minted_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_certs_parcel ON certificates(parcel_id);
CREATE INDEX idx_certs_cadastral_season ON certificates(cadastral_number, season);

-- 003_create_liens.up.sql
CREATE TABLE liens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id),
    cadastral_number VARCHAR(20) NOT NULL,
    lender_wallet VARCHAR(44) NOT NULL,
    lender_name VARCHAR(100),
    amount_tenge BIGINT NOT NULL,
    notary_cert_hash VARCHAR(64),
    on_chain_address VARCHAR(44),
    tx_signature VARCHAR(88),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'released', 'disputed')),
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    released_at TIMESTAMPTZ
);

CREATE INDEX idx_liens_parcel ON liens(parcel_id);
CREATE INDEX idx_liens_cadastral_active ON liens(cadastral_number) WHERE status = 'active';

-- 004_create_lenders.up.sql
CREATE TABLE lenders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    bin VARCHAR(12),                       -- Kazakhstan BIN
    api_key VARCHAR(64) UNIQUE NOT NULL,
    tier VARCHAR(20) DEFAULT 'starter'
        CHECK (tier IN ('starter', 'standard', 'enterprise', 'api_only')),
    queries_this_month INT DEFAULT 0,
    query_limit INT DEFAULT 200,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 005_create_credit_scores.up.sql
CREATE TABLE credit_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parcel_id UUID NOT NULL REFERENCES parcels(id) UNIQUE,
    cadastral_number VARCHAR(20) UNIQUE NOT NULL,
    ai_score SMALLINT CHECK (ai_score BETWEEN 0 AND 100),
    recommended_ltv NUMERIC(4,3),
    collateral_grade CHAR(1) CHECK (collateral_grade IN ('A', 'B', 'C', 'D')),
    estimated_value_tenge BIGINT,
    model_version VARCHAR(20),
    computed_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Indexing / Caching / Query Strategy

1. **Primary query path**: PostgreSQL first (cached off-chain data), Solana RPC only for fresh on-chain verification
2. **Cache invalidation**: WebSocket indexer updates PostgreSQL when events arrive — cache is always near-real-time
3. **Credit profile assembly**: Single SQL query joins parcels + certificates + liens + credit_scores — < 5ms
4. **On-chain verification**: For high-stakes queries (lien registration), always verify on-chain state via RPC before proceeding
5. **No Redis**: PostgreSQL is sufficient for MVP query volume. Add Redis only if latency becomes an issue.

---

## 7. Testing Strategy

### Unit Tests

**Smart Contracts (LiteSVM):**
- Each instruction tested in isolation
- PDA derivation correctness
- Constraint violations (double pledge, unverified parcel)
- Event emission verification

**Go Backend:**
- Table-driven tests for all use cases
- Repository tests with test PostgreSQL (testcontainers or embedded)
- Mock Solana client for use case tests

**Frontend:**
- Hook tests with mock API responses
- Component rendering tests (vitest + testing-library)

### Integration Tests

**Smart Contracts (solana-test-validator):**
- Full CPI flow: register_parcel → mint_certificate → register_encumbrance → release
- Double-pledge prevention across transactions
- Keeper bot seasonal check flow
- Token-2022 TransferHook execution

**Go Backend:**
- API integration tests: HTTP → handler → use case → mock repo
- Solana indexer: mock WebSocket messages → PostgreSQL state verification

### Contract Tests (Anchor)

```typescript
// tests/integration.test.ts
describe('TerraLedger Integration', () => {
  it('blocks double pledge via CPI', async () => {
    // 1. Register parcel
    await registerParcel('KZ11-0032-001', farmer)
    // 2. First lien succeeds
    await registerEncumbrance('KZ11-0032-001', lender1, 15_000_000)
    // 3. Second lien FAILS with ActiveLienExists
    await expect(
      registerEncumbrance('KZ11-0032-001', lender2, 10_000_000)
    ).to.be.rejectedWith('ActiveLienExists')
  })

  it('allows lien after release', async () => {
    await registerParcel('KZ11-0032-002', farmer)
    await registerEncumbrance('KZ11-0032-002', lender1, 15_000_000)
    await releaseEncumbrance('KZ11-0032-002', lender1)
    // Now lender2 can register
    await registerEncumbrance('KZ11-0032-002', lender2, 10_000_000)
  })
})
```

### Playwright E2E (Demo Flow Only)

```typescript
// e2e/demo-flow.spec.ts
import { test, expect } from '@playwright/test'

test('demo: query → lien → re-query → block duplicate', async ({ page }) => {
  // Setup: use test wallet, parcels pre-seeded on devnet
  await page.goto('/')

  // Step 1: Lender queries parcel — clean profile
  await page.fill('[data-testid="cadastral-input"]', 'KZ11-0032-001')
  await page.click('[data-testid="search-btn"]')
  await expect(page.locator('[data-testid="active-liens"]')).toHaveText('0')
  await expect(page.locator('[data-testid="double-pledge-risk"]')).toHaveText('No')

  // Step 2: Lender registers lien
  await page.click('[data-testid="register-lien-btn"]')
  await page.fill('[data-testid="lien-amount"]', '15000000')
  await page.click('[data-testid="confirm-lien-btn"]')
  // Wait for tx confirmation
  await expect(page.locator('[data-testid="lien-status"]')).toHaveText('Active', { timeout: 15000 })

  // Step 3: Re-query — active lien shown
  await page.click('[data-testid="search-btn"]')
  await expect(page.locator('[data-testid="active-liens"]')).toHaveText('1')

  // Step 4: Second lien attempt — blocked
  await page.click('[data-testid="register-lien-btn"]')
  await page.fill('[data-testid="lien-amount"]', '10000000')
  await page.click('[data-testid="confirm-lien-btn"]')
  await expect(page.locator('[data-testid="error-message"]')).toContainText('Active lien exists')
})
```

### Test Environment Strategy

| Layer | Local Dev | CI | Devnet |
|-------|-----------|------|--------|
| Contract unit | LiteSVM | LiteSVM | - |
| Contract integration | solana-test-validator | solana-test-validator | - |
| Backend unit | Go test + mock repos | Go test + mock repos | - |
| Backend integration | Go test + testcontainers (PG) | Go test + PG service | - |
| Frontend unit | vitest | vitest | - |
| E2E (Playwright) | - | - | Devnet (pre-seeded) |

### Mocking vs Real Validator

- **LiteSVM**: All contract unit tests (fast, in-process, no network)
- **solana-test-validator**: CPI integration tests, Token-2022 + TransferHook tests (need full runtime)
- **Devnet**: E2E tests, demo, public verification
- **Mock EGISS**: Always mock (fixture-based), production interface preserved
- **Mock KYC**: Always mock (kyc_verified = true on register), production interface preserved
- **Mock notary**: Verified-hash check only (notary_cert_hash compared against known hashes)

---

## 8. Delivery Roadmap

### Monorepo Structure

```
terraledger/
├── .github/
│   └── workflows/
│       ├── contracts.yml          # Anchor build + test
│       ├── backend.yml            # Go lint + test
│       ├── frontend.yml           # Vite build + vitest
│       └── e2e.yml                # Playwright (devnet)
├── contracts/                     # Anchor workspace (see §3)
│   ├── Anchor.toml
│   ├── Cargo.toml
│   ├── programs/
│   │   ├── zher_token/
│   │   └── lien_registry/
│   ├── tests/
│   └── scripts/
├── backend/                       # Go service (see §4)
│   ├── cmd/server/main.go
│   ├── internal/
│   ├── go.mod
│   ├── Makefile
│   └── Dockerfile
├── web/                           # React frontend (see §5)
│   ├── src/
│   ├── package.json
│   ├── vite.config.ts
│   └── Dockerfile
├── sdk/                           # TypeScript SDK
│   ├── src/
│   │   ├── index.ts
│   │   ├── client.ts
│   │   └── types.ts
│   ├── package.json
│   └── tsconfig.json
├── e2e/                           # Playwright tests
│   ├── demo-flow.spec.ts
│   ├── playwright.config.ts
│   └── package.json
├── deployments/
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   └── nginx.conf
├── docs/
│   └── plans/
│       └── 2026-04-04-terraledger.md
├── .env.example
├── Makefile                       # Top-level orchestration
└── README.md
```

### Phased Implementation Plan

#### Phase 1: Foundation (Tasks 1-6)

| Task | Description | DoD | Est. |
|------|-------------|-----|------|
| 1 | Monorepo scaffold: dirs, Makefiles, Anchor.toml, go.mod, package.json, docker-compose | All `make build` commands succeed | 30min |
| 2 | Toolchain validation: Anchor v0.31.1, Solana CLI 2.1.x, Light Protocol SDK compatibility check | `anchor build` succeeds for empty programs, Light SDK imports compile | 30min |
| 3 | zher_token program: ParcelConfig state, register_parcel + verify_parcel instructions, LiteSVM tests | 4 passing tests: register, verify, duplicate reject, PDA derivation | 2h |
| 4 | zher_token: mint_certificate with Token-2022 NonTransferable + MetadataPointer, TransferHook stub | Certificate mints, NonTransferable enforced, metadata readable | 2h |
| 5 | lien_registry program: EncumbranceAccount, LienIndex, register_encumbrance with CPI to verify_parcel, release_encumbrance | CPI works, double-pledge blocked, release allows re-registration | 3h |
| 6 | PostgreSQL schema + Go skeleton: entity models, config, migrations, health endpoint | `make migrate` runs, `GET /health` returns 200 | 1h |

#### Phase 2: Backend Core (Tasks 7-13)

| Task | Description | DoD | Est. |
|------|-------------|-----|------|
| 7 | Clean architecture wiring: repository interfaces, PostgreSQL impls for parcel + cert + lien + lender | Table-driven tests pass for all CRUD operations | 2h |
| 8 | Solana RPC adapter: GetAccountInfo, SendTransaction, account deserialization | Can read parcel PDA from devnet/local validator | 1.5h |
| 9 | REST API handlers: parcel CRUD, credit profile endpoint, lien endpoints, API key middleware | curl tests pass for all endpoints with mock data | 2h |
| 10 | Solana WebSocket indexer: subscribe to both program logs, parse events, update PostgreSQL | Register parcel on-chain → indexer creates PostgreSQL row within 2s | 2h |
| 11 | Copernicus/Sentinel-2 NDVI pipeline: OAuth, scene search, NDVI calculation, async worker | Given coordinates + date, returns NDVI value from real Sentinel-2 data | 3h |
| 12 | Credit scoring: XGBoost model via Python subprocess OR Go-native simplified scorer | Given NDVI history + lien status, returns score 0-100 + grade A-D | 2h |
| 13 | Keeper bot: seasonal check goroutine, builds + sends seasonal_check transactions | Bot detects dormant parcels, emits events, logs actions | 1.5h |

#### Phase 3: Frontend + SDK (Tasks 14-19)

| Task | Description | DoD | Est. |
|------|-------------|-----|------|
| 14 | React scaffold: Vite, @solana/kit, SolanaProvider, TopBar, global styles, CSS tokens | App renders with wallet connect button, connects to Phantom | 1h |
| 15 | Lender Dashboard: cadastral search, credit profile display, MetricCards, NDVIChart | Search parcel → shows full profile with productivity + liens + score | 2.5h |
| 16 | Lien Management: register lien form, release button, active liens list, tx status | Can register and release liens through UI with wallet signing | 2h |
| 17 | Farmer Portal: parcel registration form, land passport view, certificate history | Farmer registers parcel, sees NDVI history and certificates | 2h |
| 18 | PDPA Consent Dashboard: consent status, access log, revoke controls | Farmer sees which lenders accessed data and when | 1.5h |
| 19 | TypeScript SDK: thin REST wrapper, typed responses, npm-publishable | `npm install @terraledger/sdk` → `client.getCreditProfile('KZ19...')` works | 1.5h |

#### Phase 4: Integration + Polish (Tasks 20-23)

| Task | Description | DoD | Est. |
|------|-------------|-----|------|
| 20 | Devnet deployment: deploy both programs, seed test parcels, configure backend for devnet | Programs deployed, 5 test parcels seeded, backend reads real on-chain data | 2h |
| 21 | Playwright E2E: demo flow test (query → lien → re-query → block) | Test passes against devnet with real wallet signing | 2h |
| 22 | CI/CD: GitHub Actions for contracts + backend + frontend, Docker build, deploy script | Push to main → build → push to Yandex Registry → deploy to DO | 2h |
| 23 | Demo preparation: seed demo data (5 Akmola parcels, 3 lenders), verify 4:45 demo script | Full demo runs without errors, Explorer links work | 1h |

### Parallelization Opportunities

As a solo developer, true parallelism is limited, but work can be **pipelined**:

```
Week 1 (Days 1-3): Phase 1 + Phase 2 start
  Day 1: Tasks 1-3 (scaffold + zher_token core)
  Day 2: Tasks 4-5 (Token-2022 + lien_registry)
  Day 3: Tasks 6-8 (Go skeleton + repos + Solana adapter)

Week 1 (Days 4-5): Phase 2 finish
  Day 4: Tasks 9-10 (REST API + indexer)
  Day 5: Tasks 11-13 (NDVI + scoring + keeper)

Week 2 (Days 6-8): Phase 3
  Day 6: Tasks 14-15 (React + Lender Dashboard)
  Day 7: Tasks 16-17 (Liens + Farmer Portal)
  Day 8: Tasks 18-19 (Consent + SDK)

Week 2 (Days 9-10): Phase 4
  Day 9: Tasks 20-21 (Devnet + Playwright)
  Day 10: Tasks 22-23 (CI/CD + Demo)
```

**Key dependency chain:**
```
Contracts (Tasks 3-5) → Solana adapter (Task 8) → Indexer (Task 10) → Frontend integration (Tasks 15-17)
```

Frontend can start (Task 14) before contracts are done, using mock API data. Wire to real backend in Task 15+.

---

## 9. Risks and Open Questions

### Unknowns

| # | Unknown | Investigation Plan | Fallback |
|---|---------|-------------------|----------|
| U1 | Light Protocol SDK + Anchor v0.31.1 compatibility | Task 2: attempt import, build | Standard accounts (no ZK compression) |
| U2 | Copernicus API query latency for Kazakhstan parcels | Task 11: measure real API response time | Pre-cached synthetic NDVI data |
| U3 | XGBoost in Go feasibility | Task 12: test onnxruntime-go or Python subprocess | Simplified linear formula in Go |
| U4 | Token-2022 TransferHook CU consumption | Task 4: profile on solana-test-validator | Simplify hook to single flag check |
| U5 | Helius devnet WebSocket reliability for continuous subscription | Task 10: run for 1 hour, measure drops | Add polling reconciliation (already planned) |

### Architectural Tradeoffs

| Decision | Alternative Considered | Why This Choice |
|----------|----------------------|-----------------|
| Anchor v0.31.1 | v1.0.0 | Proven compatibility with reference patterns, less migration risk |
| PostgreSQL only (no Redis) | PG + Redis | MVP query volume doesn't justify cache layer complexity |
| Go for keeper bot | Separate TypeScript service | Single deployment unit, shared config, no polyglot ops burden |
| WebSocket indexer + polling fallback | Helius webhooks | No external dependency, works on any RPC, polling ensures consistency |
| Tab-based SPA (no router) | React Router | Simpler state, matches creditai/web reference, sufficient for 5 pages |
| Raw Solana JSON-RPC in Go | gagliardetto/solana-go | Fewer dependencies, full control, only need 3-4 RPC methods |
| CSS Modules | Tailwind | Matches project requirement, better encapsulation, no build dependency |

### Validation Checkpoints Before Implementation

| Checkpoint | When | What to Validate |
|-----------|------|-----------------|
| CP1 | After Task 2 | Anchor v0.31.1 builds, Light SDK compiles, solana-test-validator starts |
| CP2 | After Task 5 | CPI between programs works, double-pledge prevented, events emitted |
| CP3 | After Task 10 | Full loop: on-chain event → WebSocket → PostgreSQL → REST API returns updated data |
| CP4 | After Task 15 | Lender can search parcel and see real on-chain data in the browser |
| CP5 | After Task 21 | Playwright demo flow passes end-to-end against devnet |

### XGBoost Model Strategy

**Preferred approach (real model):**
1. Train XGBoost model in Python using synthetic/sample KazAgro-style data
2. Export to ONNX format
3. Load in Go via `onnxruntime-go` package
4. Features: ndvi_trend_slope, cert_count, dormant_seasons, land_class, area_ha, lien_history_count

**Fallback (Go-native scorer):**
```go
func ComputeCreditScore(p *entity.Parcel, certs []entity.NDVICertificate, liens []entity.Encumbrance) int {
    score := 50 // base

    // NDVI trend: +/- 20 points
    if trend := computeNDVITrend(certs); trend > 0 {
        score += min(int(trend*2000), 20)
    } else {
        score += max(int(trend*2000), -20)
    }

    // Certificate count: +5 per cert, max +15
    score += min(len(certs)*5, 15)

    // Clean lien history: +10 if no disputes
    if !hasDisputes(liens) { score += 10 }

    // Land class bonus: class 1-2 = +5
    if p.LandClass <= 2 { score += 5 }

    return clamp(score, 0, 100)
}
```

---

## Appendix A: Key Reference Files

| Reference | Path | What to Use |
|-----------|------|-------------|
| Anchor program structure | `/Users/maximcherbadzhi/go/src/github.com/code7unner/decentrathon/contracts/creditai/programs/creditai/src/` | Instructions layout, state design, events pattern |
| Frontend patterns | `/Users/maximcherbadzhi/go/src/github.com/code7unner/decentrathon/web/src/` | @solana/kit usage, hooks pattern, CSS Modules, SolanaProvider |
| Storage service architecture | `/Users/maximcherbadzhi/go/src/gitlab.com/shelterzoom/storage/internal/` | Clean Architecture layers: adapter/usecase/entity/infrastructure |
| Go code standards | `/Users/maximcherbadzhi/go/src/github.com/code7unner/interview-knowladgebase/CLAUDE.md` | 85-line function limit, naming, error handling, testing patterns |
| TerraLedger concept | `/Users/maximcherbadzhi/go/src/github.com/code7unner/decentrathon5/judge-workspace/iteration-3/eval-11-terraledger/with_skill/outputs/concept.md` | Domain knowledge, account structures, business rules |
| Solana Kit reference | `.claude/skills/solana-dev/references/kit/` | @solana/kit patterns, plugins, advanced usage |

## Appendix B: Environment Variables

```env
# .env.example
APP_NAME=terraledger
PORT=3000
LOG_LEVEL=info

# PostgreSQL
DATABASE_URL=postgres://terraledger:terraledger@localhost:5432/terraledger?sslmode=disable

# Solana
SOLANA_RPC_URL=https://devnet.helius-rpc.com/?api-key=YOUR_KEY
SOLANA_WS_URL=wss://devnet.helius-rpc.com/?api-key=YOUR_KEY
ZHER_TOKEN_PROGRAM_ID=
LIEN_REGISTRY_PROGRAM_ID=
RELAY_KEYPAIR_PATH=~/.config/solana/id.json
HELIUS_API_KEY=YOUR_KEY

# Copernicus (Sentinel-2)
COPERNICUS_CLIENT_ID=YOUR_CLIENT_ID
COPERNICUS_CLIENT_SECRET=YOUR_CLIENT_SECRET

# Keeper
KEEPER_INTERVAL=6h

# Frontend (web/.env)
VITE_SOLANA_RPC_URL=https://devnet.helius-rpc.com/?api-key=YOUR_KEY
VITE_API_URL=http://localhost:3000
```
