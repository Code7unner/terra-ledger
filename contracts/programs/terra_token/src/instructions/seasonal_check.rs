use anchor_lang::prelude::*;

use crate::constants::{EPOCHS_PER_SEASON, PARCEL_SEED};
use crate::errors::TerraTokenError;
use crate::events::ParcelDormant;
use crate::state::ParcelConfig;

/// Lien registry program ID — used to verify owner of optional lien_index account.
/// Defined as a constant to avoid a circular crate dependency on lien_registry.
const LIEN_REGISTRY_PROGRAM_ID: Pubkey =
    pubkey!("3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4");

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

    /// Optional lien index account from lien_registry program.
    /// CHECK: If provided, owner must be lien_registry program.
    /// Used to check active lien count.
    pub lien_index: Option<UncheckedAccount<'info>>,
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

        let has_active_lien = resolve_active_lien(&ctx.accounts.lien_index);

        emit!(ParcelDormant {
            cadastral_number,
            seasons_dormant: parcel.dormant_seasons,
            has_active_lien,
        });
    } else {
        parcel.dormant_seasons = 0;
    }

    // Reset for next season
    parcel.ndvi_submissions_this_season = 0;

    Ok(())
}

/// Reads the optional lien_index account and returns whether it indicates
/// an active lien. Returns `false` when the account is absent, owned by
/// the wrong program, or too small to contain the expected data.
fn resolve_active_lien(lien_index: &Option<UncheckedAccount>) -> bool {
    let account = match lien_index {
        Some(acc) => acc,
        None => return false,
    };

    if account.owner != &LIEN_REGISTRY_PROGRAM_ID {
        return false;
    }

    let data = match account.try_borrow_data() {
        Ok(d) => d,
        Err(_) => return false,
    };

    // Layout: 8-byte discriminator + 32-byte Pubkey + 1-byte active_lien_count
    const ACTIVE_LIEN_OFFSET: usize = 8 + 32;
    if data.len() < ACTIVE_LIEN_OFFSET + 1 {
        return false;
    }

    data[ACTIVE_LIEN_OFFSET] > 0
}
