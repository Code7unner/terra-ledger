use anchor_lang::prelude::*;

use crate::constants::PARCEL_SEED;
use crate::errors::TerraTokenError;
use crate::events::CertificateMinted;
use crate::state::ParcelConfig;

#[derive(Accounts)]
#[instruction(cadastral_number: String)]
pub struct MintCertificate<'info> {
    #[account(
        mut,
        seeds = [PARCEL_SEED, cadastral_number.as_bytes()],
        bump = parcel_config.bump,
        has_one = mint_authority,
    )]
    pub parcel_config: Account<'info, ParcelConfig>,

    /// Either the parcel owner or a whitelisted backend relay.
    pub mint_authority: Signer<'info>,
}

pub fn handler(
    ctx: Context<MintCertificate>,
    cadastral_number: String,
    season: String,
    ndvi_score: u16,
) -> Result<()> {
    let parcel = &mut ctx.accounts.parcel_config;

    require!(parcel.kyc_verified, TerraTokenError::ParcelNotVerified);
    require!(parcel.risk_flag == 0, TerraTokenError::RiskFlagSet);

    parcel.last_cert_epoch = Clock::get()?.slot;
    parcel.cert_count = parcel.cert_count.checked_add(1).unwrap();
    parcel.ndvi_submissions_this_season = parcel
        .ndvi_submissions_this_season
        .checked_add(1)
        .unwrap();

    emit!(CertificateMinted {
        cadastral_number,
        season,
        ndvi_score,
        cert_address: parcel.key(),
    });

    Ok(())
}
