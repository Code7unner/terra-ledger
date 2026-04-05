use anchor_lang::prelude::*;

#[error_code]
pub enum TransferHookError {
    #[msg("Transfer blocked: parcel is fraud-flagged")]
    FraudFlagged,
    #[msg("Transfer blocked: certificate expired (dormant > 2 seasons)")]
    CertificateExpired,
}
