use anchor_lang::prelude::*;

#[account]
#[derive(InitSpace)]
pub struct ParcelConfig {
    pub owner: Pubkey,
    pub mint_authority: Pubkey,
    #[max_len(20)]
    pub cadastral_str: String,
    pub area_ha: u32,
    pub land_class: u8,
    pub kyc_verified: bool,
    pub kyc_method: u8,
    pub last_cert_epoch: u64,
    pub cert_count: u16,
    pub ndvi_submissions_this_season: u8,
    pub dormant_seasons: u8,
    pub egiss_snapshot_hash: [u8; 32],
    pub risk_flag: u8,
    pub registered_at: i64,
    pub bump: u8,
    pub ai_score: u8,           // 0-100 AI credit score
    pub collateral_grade: u8,   // 0=A, 1=B, 2=C, 3=D
    pub recommended_ltv: u16,   // basis points: 7000 = 70.00%
}
