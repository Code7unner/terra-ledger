/**
 * Devnet seeding script for TerraLedger.
 *
 * Registers 5 test parcels, mints NDVI certificates on each,
 * and registers a lien on parcel 001.
 *
 * Usage: npx ts-node scripts/seed.ts
 */
import * as anchor from "@anchor-lang/core";
import { Program } from "@anchor-lang/core";
import BN from "bn.js";
import { TerraToken } from "../target/types/terra_token";
import { LienRegistry } from "../target/types/lien_registry";

const { PublicKey, SystemProgram, Keypair } = anchor.web3;
const TOKEN_2022_PROGRAM_ID = new PublicKey("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb");
const ASSOCIATED_TOKEN_PROGRAM_ID = new PublicKey("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL");

// ---------------------------------------------------------------------------
// PDA helpers (mirrors tests/helpers/setup.ts)
// ---------------------------------------------------------------------------

function getParcelPda(
  program: Program<TerraToken>,
  cadastralNumber: string,
): [anchor.web3.PublicKey, number] {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("parcel"), Buffer.from(cadastralNumber)],
    program.programId,
  );
}

function getEncumbrancePda(
  program: Program<LienRegistry>,
  parcelPda: anchor.web3.PublicKey,
  lender: anchor.web3.PublicKey,
): [anchor.web3.PublicKey, number] {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("encumbrance"), parcelPda.toBuffer(), lender.toBuffer()],
    program.programId,
  );
}

function getLienIndexPda(
  program: Program<LienRegistry>,
  parcelPda: anchor.web3.PublicKey,
): [anchor.web3.PublicKey, number] {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("lien_index"), parcelPda.toBuffer()],
    program.programId,
  );
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

interface ParcelSeed {
  cadastral: string;
  areaHa: number;
  landClass: number;
  certificates: { season: string; ndviScore: number }[];
}

const PARCELS: ParcelSeed[] = [
  {
    cadastral: "KZ11-0033-001",
    areaHa: 4530,
    landClass: 2,
    certificates: [
      { season: "2025-Q3", ndviScore: 760 },
      { season: "2025-Q4", ndviScore: 740 },
      { season: "2026-Q1", ndviScore: 780 },
    ],
  },
  {
    cadastral: "KZ11-0033-002",
    areaHa: 2100,
    landClass: 1,
    certificates: [
      { season: "2025-Q3", ndviScore: 680 },
      { season: "2025-Q4", ndviScore: 720 },
    ],
  },
  {
    cadastral: "KZ11-0033-003",
    areaHa: 8750,
    landClass: 3,
    certificates: [
      { season: "2025-Q3", ndviScore: 820 },
      { season: "2025-Q4", ndviScore: 790 },
      { season: "2026-Q1", ndviScore: 810 },
    ],
  },
  {
    cadastral: "KZ11-0033-004",
    areaHa: 1200,
    landClass: 5,
    certificates: [
      { season: "2025-Q4", ndviScore: 650 },
      { season: "2026-Q1", ndviScore: 670 },
    ],
  },
  {
    cadastral: "KZ11-0033-005",
    areaHa: 6300,
    landClass: 2,
    certificates: [
      { season: "2025-Q3", ndviScore: 750 },
      { season: "2025-Q4", ndviScore: 770 },
      { season: "2026-Q1", ndviScore: 760 },
    ],
  },
];

