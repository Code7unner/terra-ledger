import * as anchor from "@anchor-lang/core";
import { Program } from "@anchor-lang/core";
import { expect } from "chai";
import { TransferHook } from "../target/types/transfer_hook";
import {
  createTestEnv,
  getParcelPda,
  TestEnv,
  TEST_AREA_HA,
  TEST_LAND_CLASS,
  TEST_EGISS_HASH,
} from "./helpers/setup";

const { Keypair, SystemProgram } = anchor.web3;

const TEST_CADASTRAL_HOOK = "KZ11-0032-HK1";
const TEST_CADASTRAL_DORMANT = "KZ11-0032-HK2";

describe("transfer_hook", () => {
  let env: TestEnv;
  let transferHook: Program<TransferHook>;

  before(async () => {
    env = await createTestEnv();

    transferHook = new Program<TransferHook>(
      require("../target/idl/transfer_hook.json"),
      env.provider,
    );

    // Register a healthy parcel (risk_flag=0, dormant_seasons=0)
    const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_HOOK);
    await env.terraToken.methods
      .registerParcel(TEST_CADASTRAL_HOOK, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
      .accounts({ parcelConfig: parcelPda, owner: env.farmer.publicKey, systemProgram: SystemProgram.programId })
      .signers([env.farmer])
      .rpc();

    // Register a parcel that will become dormant
    const [dormantPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_DORMANT);
    await env.terraToken.methods
      .registerParcel(TEST_CADASTRAL_DORMANT, TEST_AREA_HA, TEST_LAND_CLASS, TEST_EGISS_HASH)
      .accounts({ parcelConfig: dormantPda, owner: env.farmer.publicKey, systemProgram: SystemProgram.programId })
      .signers([env.farmer])
      .rpc();
  });

  describe("execute", () => {
    it("succeeds for healthy parcel (risk_flag=0, dormant_seasons=0)", async () => {
      const [parcelPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_HOOK);
      const dummy = Keypair.generate();

      // Should not throw
      await transferHook.methods
        .execute(new anchor.BN(1))
        .accountsStrict({
          source: dummy.publicKey,
          mint: dummy.publicKey,
          destination: dummy.publicKey,
          authority: dummy.publicKey,
          parcelConfig: parcelPda,
        })
        .rpc();
    });

    it("rejects when dormant_seasons > 2", async () => {
      const [dormantPda] = getParcelPda(env.terraToken, TEST_CADASTRAL_DORMANT);

      // Run seasonal_check 3 times with different keepers to avoid tx dedup.
      // Parcel has 0 ndvi submissions, so each check increments dormant_seasons.
      const keepers = [env.keeper, env.lender1, env.lender2];
      for (let i = 0; i < 3; i++) {
        await env.terraToken.methods
          .seasonalCheck(TEST_CADASTRAL_DORMANT)
          .accounts({ parcelConfig: dormantPda, keeper: keepers[i].publicKey, lienIndex: null })
          .signers([keepers[i]])
          .rpc();
      }

      // Verify dormant_seasons = 3
      const parcel = await env.terraToken.account.parcelConfig.fetch(dormantPda);
      expect(parcel.dormantSeasons).to.equal(3);

      // transfer_hook::execute should fail
      const dummy = Keypair.generate();
      try {
        await transferHook.methods
          .execute(new anchor.BN(1))
          .accountsStrict({
            source: dummy.publicKey,
            mint: dummy.publicKey,
            destination: dummy.publicKey,
            authority: dummy.publicKey,
            parcelConfig: dormantPda,
          })
          .rpc();
        expect.fail("should have thrown CertificateExpired");
      } catch (err: any) {
        expect(err.toString()).to.include("CertificateExpired");
      }
    });
  });
});
