use anchor_lang::prelude::*;

#[account]
#[derive(InitSpace)]
pub struct EncumbranceAccount {
    pub parcel_pda: Pubkey,
    pub lender: Pubkey,
    pub amount: u64,
    pub notary_sig_hash: [u8; 32],
    pub notary_cert_hash: [u8; 32],
    pub egiss_snapshot_hash: [u8; 32],
    pub registered_at: i64,
    pub released_at: i64,
    pub status: u8, // 0=Active, 1=Released, 2=Disputed
    pub bump: u8,
}

impl EncumbranceAccount {
    pub const STATUS_ACTIVE: u8 = 0;
    pub const STATUS_RELEASED: u8 = 1;
}
