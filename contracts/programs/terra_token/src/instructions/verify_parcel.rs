use anchor_lang::prelude::*;

use crate::constants::PARCEL_SEED;
use crate::state::ParcelConfig;

#[derive(Accounts)]
#[instruction(cadastral_number: String)]
pub struct VerifyParcel<'info> {
    #[account(
        seeds = [PARCEL_SEED, cadastral_number.as_bytes()],
        bump = parcel_config.bump,
    )]
    pub parcel_config: Account<'info, ParcelConfig>,
}

pub fn handler(ctx: Context<VerifyParcel>, _cadastral_number: String) -> Result<()> {
    let _parcel = &ctx.accounts.parcel_config;
    // CPI callers read parcel_config account data directly after this succeeds.
    // If the PDA doesn't exist or seeds don't match, this instruction fails.
    Ok(())
}
