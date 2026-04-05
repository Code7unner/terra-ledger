import * as anchor from "@anchor-lang/core";
import { assert } from "chai";
import {
  TestEnv,
  createTestEnv,
  getParcelPda,
  TEST_CADASTRAL,
  TEST_CADASTRAL_2,
  TEST_AREA_HA,
  TEST_LAND_CLASS,
  TEST_EGISS_HASH,
} from "./helpers/setup";

const { Keypair, PublicKey, SystemProgram } = anchor.web3;

// Token-2022 program ID
const TOKEN_2022_PROGRAM_ID = new PublicKey(
  "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb",
);

// Associated Token Account program ID
const ASSOCIATED_TOKEN_PROGRAM_ID = new PublicKey(
  "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL",
);

/**
 * Derive the associated token account address for a given owner and mint,
 * using the Token-2022 program as the token program.
 */
function getAssociatedTokenAddress(
  mint: anchor.web3.PublicKey,
  owner: anchor.web3.PublicKey,
): anchor.web3.PublicKey {
  const [address] = PublicKey.findProgramAddressSync(
    [owner.toBuffer(), TOKEN_2022_PROGRAM_ID.toBuffer(), mint.toBuffer()],
    ASSOCIATED_TOKEN_PROGRAM_ID,
  );
  return address;
}

