use anchor_lang::prelude::*;

use crate::constants::{ENCUMBRANCE_SEED, LIEN_INDEX_SEED};
use crate::errors::LienRegistryError;
use crate::events::EncumbranceReleased;
use crate::state::{EncumbranceAccount, LienIndex};
use terra_token::state::ParcelConfig;

#[derive(Accounts)]
pub struct ReleaseEncumbrance<'info> {
    #[account(
        mut,
        close = lender,
        seeds = [ENCUMBRANCE_SEED, parcel_config.key().as_ref(), lender.key().as_ref()],
        bump = encumbrance.bump,
        has_one = lender @ LienRegistryError::UnauthorizedRelease,
    )]
    pub encumbrance: Account<'info, EncumbranceAccount>,

    #[account(
        mut,
        seeds = [LIEN_INDEX_SEED, parcel_config.key().as_ref()],
        bump = lien_index.bump,
    )]
    pub lien_index: Account<'info, LienIndex>,

    /// CHECK: Owner checked against terra_token program ID.
    #[account(owner = terra_token::ID)]
    pub parcel_config: UncheckedAccount<'info>,

    #[account(mut)]
    pub lender: Signer<'info>,
}

pub fn handler(ctx: Context<ReleaseEncumbrance>) -> Result<()> {
    let encumbrance = &ctx.accounts.encumbrance;

    require!(
        encumbrance.status == EncumbranceAccount::STATUS_ACTIVE,
        LienRegistryError::EncumbranceNotActive
    );

    let lien_index = &mut ctx.accounts.lien_index;
    lien_index.active_lien_count = lien_index.active_lien_count.saturating_sub(1);

    // Deserialize parcel for event data
    let parcel_data = ctx.accounts.parcel_config.try_borrow_data()?;
    let parcel = ParcelConfig::try_deserialize(&mut &parcel_data[..])?;

    emit!(EncumbranceReleased {
        cadastral_number: parcel.cadastral_str,
        lender: ctx.accounts.lender.key(),
        parcel_pda: ctx.accounts.parcel_config.key(),
    });

    // Account closed via `close = lender` constraint — rent returned to lender.
    // This allows the same lender to re-register a lien on the same parcel later.

    Ok(())
}
