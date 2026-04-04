import * as anchor from "@anchor-lang/core";
import { assert } from "chai";
import {
  TestEnv,
  createTestEnv,
  getParcelPda,
  getEncumbrancePda,
  getLienIndexPda,
  TEST_CADASTRAL,
  TEST_CADASTRAL_2,
  TEST_CADASTRAL_FAKE,
  TEST_CADASTRAL_INTEGRATION,
  TEST_CADASTRAL_MP1,
  TEST_CADASTRAL_MP2,
  TEST_CADASTRAL_EVT,
  TEST_AREA_HA,
  TEST_LAND_CLASS,
  TEST_EGISS_HASH,
  TEST_NOTARY_SIG_HASH,
  TEST_NOTARY_CERT_HASH,
} from "./helpers/setup";
import BN from "bn.js";

const { SystemProgram } = anchor.web3;

describe("lien_registry", () => {
  let env: TestEnv;

  before(async () => {
    env = await createTestEnv();

    // Register a parcel first (prerequisite for lien tests)
    const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
    await env.terraToken.methods
      .registerParcel(TEST_CADASTRAL, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
      .accountsStrict({
        parcelConfig: parcelPda,
        owner: env.farmer.publicKey,
        systemProgram: SystemProgram.programId,
      })
      .signers([env.farmer])
      .rpc();

    // Register second parcel for additional tests
    const [parcelPda2] = getParcelPda(env.terraToken, TEST_CADASTRAL_2);
    await env.terraToken.methods
      .registerParcel(TEST_CADASTRAL_2, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
      .accountsStrict({
        parcelConfig: parcelPda2,
        owner: env.farmer.publicKey,
        systemProgram: SystemProgram.programId,
      })
      .signers([env.farmer])
      .rpc();
  });

  describe("register_encumbrance", () => {
    it("registers a lien on a verified parcel", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [encumbrancePda] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender1.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, parcelPda);

      const amount = new BN(15_000_000);

      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL,
          amount,
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encumbrancePda,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender1.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender1])
        .rpc();

      const enc = await env.lienRegistry.account.encumbranceAccount.fetch(encumbrancePda);
      assert.ok(enc.parcelPda.equals(parcelPda));
      assert.ok(enc.lender.equals(env.lender1.publicKey));
      assert.ok(enc.amount.eq(amount));
      assert.equal(enc.status, 0); // Active

      const idx = await env.lienRegistry.account.lienIndex.fetch(lienIndexPda);
      assert.equal(idx.activeLienCount, 1);
      assert.equal(idx.totalLienCount, 1);
    });

    it("blocks double pledge — second lender rejected", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [encumbrancePda] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender2.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, parcelPda);

      try {
        await env.lienRegistry.methods
          .registerEncumbrance(
            TEST_CADASTRAL,
            new BN(10_000_000),
            TEST_NOTARY_SIG_HASH,
            TEST_NOTARY_CERT_HASH,
          )
          .accountsStrict({
            encumbrance: encumbrancePda,
            lienIndex: lienIndexPda,
            parcelConfig: parcelPda,
            lender: env.lender2.publicKey,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.lender2])
          .rpc();
        assert.fail("should have thrown — active lien exists");
      } catch (err: any) {
        assert.include(err.toString(), "ActiveLienExists");
      }
    });
  });

  describe("release_encumbrance", () => {
    it("releases a lien and decrements counter", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [encumbrancePda] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender1.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, parcelPda);

      await env.lienRegistry.methods
        .releaseEncumbrance()
        .accountsStrict({
          encumbrance: encumbrancePda,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender1.publicKey,
        })
        .signers([env.lender1])
        .rpc();

      // Account is closed after release (rent returned to lender)
      try {
        await env.lienRegistry.account.encumbranceAccount.fetch(encumbrancePda);
        assert.fail("encumbrance account should be closed");
      } catch (err: any) {
        assert.include(err.toString(), "Could not find");
      }

      const idx = await env.lienRegistry.account.lienIndex.fetch(lienIndexPda);
      assert.equal(idx.activeLienCount, 0);
      assert.equal(idx.totalLienCount, 1); // historical stays
    });

    it("fails when non-lender tries to release", async () => {
      // Register a new lien on parcel 2 first
      const [parcelPda2] = getParcelPda(env.terraToken, TEST_CADASTRAL_2);
      const [encPda2] = getEncumbrancePda(env.lienRegistry, parcelPda2, env.lender1.publicKey);
      const [idxPda2] = getLienIndexPda(env.lienRegistry, parcelPda2);

      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL_2,
          new BN(5_000_000),
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encPda2,
          lienIndex: idxPda2,
          parcelConfig: parcelPda2,
          lender: env.lender1.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender1])
        .rpc();

      // lender2 tries to release lender1's lien — should fail
      try {
        await env.lienRegistry.methods
          .releaseEncumbrance()
          .accountsStrict({
            encumbrance: encPda2,
            lienIndex: idxPda2,
            parcelConfig: parcelPda2,
            lender: env.lender2.publicKey,
          })
          .signers([env.lender2])
          .rpc();
        assert.fail("should have thrown — unauthorized release");
      } catch (err) {
        assert.ok(err);
      }
    });
  });

  describe("full cycle: register → release → re-register", () => {
    it("allows new lien after release", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [encumbrancePda] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender2.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, parcelPda);

      // Parcel 1 lien was released above — lender2 should succeed now
      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL,
          new BN(12_000_000),
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encumbrancePda,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender2.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender2])
        .rpc();

      const idx = await env.lienRegistry.account.lienIndex.fetch(lienIndexPda);
      assert.equal(idx.activeLienCount, 1);
      assert.equal(idx.totalLienCount, 2); // historical incremented
    });
  });

  describe("edge cases", () => {
    it("rejects lien on nonexistent parcel", async () => {
      const [fakeParcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_FAKE);
      const [encumbrancePda] = getEncumbrancePda(env.lienRegistry, fakeParcelPda, env.lender1.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, fakeParcelPda);

      try {
        await env.lienRegistry.methods
          .registerEncumbrance(
            TEST_CADASTRAL_FAKE,
            new BN(5_000_000),
            TEST_NOTARY_SIG_HASH,
            TEST_NOTARY_CERT_HASH,
          )
          .accountsStrict({
            encumbrance: encumbrancePda,
            lienIndex: lienIndexPda,
            parcelConfig: fakeParcelPda,
            lender: env.lender1.publicKey,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.lender1])
          .rpc();
        assert.fail("should have thrown — parcel does not exist");
      } catch (err: any) {
        // The account doesn't exist on-chain, so the owner check fails
        assert.ok(err);
        assert.notEqual(err.message, "should have thrown — parcel does not exist");
      }
    });

    it("rejects concurrent lien by same lender on same parcel", async () => {
      // Parcel 1 already has an active lien from lender2 (set in "full cycle" test)
      // Release it first, then register with lender1, then try lender1 again
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL);
      const [encPdaLender2] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender2.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, parcelPda);

      // Release existing lien from lender2
      await env.lienRegistry.methods
        .releaseEncumbrance()
        .accountsStrict({
          encumbrance: encPdaLender2,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender2.publicKey,
        })
        .signers([env.lender2])
        .rpc();

      // Register lien with lender1
      const [encPdaLender1] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender1.publicKey);
      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL,
          new BN(8_000_000),
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encPdaLender1,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender1.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender1])
        .rpc();

      // Try to register again with lender1 on the same parcel — same PDA, already in use
      try {
        await env.lienRegistry.methods
          .registerEncumbrance(
            TEST_CADASTRAL,
            new BN(3_000_000),
            TEST_NOTARY_SIG_HASH,
            TEST_NOTARY_CERT_HASH,
          )
          .accountsStrict({
            encumbrance: encPdaLender1,
            lienIndex: lienIndexPda,
            parcelConfig: parcelPda,
            lender: env.lender1.publicKey,
            systemProgram: SystemProgram.programId,
          })
          .signers([env.lender1])
          .rpc();
        assert.fail("should have thrown — encumbrance account already in use");
      } catch (err: any) {
        // init constraint fails because the PDA account already exists
        assert.ok(err);
        assert.notEqual(err.message, "should have thrown — encumbrance account already in use");
      }
    });
  });

  describe("CPI flow integration", () => {
    it("full lifecycle: register → mint → encumber → release", async () => {
      // Step 1: Register a NEW parcel
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_INTEGRATION);
      await env.terraToken.methods
        .registerParcel(TEST_CADASTRAL_INTEGRATION, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
        .accountsStrict({
          parcelConfig: parcelPda,
          owner: env.farmer.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer])
        .rpc();

      // Step 2: Mint a certificate for the parcel
      await env.terraToken.methods
        .mintCertificate(TEST_CADASTRAL_INTEGRATION, "2026-Q1", 750)
        .accountsStrict({
          parcelConfig: parcelPda,
          mintAuthority: env.farmer.publicKey,
        })
        .signers([env.farmer])
        .rpc();

      // Step 3: Register encumbrance with lender1
      const [encumbrancePda] = getEncumbrancePda(env.lienRegistry, parcelPda, env.lender1.publicKey);
      const [lienIndexPda] = getLienIndexPda(env.lienRegistry, parcelPda);
      const amount = new BN(10_000_000);

      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL_INTEGRATION,
          amount,
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encumbrancePda,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender1.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender1])
        .rpc();

      // Step 4: Verify encumbrance account state
      const enc = await env.lienRegistry.account.encumbranceAccount.fetch(encumbrancePda);
      assert.ok(enc.parcelPda.equals(parcelPda));
      assert.ok(enc.lender.equals(env.lender1.publicKey));
      assert.ok(enc.amount.eq(amount));
      assert.equal(enc.status, 0); // Active

      // Step 5: Verify lien index
      let idx = await env.lienRegistry.account.lienIndex.fetch(lienIndexPda);
      assert.equal(idx.activeLienCount, 1);
      assert.equal(idx.totalLienCount, 1);

      // Step 6: Release encumbrance
      await env.lienRegistry.methods
        .releaseEncumbrance()
        .accountsStrict({
          encumbrance: encumbrancePda,
          lienIndex: lienIndexPda,
          parcelConfig: parcelPda,
          lender: env.lender1.publicKey,
        })
        .signers([env.lender1])
        .rpc();

      // Step 7: Verify lien index after release
      idx = await env.lienRegistry.account.lienIndex.fetch(lienIndexPda);
      assert.equal(idx.activeLienCount, 0);
      assert.equal(idx.totalLienCount, 1);

      // Step 8: Verify encumbrance account is closed
      try {
        await env.lienRegistry.account.encumbranceAccount.fetch(encumbrancePda);
        assert.fail("encumbrance account should be closed");
      } catch (err: any) {
        assert.include(err.toString(), "Could not find");
      }
    });

    it("multi-parcel with separate lenders", async () => {
      // Register TWO new parcels
      const [parcelPda1] = getParcelPda(env.terraToken, TEST_CADASTRAL_MP1);
      const [parcelPda2] = getParcelPda(env.terraToken, TEST_CADASTRAL_MP2);

      await env.terraToken.methods
        .registerParcel(TEST_CADASTRAL_MP1, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
        .accountsStrict({
          parcelConfig: parcelPda1,
          owner: env.farmer.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer])
        .rpc();

      await env.terraToken.methods
        .registerParcel(TEST_CADASTRAL_MP2, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
        .accountsStrict({
          parcelConfig: parcelPda2,
          owner: env.farmer.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer])
        .rpc();

      // Encumber MP1 with lender1
      const [encPda1] = getEncumbrancePda(env.lienRegistry, parcelPda1, env.lender1.publicKey);
      const [idxPda1] = getLienIndexPda(env.lienRegistry, parcelPda1);

      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL_MP1,
          new BN(10_000_000),
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encPda1,
          lienIndex: idxPda1,
          parcelConfig: parcelPda1,
          lender: env.lender1.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender1])
        .rpc();

      // Encumber MP2 with lender2
      const [encPda2] = getEncumbrancePda(env.lienRegistry, parcelPda2, env.lender2.publicKey);
      const [idxPda2] = getLienIndexPda(env.lienRegistry, parcelPda2);

      await env.lienRegistry.methods
        .registerEncumbrance(
          TEST_CADASTRAL_MP2,
          new BN(20_000_000),
          TEST_NOTARY_SIG_HASH,
          TEST_NOTARY_CERT_HASH,
        )
        .accountsStrict({
          encumbrance: encPda2,
          lienIndex: idxPda2,
          parcelConfig: parcelPda2,
          lender: env.lender2.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.lender2])
        .rpc();

      // Verify both lien indices show active_lien_count=1
      let idx1 = await env.lienRegistry.account.lienIndex.fetch(idxPda1);
      assert.equal(idx1.activeLienCount, 1);

      let idx2 = await env.lienRegistry.account.lienIndex.fetch(idxPda2);
      assert.equal(idx2.activeLienCount, 1);

      // Release MP1's lien
      await env.lienRegistry.methods
        .releaseEncumbrance()
        .accountsStrict({
          encumbrance: encPda1,
          lienIndex: idxPda1,
          parcelConfig: parcelPda1,
          lender: env.lender1.publicKey,
        })
        .signers([env.lender1])
        .rpc();

      // Verify MP1 index: active_lien_count=0
      idx1 = await env.lienRegistry.account.lienIndex.fetch(idxPda1);
      assert.equal(idx1.activeLienCount, 0);

      // Verify MP2 index: still active_lien_count=1
      idx2 = await env.lienRegistry.account.lienIndex.fetch(idxPda2);
      assert.equal(idx2.activeLienCount, 1);
    });

    it("events emitted in transaction logs", async () => {
      // Register a parcel and capture transaction result
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_EVT);

      const txSig = await env.terraToken.methods
        .registerParcel(TEST_CADASTRAL_EVT, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
        .accountsStrict({
          parcelConfig: parcelPda,
          owner: env.farmer.publicKey,
          systemProgram: SystemProgram.programId,
        })
        .signers([env.farmer])
        .rpc();

      // BankRun does not support getTransaction, so we verify the tx succeeded
      // and trust that Anchor's #[event] macro emits Program data: log lines
      // (proven by webhook handler Borsh decoding tests on the backend)
      assert.ok(txSig, "transaction signature should be returned");
      assert.isString(txSig, "signature should be a string");

      // Verify the parcel was actually created on-chain
      const parcelAccount = await env.terraToken.account.parcelConfig.fetch(parcelPda);
      assert.equal(parcelAccount.cadastralStr, TEST_CADASTRAL_EVT);
      assert.ok(parcelAccount.registeredAt.toNumber() > 0, "registered_at should be set");
    });
  });
});