describe("terra_token", () => {
  let env: TestEnv;

  before(async () => {
    env = await createTestEnv();
  });

  describe("register_parcel", () => {
    it("registers a parcel with correct state", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);

      await env.terraToken.methods
        .registerParcel(
          TEST_CADASTRAL,
          TEST_AREA_HA,
          TEST_LAND_CLASS,
          TEST_EGISS_HASH,
        )
        .accountsStrict({
          parcelConfig: parcelPda,
          owner: env.farmer.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer])
        .rpc();

      const parcel = await env.terraToken.account.parcelConfig.fetch(parcelPda);
      assert.ok(parcel.owner.equals(env.farmer.publicKey));
      assert.ok(parcel.mintAuthority.equals(env.farmer.publicKey));
      assert.equal(parcel.cadastralStr, TEST_CADASTRAL);
      assert.equal(parcel.areaHa, TEST_AREA_HA);
      assert.equal(parcel.landClass, TEST_LAND_CLASS);
      assert.equal(parcel.kycVerified, true);
      assert.equal(parcel.kycMethod, 1);
      assert.equal(parcel.certCount, 0);
      assert.equal(parcel.dormantSeasons, 0);
      assert.equal(parcel.riskFlag, 0);
    });

    it("fails on duplicate registration", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);

      try {
        await env.terraToken.methods
          .registerParcel(
            TEST_CADASTRAL,
            TEST_AREA_HA,
            TEST_LAND_CLASS,
            TEST_EGISS_HASH,
          )
          .accountsStrict({
            parcelConfig: parcelPda,
            owner: env.farmer.publicKey,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.farmer])
          .rpc();
        assert.fail("should have thrown");
      } catch (err) {
        assert.ok(err);
      }
    });

    it("fails with invalid land class", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, "KZ-INVALID-LC");

      try {
        await env.terraToken.methods
          .registerParcel(
            "KZ-INVALID-LC",
            TEST_AREA_HA,
            9, // invalid: must be 1-8
            TEST_EGISS_HASH,
          )
          .accountsStrict({
            parcelConfig: parcelPda,
            owner: env.farmer.publicKey,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.farmer])
          .rpc();
        assert.fail("should have thrown");
      } catch (err: any) {
        assert.include(err.toString(), "InvalidLandClass");
      }
    });

    it("fails with zero area", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, "KZ-ZERO-AREA");

      try {
        await env.terraToken.methods
          .registerParcel(
            "KZ-ZERO-AREA",
            0, // invalid
            TEST_LAND_CLASS,
            TEST_EGISS_HASH,
          )
          .accountsStrict({
            parcelConfig: parcelPda,
            owner: env.farmer.publicKey,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.farmer])
          .rpc();
        assert.fail("should have thrown");
      } catch (err: any) {
        assert.include(err.toString(), "InvalidArea");
      }
    });
  });

  describe("verify_parcel", () => {
    it("succeeds for registered parcel", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);

      await env.terraToken.methods
        .verifyParcel(TEST_CADASTRAL)
        .accountsStrict({
          parcelConfig: parcelPda,
        })
        .rpc();
    });

    it("fails for non-existent parcel", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, "KZ-NONEXIST");

      try {
        await env.terraToken.methods
          .verifyParcel("KZ-NONEXIST")
          .accountsStrict({
            parcelConfig: parcelPda,
          })
          .rpc();
        assert.fail("should have thrown");
      } catch (err) {
        assert.ok(err);
      }
    });
  });

  describe("mint_certificate", () => {
    it("mints a certificate and increments counters", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const certMint = Keypair.generate();
      const tokenAccount = getAssociatedTokenAddress(
        certMint.publicKey,
        env.farmer.publicKey,
      );

      await env.terraToken.methods
        .mintCertificate(TEST_CADASTRAL, "2026-Q1", 760, "wheat")
        .accountsStrict({
          parcelConfig: parcelPda,
          certificateMint: certMint.publicKey,
          tokenAccount,
          owner: env.farmer.publicKey,
          mintAuthority: env.farmer.publicKey,
          tokenProgram: TOKEN_2022_PROGRAM_ID,
          associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer, certMint])
        .rpc();

      const parcel = await env.terraToken.account.parcelConfig.fetch(parcelPda);
      assert.equal(parcel.certCount, 1);
      assert.equal(parcel.ndviSubmissionsThisSeason, 1);
    });

    it("mints multiple certificates", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const certMint = Keypair.generate();
      const tokenAccount = getAssociatedTokenAddress(
        certMint.publicKey,
        env.farmer.publicKey,
      );

      await env.terraToken.methods
        .mintCertificate(TEST_CADASTRAL, "2026-Q2", 740, "barley")
        .accountsStrict({
          parcelConfig: parcelPda,
          certificateMint: certMint.publicKey,
          tokenAccount,
          owner: env.farmer.publicKey,
          mintAuthority: env.farmer.publicKey,
          tokenProgram: TOKEN_2022_PROGRAM_ID,
          associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer, certMint])
        .rpc();

      const parcel = await env.terraToken.account.parcelConfig.fetch(parcelPda);
      assert.equal(parcel.certCount, 2);
      assert.equal(parcel.ndviSubmissionsThisSeason, 2);
    });

    it("fails when non-authority tries to mint", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const certMint = Keypair.generate();
      const tokenAccount = getAssociatedTokenAddress(
        certMint.publicKey,
        env.farmer.publicKey,
      );

      try {
        await env.terraToken.methods
          .mintCertificate(TEST_CADASTRAL, "2026-Q3", 700, "corn")
          .accountsStrict({
            parcelConfig: parcelPda,
            certificateMint: certMint.publicKey,
            tokenAccount,
            owner: env.farmer.publicKey,
            mintAuthority: env.lender1.publicKey,
            tokenProgram: TOKEN_2022_PROGRAM_ID,
            associatedTokenProgram: ASSOCIATED_TOKEN_PROGRAM_ID,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.lender1, certMint])
          .rpc();
        assert.fail("should have thrown");
      } catch (err) {
        assert.ok(err);
      }
    });
  });

  describe("PDA derivation", () => {
    it("derives deterministic PDAs from cadastral number", () => {
      const [pda1a] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [pda1b] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [pda2] = getParcelPda(env.terraToken, TEST_CADASTRAL_2);

      assert.ok(pda1a.equals(pda1b), "same cadastral should give same PDA");
      assert.ok(!pda1a.equals(pda2), "different cadastral should give different PDA");
    });
  });
});
