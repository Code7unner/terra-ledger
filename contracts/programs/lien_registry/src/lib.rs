#![allow(ambiguous_glob_reexports)]
use anchor_lang::prelude::*;

pub mod constants;
pub mod errors;
pub mod events;
pub mod instructions;
pub mod state;

use instructions::*;

declare_id!("3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4");

#[program]
pub mod lien_registry {
    use super::*;

    pub fn register_encumbrance(
        ctx: Context<RegisterEncumbrance>,
        cadastral_number: String,
        amount: u64,
        notary_sig_hash: [u8; 32],
        notary_cert_hash: [u8; 32],
    ) -> Result<()> {
        instructions::register_encumbrance::handler(
            ctx,
            cadastral_number,
            amount,
            notary_sig_hash,
            notary_cert_hash,
        )
    }

    pub fn release_encumbrance(ctx: Context<ReleaseEncumbrance>) -> Result<()> {
        instructions::release_encumbrance::handler(ctx)
    }

}
