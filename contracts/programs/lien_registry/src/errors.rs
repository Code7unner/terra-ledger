use anchor_lang::prelude::*;

#[error_code]
pub enum LienRegistryError {
    #[msg("Active lien already exists on this parcel")]
    ActiveLienExists,
    #[msg("Parcel is not KYC verified")]
    ParcelNotVerified,
    #[msg("Only the lender can release this encumbrance")]
    UnauthorizedRelease,
    #[msg("Encumbrance is not active")]
    EncumbranceNotActive,
    #[msg("Lien amount must be greater than zero")]
    InvalidAmount,
}
