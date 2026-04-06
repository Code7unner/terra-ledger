use anchor_lang::prelude::*;

use crate::constants::PARCEL_SEED;
use crate::errors::TerraTokenError;
use crate::events::RiskAssessmentUpdated;
use crate::state::ParcelConfig;

#[derive(Accounts)]
#[instruction(cadastral_number: String)]
pub struct UpdateRiskAssessment<'info> {
    #[account(
        mut,
        seeds = [PARCEL_SEED, cadastral_number.as_bytes()],
        bump = parcel_config.bump,
    )]
    pub parcel_config: Account<'info, ParcelConfig>,

    #[account(
        constraint = authority.key() == parcel_config.mint_authority @ TerraTokenError::UnauthorizedAuthority
    )]
    pub authority: Signer<'info>,
}

pub fn handler(
    ctx: Context<UpdateRiskAssessment>,
    cadastral_number: String,
    ai_score: u8,
    collateral_grade: u8,
    recommended_ltv: u16,
) -> Result<()> {
    require!(ai_score <= 100, TerraTokenError::InvalidAIScore);
    require!(collateral_grade <= 3, TerraTokenError::InvalidCollateralGrade);

    let parcel = &mut ctx.accounts.parcel_config;
    parcel.ai_score = ai_score;
    parcel.collateral_grade = collateral_grade;
    parcel.recommended_ltv = recommended_ltv;

    if ai_score < 20 && parcel.risk_flag == 0 {
        parcel.risk_flag = 1;
    }
    if ai_score >= 40 && parcel.risk_flag == 1 {
        parcel.risk_flag = 0;
    }

    emit!(RiskAssessmentUpdated {
        cadastral_number,
        ai_score,
        collateral_grade,
        recommended_ltv,
        risk_flag: parcel.risk_flag,
    });

    Ok(())
}
