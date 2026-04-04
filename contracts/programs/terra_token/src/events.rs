use anchor_lang::prelude::*;

#[event]
pub struct ParcelRegistered {
    pub cadastral_number: String,
    pub owner: Pubkey,
    pub area_ha: u32,
}

#[event]
pub struct CertificateMinted {
    pub cadastral_number: String,
    pub season: String,
    pub ndvi_score: u16,
    pub cert_address: Pubkey,
}

#[event]
pub struct ParcelDormant {
    pub cadastral_number: String,
    pub seasons_dormant: u8,
}
