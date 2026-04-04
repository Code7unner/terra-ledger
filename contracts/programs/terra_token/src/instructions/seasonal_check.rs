use anchor_lang::prelude::*;

use crate::constants::{EPOCHS_PER_SEASON, PARCEL_SEED};
use crate::errors::TerraTokenError;
use crate::events::ParcelDormant;
use crate::state::ParcelConfig;

#[derive(Accounts)]
#[instruction(cadastral_number: String)]
pub struct SeasonalCheck<'info> {
    #[account(
        mut,
        seeds = [PARCEL_SEED, cadastral_number.as_bytes()],
        bump = parcel_config.bump,
    )]
    pub parcel_config: Account<'info, ParcelConfig>,

    /// Permissionless crank — any wallet can trigger seasonal checks.
    /// This is an intentional design choice (same pattern as major Solana protocols).
    /// On-chain timing constraints prevent abuse.
    pub keeper: Signer<'info>,
}

pub fn handler(ctx: Context<SeasonalCheck>, cadastral_number: String) -> Result<()> {
    let parcel = &mut ctx.accounts.parcel_config;
    let current_slot = Clock::get()?.slot;

    require!(
        parcel.last_cert_epoch == 0
            || current_slot >= parcel.last_cert_epoch + EPOCHS_PER_SEASON,
        TerraTokenError::TooEarlyForSeasonalCheck
    );

    if parcel.ndvi_submissions_this_season == 0 {
        parcel.dormant_seasons = parcel.dormant_seasons.saturating_add(1);

        emit!(ParcelDormant {
            cadastral_number,
            seasons_dormant: parcel.dormant_seasons,
        });
    } else {
        parcel.dormant_seasons = 0;
    }

    // Reset for next season
    parcel.ndvi_submissions_this_season = 0;

    Ok(())
}
