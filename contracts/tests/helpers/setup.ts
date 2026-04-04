import * as anchor from "@anchor-lang/core";
import { Program } from "@anchor-lang/core";
import { BankrunProvider } from "anchor-bankrun";
import { startAnchor, ProgramTestContext } from "solana-bankrun";
import BN from "bn.js";
import { TerraToken } from "../../target/types/terra_token";
import { LienRegistry } from "../../target/types/lien_registry";

// Use anchor.web3 re-exports instead of @solana/web3.js directly
const { Keypair, PublicKey, SystemProgram, LAMPORTS_PER_SOL } = anchor.web3;

export interface TestEnv {
  context: ProgramTestContext;
  provider: BankrunProvider;
  terraToken: Program<TerraToken>;
  lienRegistry: Program<LienRegistry>;
  farmer: anchor.web3.Keypair;
  lender1: anchor.web3.Keypair;
  lender2: anchor.web3.Keypair;
  keeper: anchor.web3.Keypair;
}

export async function createTestEnv(): Promise<TestEnv> {
  const farmer = Keypair.generate();
  const lender1 = Keypair.generate();
  const lender2 = Keypair.generate();
  const keeper = Keypair.generate();

  const context = await startAnchor(
    "",
    [],
    [
      {
        address: farmer.publicKey,
        info: {
          lamports: 100 * LAMPORTS_PER_SOL,
          data: Buffer.alloc(0),
          owner: SystemProgram.programId,
          executable: false,
        },
      },
      {
        address: lender1.publicKey,
        info: {
          lamports: 100 * LAMPORTS_PER_SOL,
          data: Buffer.alloc(0),
          owner: SystemProgram.programId,
          executable: false,
        },
      },
      {
        address: lender2.publicKey,
        info: {
          lamports: 100 * LAMPORTS_PER_SOL,
          data: Buffer.alloc(0),
          owner: SystemProgram.programId,
          executable: false,
        },
      },
      {
        address: keeper.publicKey,
        info: {
          lamports: 10 * LAMPORTS_PER_SOL,
          data: Buffer.alloc(0),
          owner: SystemProgram.programId,
          executable: false,
        },
      },
    ]
  );

  const provider = new BankrunProvider(context);
  anchor.setProvider(provider);

  const terraToken = new Program<TerraToken>(
    require("../../target/idl/terra_token.json"),
    provider,
  );

  const lienRegistry = new Program<LienRegistry>(
    require("../../target/idl/lien_registry.json"),
    provider,
  );

  return {
    context,
    provider,
    terraToken,
    lienRegistry,
    farmer,
    lender1,
    lender2,
    keeper,
  };
}

export function getParcelPda(
  program: Program<TerraToken>,
  cadastralNumber: string,
): [anchor.web3.PublicKey, number] {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("parcel"), Buffer.from(cadastralNumber)],
    program.programId,
  );
}

export function getEncumbrancePda(
  program: Program<LienRegistry>,
  parcelPda: anchor.web3.PublicKey,
  lender: anchor.web3.PublicKey,
): [anchor.web3.PublicKey, number] {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("encumbrance"), parcelPda.toBuffer(), lender.toBuffer()],
    program.programId,
  );
}

export function getLienIndexPda(
  program: Program<LienRegistry>,
  parcelPda: anchor.web3.PublicKey,
): [anchor.web3.PublicKey, number] {
  return PublicKey.findProgramAddressSync(
    [Buffer.from("lien_index"), parcelPda.toBuffer()],
    program.programId,
  );
}

export const TEST_CADASTRAL = "KZ11-0032-001";
export const TEST_CADASTRAL_2 = "KZ11-0032-002";
export const TEST_CADASTRAL_FAKE = "KZ-NONEXIST-999";
export const TEST_CADASTRAL_INTEGRATION = "KZ11-0032-INT";
export const TEST_CADASTRAL_MP1 = "KZ11-0032-MP1";
export const TEST_CADASTRAL_MP2 = "KZ11-0032-MP2";
export const TEST_CADASTRAL_EVT = "KZ11-0032-EVT";
export const TEST_AREA_HA = 4530;
export const TEST_LAND_CLASS = 2;
export const TEST_EGISS_HASH = Array.from(Buffer.alloc(32, 0xab));
export const TEST_NOTARY_SIG_HASH = Array.from(Buffer.alloc(32, 0xcd));
export const TEST_NOTARY_CERT_HASH = Array.from(Buffer.alloc(32, 0xef));
