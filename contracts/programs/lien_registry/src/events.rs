use anchor_lang::prelude::*;

#[event]
pub struct EncumbranceRegistered {
    pub cadastral_number: String,
    pub lender: Pubkey,
    pub amount: u64,
    pub parcel_pda: Pubkey,
}

#[event]
pub struct EncumbranceReleased {
    pub cadastral_number: String,
    pub lender: Pubkey,
    pub parcel_pda: Pubkey,
}
