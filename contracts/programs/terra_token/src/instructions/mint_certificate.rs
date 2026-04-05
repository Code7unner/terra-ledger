#![allow(deprecated)]
use anchor_lang::prelude::*;
use anchor_lang::solana_program;
use anchor_spl::{
    associated_token::AssociatedToken,
    token_2022,
    token_2022_extensions::{
        metadata_pointer, non_transferable,
        spl_token_metadata_interface, token_metadata,
    },
    token_interface::TokenInterface,
};

use crate::constants::PARCEL_SEED;
use crate::errors::TerraTokenError;
use crate::events::CertificateMinted;
use crate::state::ParcelConfig;

type ExtensionType = anchor_spl::token_2022::spl_token_2022::extension::ExtensionType;
type SplMint = anchor_spl::token_2022::spl_token_2022::state::Mint;

/// Extra lamports to cover rent after Token-2022 reallocates the mint
/// for the variable-length TokenMetadata TLV entry.
/// 512 bytes is a generous upper-bound for name + symbol + 3 k/v pairs.
const METADATA_RENT_HEADROOM_BYTES: usize = 512;

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

    /// New keypair that will become the Token-2022 mint for this certificate.
    /// CHECK: Created and initialized within the handler via CPI.
    #[account(mut, signer)]
    pub certificate_mint: AccountInfo<'info>,

    /// ATA for the parcel owner. Created via CPI to the associated-token program.
    /// CHECK: Validated by the associated-token program during create_idempotent.
    #[account(mut)]
    pub token_account: AccountInfo<'info>,

    /// The parcel owner who receives the certificate token.
    /// CHECK: Matched against parcel_config.owner in the handler.
    pub owner: AccountInfo<'info>,

    /// Either the parcel owner or a whitelisted backend relay.
    #[account(mut)]
    pub mint_authority: Signer<'info>,

    pub token_program: Interface<'info, TokenInterface>,
    pub associated_token_program: Program<'info, AssociatedToken>,
    pub system_program: Program<'info, System>,
}

