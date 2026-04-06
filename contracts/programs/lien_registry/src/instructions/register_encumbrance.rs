use anchor_lang::prelude::*;

use crate::constants::{ENCUMBRANCE_SEED, LIEN_INDEX_SEED};
use crate::errors::LienRegistryError;
use crate::events::EncumbranceRegistered;
use crate::state::{EncumbranceAccount, LienIndex};
use terra_token::state::ParcelConfig;

#[derive(Accounts)]
#[instruction(cadastral_number: String)]
pub struct RegisterEncumbrance<'info> {
    #[account(
        init,
        payer = lender,
        space = 8 + EncumbranceAccount::INIT_SPACE,
        seeds = [ENCUMBRANCE_SEED, parcel_config.key().as_ref(), lender.key().as_ref()],
        bump,
    )]
    pub encumbrance: Account<'info, EncumbranceAccount>,

    #[account(
        init_if_needed,
        payer = lender,
        space = 8 + LienIndex::INIT_SPACE,
        seeds = [LIEN_INDEX_SEED, parcel_config.key().as_ref()],
        bump,
    )]
    pub lien_index: Account<'info, LienIndex>,

    /// Parcel config from terra_token program. Validated manually in handler.
    /// CHECK: Owner checked against terra_token program ID, data deserialized as ParcelConfig.
    #[account(owner = terra_token::ID)]
    pub parcel_config: UncheckedAccount<'info>,

    #[account(mut)]
    pub lender: Signer<'info>,

    pub system_program: Program<'info, System>,

    /// terra_token program — required in transaction for CPI invoke
    pub terra_token_program: Program<'info, terra_token::program::TerraToken>,
}

pub fn handler(
    ctx: Context<RegisterEncumbrance>,
    cadastral_number: String,
    amount: u64,
    notary_sig_hash: [u8; 32],
    notary_cert_hash: [u8; 32],
) -> Result<()> {
    // CPI: verify parcel exists and PDA seeds match
    let cpi_accounts = terra_token::cpi::accounts::VerifyParcel {
        parcel_config: ctx.accounts.parcel_config.to_account_info(),
    };
    let cpi_ctx = CpiContext::new(
        terra_token::ID,
        cpi_accounts,
    );
    terra_token::cpi::verify_parcel(cpi_ctx, cadastral_number.clone())?;

    // Deserialize parcel config from terra_token program
    let parcel_data = ctx.accounts.parcel_config.try_borrow_data()?;
    let parcel = ParcelConfig::try_deserialize(&mut &parcel_data[..])?;

    // Validate inputs
    require!(amount > 0, LienRegistryError::InvalidAmount);
    require!(parcel.kyc_verified, LienRegistryError::ParcelNotVerified);

    // Check for existing active liens (double-pledge prevention)
    let lien_index = &mut ctx.accounts.lien_index;
    require!(
        lien_index.active_lien_count == 0,
        LienRegistryError::ActiveLienExists
    );

    let parcel_key = ctx.accounts.parcel_config.key();

    // Store encumbrance
    let encumbrance = &mut ctx.accounts.encumbrance;
    encumbrance.parcel_pda = parcel_key;
    encumbrance.lender = ctx.accounts.lender.key();
    encumbrance.amount = amount;
    encumbrance.notary_sig_hash = notary_sig_hash;
    encumbrance.notary_cert_hash = notary_cert_hash;
    encumbrance.egiss_snapshot_hash = parcel.egiss_snapshot_hash;
    encumbrance.registered_at = Clock::get()?.unix_timestamp;
    encumbrance.released_at = 0;
    encumbrance.status = EncumbranceAccount::STATUS_ACTIVE;
    encumbrance.bump = ctx.bumps.encumbrance;

    // Update lien index (safe with init_if_needed: PDA is unique per parcel,
    // and encumbrance uses `init` which prevents double-registration)
    if lien_index.parcel_pda == Pubkey::default() {
        // First time init
        lien_index.parcel_pda = parcel_key;
        lien_index.bump = ctx.bumps.lien_index;
    }
    lien_index.active_lien_count = lien_index.active_lien_count.checked_add(1).unwrap();
    lien_index.total_lien_count = lien_index.total_lien_count.checked_add(1).unwrap();

    emit!(EncumbranceRegistered {
        cadastral_number,
        lender: ctx.accounts.lender.key(),
        amount,
        parcel_pda: parcel_key,
    });

    Ok(())
}
