use anchor_lang::prelude::*;

use crate::constants::{MAX_CADASTRAL_LEN, PARCEL_SEED};
use crate::errors::TerraTokenError;
use crate::events::ParcelRegistered;
use crate::state::ParcelConfig;

#[derive(Accounts)]
#[instruction(cadastral_number: String)]
pub struct RegisterParcel<'info> {
    #[account(
        init,
        payer = owner,
        space = 8 + ParcelConfig::INIT_SPACE,
        seeds = [PARCEL_SEED, cadastral_number.as_bytes()],
        bump,
    )]
    pub parcel_config: Account<'info, ParcelConfig>,

    #[account(mut)]
    pub owner: Signer<'info>,

    pub system_program: Program<'info, System>,
}

pub fn handler(
    ctx: Context<RegisterParcel>,
    cadastral_number: String,
    area_ha: u32,
    land_class: u8,
    egiss_snapshot_hash: [u8; 32],
) -> Result<()> {
    require!(
        cadastral_number.len() <= MAX_CADASTRAL_LEN,
        TerraTokenError::CadastralTooLong
    );
    require!(area_ha > 0, TerraTokenError::InvalidArea);
    require!(
        land_class >= 1 && land_class <= 8,
        TerraTokenError::InvalidLandClass
    );

    let parcel = &mut ctx.accounts.parcel_config;
    parcel.owner = ctx.accounts.owner.key();
    parcel.mint_authority = ctx.accounts.owner.key(); // defaults to owner; can be updated to relay
    parcel.cadastral_str = cadastral_number.clone();
    parcel.area_ha = area_ha;
    parcel.land_class = land_class;
    parcel.kyc_verified = true; // Mock KYC
    parcel.kyc_method = 1; // EGISS_NCA
    parcel.last_cert_epoch = 0;
    parcel.cert_count = 0;
    parcel.ndvi_submissions_this_season = 0;
    parcel.dormant_seasons = 0;
    parcel.egiss_snapshot_hash = egiss_snapshot_hash;
    parcel.risk_flag = 0;
    parcel.registered_at = Clock::get()?.unix_timestamp;
    parcel.bump = ctx.bumps.parcel_config;

    emit!(ParcelRegistered {
        cadastral_number,
        owner: parcel.owner,
        area_ha,
    });

    Ok(())
}
