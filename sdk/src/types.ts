export interface Parcel {
  id: string;
  cadastral_number: string;
  owner_wallet: string;
  on_chain_address?: string;
  area_ha: number;
  land_class: number;
  kyc_verified: boolean;
  oblast?: string;
  rayon?: string;
  holder_name?: string;
  holder_iin_hash?: string;
  registered_at: string;
  updated_at: string;
}

export interface NDVICertificate {
  id: string;
  parcel_id: string;
  cadastral_number: string;
  season: string;
  ndvi_score: number;
  crop_type?: string;
  yield_t_ha?: number;
  sentinel_scene_id?: string;
  on_chain_address?: string;
  tx_signature?: string;
  minted_at: string;
}

export interface Encumbrance {
  id: string;
  parcel_id: string;
  cadastral_number: string;
  lender_wallet: string;
  lender_name?: string;
  amount_tenge: number;
  notary_cert_hash?: string;
  on_chain_address?: string;
  tx_signature?: string;
  status: "active" | "released" | "disputed";
  registered_at: string;
  released_at?: string;
}

export interface CreditScore {
  id: string;
  parcel_id: string;
  cadastral_number: string;
  ai_score: number;
  recommended_ltv: number;
  collateral_grade: "A" | "B" | "C" | "D";
  estimated_value_tenge: number;
  model_version: string;
  explanation: string;
  risk_factors: string[];
  computed_at: string;
}

export interface ProductivityData {
  certificates: NDVICertificate[];
  ndvi_trend: string;
  dormancy_risk: string;
}

export interface EncumbranceData {
  active_liens: Encumbrance[];
  lien_count_historical: number;
  double_pledge_risk: boolean;
}

export interface CreditProfile {
  parcel: Parcel;
  productivity: ProductivityData;
  encumbrances: EncumbranceData;
  credit_intelligence?: CreditScore;
}

export interface RegisterParcelInput {
  cadastral_number: string;
  owner_wallet: string;
  area_ha: number;
  land_class: number;
  oblast: string;
  rayon: string;
  holder_name: string;
  holder_iin_hash: string;
}

export interface RegisterLienInput {
  cadastral_number: string;
  lender_wallet: string;
  amount_tenge: number;
  notary_cert_hash: string;
}