pub fn handler(
    ctx: Context<MintCertificate>,
    cadastral_number: String,
    season: String,
    ndvi_score: u16,
    crop_type: String,
) -> Result<()> {
    let parcel = &mut ctx.accounts.parcel_config;

    require!(parcel.kyc_verified, TerraTokenError::ParcelNotVerified);
    require!(parcel.risk_flag == 0, TerraTokenError::RiskFlagSet);
    require!(
        parcel.owner == ctx.accounts.owner.key(),
        TerraTokenError::OwnerMismatch
    );

    let token_program_id = ctx.accounts.token_program.key();

    // ---------------------------------------------------------------
    // 1. Compute mint account size for the fixed-length extensions.
    //    Token-2022 requires the account size to EXACTLY match
    //    try_calculate_account_len for the initialized extensions.
    //    The variable-length TokenMetadata extension is added later
    //    via realloc inside the Token-2022 program.
    // ---------------------------------------------------------------
    let mint_space = ExtensionType::try_calculate_account_len::<SplMint>(&[
        ExtensionType::NonTransferable,
        ExtensionType::MetadataPointer,
    ])
    .map_err(|_| TerraTokenError::MintSpaceCalculation)?;

    // Pre-fund with extra lamports to cover the metadata realloc.
    // Token-2022 will realloc the account but won't transfer lamports,
    // so the account must already be rent-exempt for the final size.
    let rent = Rent::get()?;
    let lamports = rent.minimum_balance(
        mint_space
            .checked_add(METADATA_RENT_HEADROOM_BYTES)
            .ok_or(TerraTokenError::MintSpaceCalculation)?,
    );

    // ---------------------------------------------------------------
    // 2. Create the mint account (owned by Token-2022 program)
    // ---------------------------------------------------------------
    solana_program::program::invoke(
        &solana_program::system_instruction::create_account(
            ctx.accounts.mint_authority.key,
            ctx.accounts.certificate_mint.key,
            lamports,
            mint_space as u64,
            &token_program_id,
        ),
        &[
            ctx.accounts.mint_authority.to_account_info(),
            ctx.accounts.certificate_mint.to_account_info(),
            ctx.accounts.system_program.to_account_info(),
        ],
    )?;

    // ---------------------------------------------------------------
    // 3. Initialize NonTransferable extension (BEFORE mint init)
    // ---------------------------------------------------------------
    non_transferable::non_transferable_mint_initialize(CpiContext::new(
        token_program_id,
        non_transferable::NonTransferableMintInitialize {
            token_program_id: ctx.accounts.token_program.to_account_info(),
            mint: ctx.accounts.certificate_mint.to_account_info(),
        },
    ))?;

    // ---------------------------------------------------------------
    // 4. Initialize MetadataPointer extension (points to self)
    // ---------------------------------------------------------------
    metadata_pointer::metadata_pointer_initialize(
        CpiContext::new(
            token_program_id,
            metadata_pointer::MetadataPointerInitialize {
                token_program_id: ctx.accounts.token_program.to_account_info(),
                mint: ctx.accounts.certificate_mint.to_account_info(),
            },
        ),
        Some(ctx.accounts.mint_authority.key()),
        Some(ctx.accounts.certificate_mint.key()),
    )?;

    // ---------------------------------------------------------------
    // 5. Initialize the mint (0 decimals, authority = mint_authority)
    // ---------------------------------------------------------------
    token_2022::initialize_mint2(
        CpiContext::new(
            token_program_id,
            token_2022::InitializeMint2 {
                mint: ctx.accounts.certificate_mint.to_account_info(),
            },
        ),
        0,
        &ctx.accounts.mint_authority.key(),
        None,
    )?;

    // ---------------------------------------------------------------
    // 6. Initialize on-chain TokenMetadata.
    //    Token-2022 will realloc the mint account to fit the metadata.
    // ---------------------------------------------------------------
    token_metadata::token_metadata_initialize(
        CpiContext::new(
            token_program_id,
            token_metadata::TokenMetadataInitialize {
                program_id: ctx.accounts.token_program.to_account_info(),
                metadata: ctx.accounts.certificate_mint.to_account_info(),
                update_authority: ctx.accounts.mint_authority.to_account_info(),
                mint_authority: ctx.accounts.mint_authority.to_account_info(),
                mint: ctx.accounts.certificate_mint.to_account_info(),
            },
        ),
        "TerraLedger Certificate".to_string(),
        "TERRA".to_string(),
        String::new(),
    )?;

    // Add custom metadata fields: ndvi_score, season, crop_type
    let field_type = spl_token_metadata_interface::state::Field::Key;

    token_metadata::token_metadata_update_field(
        CpiContext::new(
            token_program_id,
            token_metadata::TokenMetadataUpdateField {
                program_id: ctx.accounts.token_program.to_account_info(),
                metadata: ctx.accounts.certificate_mint.to_account_info(),
                update_authority: ctx.accounts.mint_authority.to_account_info(),
            },
        ),
        field_type("ndvi_score".to_string()),
        ndvi_score.to_string(),
    )?;

    token_metadata::token_metadata_update_field(
        CpiContext::new(
            token_program_id,
            token_metadata::TokenMetadataUpdateField {
                program_id: ctx.accounts.token_program.to_account_info(),
                metadata: ctx.accounts.certificate_mint.to_account_info(),
                update_authority: ctx.accounts.mint_authority.to_account_info(),
            },
        ),
        field_type("season".to_string()),
        season.clone(),
    )?;

    token_metadata::token_metadata_update_field(
        CpiContext::new(
            token_program_id,
            token_metadata::TokenMetadataUpdateField {
                program_id: ctx.accounts.token_program.to_account_info(),
                metadata: ctx.accounts.certificate_mint.to_account_info(),
                update_authority: ctx.accounts.mint_authority.to_account_info(),
            },
        ),
        field_type("crop_type".to_string()),
        crop_type,
    )?;

    // ---------------------------------------------------------------
    // 7. Create associated token account for the parcel owner
    // ---------------------------------------------------------------
    anchor_spl::associated_token::create_idempotent(CpiContext::new(
        ctx.accounts.associated_token_program.key(),
        anchor_spl::associated_token::Create {
            payer: ctx.accounts.mint_authority.to_account_info(),
            associated_token: ctx.accounts.token_account.to_account_info(),
            authority: ctx.accounts.owner.to_account_info(),
            mint: ctx.accounts.certificate_mint.to_account_info(),
            system_program: ctx.accounts.system_program.to_account_info(),
            token_program: ctx.accounts.token_program.to_account_info(),
        },
    ))?;

    // ---------------------------------------------------------------
    // 8. Mint exactly 1 certificate token
    // ---------------------------------------------------------------
    token_2022::mint_to(
        CpiContext::new(
            token_program_id,
            token_2022::MintTo {
                mint: ctx.accounts.certificate_mint.to_account_info(),
                to: ctx.accounts.token_account.to_account_info(),
                authority: ctx.accounts.mint_authority.to_account_info(),
            },
        ),
        1,
    )?;

    // ---------------------------------------------------------------
    // 9. Update ParcelConfig state
    // ---------------------------------------------------------------
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
        cert_address: ctx.accounts.certificate_mint.key(),
    });

    Ok(())
}
