use anchor_lang::prelude::*;

#[error_code]
pub enum TerraTokenError {
    #[msg("Cadastral number exceeds maximum length")]
    CadastralTooLong,
    #[msg("Parcel is not KYC verified")]
    ParcelNotVerified,
    #[msg("Too early for seasonal check")]
    TooEarlyForSeasonalCheck,
    #[msg("Risk flag is set — operation blocked")]
    RiskFlagSet,
    #[msg("Invalid land class (must be 1-8)")]
    InvalidLandClass,
    #[msg("Area must be greater than zero")]
    InvalidArea,
    #[msg("Owner does not match parcel owner")]
    OwnerMismatch,
    #[msg("Failed to calculate mint account space")]
    MintSpaceCalculation,
    #[msg("Unauthorized: signer is not mint authority")]
    UnauthorizedAuthority,
    #[msg("Invalid AI score (must be 0-100)")]
    InvalidAIScore,
    #[msg("Invalid collateral grade (must be 0-3)")]
    InvalidCollateralGrade,
}
