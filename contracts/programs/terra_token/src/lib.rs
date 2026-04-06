#![allow(ambiguous_glob_reexports)]
use anchor_lang::prelude::*;

pub mod constants;
pub mod errors;
pub mod events;
pub mod instructions;
pub mod state;

use instructions::*;

declare_id!("2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX");

#[program]
pub mod terra_token {
    use super::*;

    pub fn register_parcel(
        ctx: Context<RegisterParcel>,
        cadastral_number: String,
        area_ha: u32,
        land_class: u8,
        egiss_snapshot_hash: [u8; 32],
    ) -> Result<()> {
        instructions::register_parcel::handler(
            ctx,
            cadastral_number,
            area_ha,
            land_class,
            egiss_snapshot_hash,
        )
    }

    pub fn verify_parcel(
        ctx: Context<VerifyParcel>,
        cadastral_number: String,
    ) -> Result<()> {
        instructions::verify_parcel::handler(ctx, cadastral_number)
    }

    pub fn mint_certificate(
        ctx: Context<MintCertificate>,
        cadastral_number: String,
        season: String,
        ndvi_score: u16,
        crop_type: String,
    ) -> Result<()> {
        instructions::mint_certificate::handler(ctx, cadastral_number, season, ndvi_score, crop_type)
    }

    pub fn seasonal_check(
        ctx: Context<SeasonalCheck>,
        cadastral_number: String,
    ) -> Result<()> {
        instructions::seasonal_check::handler(ctx, cadastral_number)
    }

    pub fn update_risk_assessment(
        ctx: Context<UpdateRiskAssessment>,
        cadastral_number: String,
        ai_score: u8,
        collateral_grade: u8,
        recommended_ltv: u16,
    ) -> Result<()> {
        instructions::update_risk_assessment::handler(
            ctx,
            cadastral_number,
            ai_score,
            collateral_grade,
            recommended_ltv,
        )
    }
}