const EGISS_HASH = Array.from(Buffer.alloc(32, 0xab));
const NOTARY_SIG_HASH = Array.from(Buffer.alloc(32, 0xcd));
const NOTARY_CERT_HASH = Array.from(Buffer.alloc(32, 0xef));

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main() {
  // Configure the provider from Anchor.toml / environment
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);

  const terraToken = new Program<TerraToken>(
    require("../target/idl/terra_token.json"),
    provider,
  );

  const lienRegistry = new Program<LienRegistry>(
    require("../target/idl/lien_registry.json"),
    provider,
  );

  const farmer = provider.wallet;
  const lender = Keypair.generate();

  console.log("=== TerraLedger Devnet Seed ===");
  console.log(`Farmer (payer): ${farmer.publicKey.toBase58()}`);
  console.log(`Lender:         ${lender.publicKey.toBase58()}`);
  console.log(`terra_token:    ${terraToken.programId.toBase58()}`);
  console.log(`lien_registry:  ${lienRegistry.programId.toBase58()}`);
  console.log("");

  // Fund the lender account
  console.log("Funding lender account...");
  const fundTx = new anchor.web3.Transaction().add(
    SystemProgram.transfer({
      fromPubkey: farmer.publicKey,
      toPubkey: lender.publicKey,
      lamports: 2 * anchor.web3.LAMPORTS_PER_SOL,
    }),
  );
  await provider.sendAndConfirm(fundTx);
  console.log("  Funded lender with 2 SOL\n");

  // -------------------------------------------------------------------------
  // 1. Register parcels
  // -------------------------------------------------------------------------
  console.log("--- Registering parcels ---");
  for (const parcel of PARCELS) {
    const [parcelPda] = getParcelPda(terraToken, parcel.cadastral);

    try {
      await terraToken.methods
        .registerParcel(
          parcel.cadastral,
          parcel.areaHa,
          parcel.landClass,
          EGISS_HASH,
        )
        .accountsStrict({
          parcelConfig: parcelPda,
          owner: farmer.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .rpc();

      console.log(
        `  [OK] ${parcel.cadastral}  area=${parcel.areaHa}ha  class=${parcel.landClass}  pda=${parcelPda.toBase58()}`,
      );
    } catch (err: any) {
      // Parcel may already exist on re-run
      if (err.toString().includes("already in use")) {
        console.log(`  [SKIP] ${parcel.cadastral} already registered`);
      } else {
        console.error(`  [ERR] ${parcel.cadastral}: ${err}`);
      }
    }
  }
  console.log("");

  // -------------------------------------------------------------------------
  // 2. Mint NDVI certificates
  // -------------------------------------------------------------------------
  console.log("--- Minting NDVI certificates ---");
  for (const parcel of PARCELS) {
    const [parcelPda] = getParcelPda(terraToken, parcel.cadastral);

    for (const cert of parcel.certificates) {
      try {
        const certMint = Keypair.generate();

        // Derive ATA for the farmer (owner) for this certificate mint
        const [tokenAccount] = PublicKey.findProgramAddressSync(
          [
            farmer.publicKey.toBuffer(),
            TOKEN_2022_PROGRAM_ID.toBuffer(),
            certMint.publicKey.toBuffer(),
          ],
          ASSOCIATED_TOKEN_PROGRAM_ID,
        );

        await terraToken.methods
          .mintCertificate(parcel.cadastral, cert.season, cert.ndviScore, "winter_wheat")
          .accountsStrict({
            parcelConfig: parcelPda,
            certificateMint: certMint.publicKey,
            tokenAccount,
            owner: farmer.publicKey,
            mintAuthority: farmer.publicKey,
            tokenProgram: TOKEN_2022_PROGRAM_ID,
            associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
            systemProgram: SystemProgram.programId,
          })
          .signers([certMint])
          .rpc();

        const ndviDecimal = (cert.ndviScore / 1000).toFixed(3);
        console.log(
          `  [OK] ${parcel.cadastral} / ${cert.season}  ndvi=${ndviDecimal}  mint=${certMint.publicKey.toBase58().slice(0, 8)}...`,
        );
      } catch (err: any) {
        console.error(
          `  [ERR] ${parcel.cadastral} / ${cert.season}: ${err}`,
        );
      }
    }
  }
  console.log("");

  // -------------------------------------------------------------------------
  // 3. Register lien on parcel 001
  // -------------------------------------------------------------------------
  console.log("--- Registering lien on KZ11-0033-001 ---");
  const lienCadastral = PARCELS[0].cadastral;
  const [parcelPda001] = getParcelPda(terraToken, lienCadastral);
  const [encumbrancePda] = getEncumbrancePda(
    lienRegistry,
    parcelPda001,
    lender.publicKey,
  );
  const [lienIndexPda] = getLienIndexPda(lienRegistry, parcelPda001);

  const lienAmount = new BN(15_000_000); // 15M tenge equivalent

  try {
    await lienRegistry.methods
      .registerEncumbrance(
        lienCadastral,
        lienAmount,
        NOTARY_SIG_HASH,
        NOTARY_CERT_HASH,
      )
      .accountsStrict({
        encumbrance: encumbrancePda,
        lienIndex: lienIndexPda,
        parcelConfig: parcelPda001,
        lender: lender.publicKey,
        systemProgram: SystemProgram.programId,
        terraTokenProgram: terraToken.programId,
      })
      .signers([lender])
      .rpc();

    console.log(
      `  [OK] Lien registered: amount=${lienAmount.toString()}  lender=${lender.publicKey.toBase58()}`,
    );
    console.log(`       encumbrance PDA: ${encumbrancePda.toBase58()}`);
    console.log(`       lien_index PDA:  ${lienIndexPda.toBase58()}`);
  } catch (err: any) {
    if (err.toString().includes("already in use")) {
      console.log(`  [SKIP] Lien on ${lienCadastral} already registered`);
    } else {
      console.error(`  [ERR] Lien registration: ${err}`);
    }
  }

  // -------------------------------------------------------------------------
  // Summary
  // -------------------------------------------------------------------------
  console.log("\n=== Seed complete ===");
  console.log(`Parcels registered: ${PARCELS.length}`);
  const totalCerts = PARCELS.reduce((s, p) => s + p.certificates.length, 0);
  console.log(`Certificates minted: ${totalCerts}`);
  console.log(`Liens registered: 1`);
  console.log("");

  // Print PDAs for verification
  console.log("--- Parcel PDAs ---");
  for (const parcel of PARCELS) {
    const [pda] = getParcelPda(terraToken, parcel.cadastral);
    console.log(`  ${parcel.cadastral}: ${pda.toBase58()}`);
  }
}

main()
  .then(() => process.exit(0))
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
