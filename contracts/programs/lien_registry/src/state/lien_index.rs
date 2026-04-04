use anchor_lang::prelude::*;

#[account]
#[derive(InitSpace)]
pub struct LienIndex {
    pub parcel_pda: Pubkey,
    pub active_lien_count: u8,
    pub total_lien_count: u16,
    pub bump: u8,
}
