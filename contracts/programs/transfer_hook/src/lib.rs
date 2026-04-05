#![allow(ambiguous_glob_reexports)]
use anchor_lang::prelude::*;
use terra_token::state::ParcelConfig;

pub mod errors;
use errors::*;

declare_id!("CpvRLN1XUqpjPw1uHAQcPHQjgbvSF7jnaftwEghta964");

#[program]
pub mod transfer_hook {
    use super::*;

    /// Called by Token-2022 during transfer. Rejects if:
    /// 1. Parcel has risk_flag != 0 (fraud flagged)
    /// 2. Certificate is expired (dormant_seasons > 2)
    pub fn execute(ctx: Context<Execute>, _amount: u64) -> Result<()> {
        let parcel_data = ctx.accounts.parcel_config.try_borrow_data()?;
        let parcel = ParcelConfig::try_deserialize(&mut &parcel_data[..])?;

        require!(parcel.risk_flag == 0, TransferHookError::FraudFlagged);
        require!(
            parcel.dormant_seasons <= 2,
            TransferHookError::CertificateExpired
        );

        Ok(())
    }
}

#[derive(Accounts)]
pub struct Execute<'info> {
    /// CHECK: Token-2022 passes the source token account
    pub source: UncheckedAccount<'info>,
    /// CHECK: Token-2022 passes the mint
    pub mint: UncheckedAccount<'info>,
    /// CHECK: Token-2022 passes the destination
    pub destination: UncheckedAccount<'info>,
    /// CHECK: Token-2022 passes the authority
    pub authority: UncheckedAccount<'info>,
    /// CHECK: Extra account — parcel config for validation
    #[account(owner = terra_token::ID)]
    pub parcel_config: UncheckedAccount<'info>,
}
