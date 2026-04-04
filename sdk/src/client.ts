// TODO: This SDK wraps the REST API as a temporary solution.
// The proper approach is a direct on-chain SDK using @solana/kit that:
// 1. Derives PDAs and reads accounts directly from Solana RPC
// 2. Builds and signs transactions client-side
// 3. Eliminates the backend as a middleman for read operations
// 4. Uses Anchor IDL types for type-safe account deserialization
// See: web/src/solana/program.ts and web/src/solana/accounts.ts for
// the @solana/kit-based approach already used by the frontend.

import type {
  Parcel,
  CreditProfile,
  NDVICertificate,
  Encumbrance,
  RegisterParcelInput,
  RegisterLienInput,
} from "./types";

export class TerraLedgerClient {
  constructor(
    private baseUrl: string,
    private apiKey?: string,
  ) {}

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.apiKey) {
      headers["X-API-Key"] = this.apiKey;
    }

    const res = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    const data = await res.json();

    if (!res.ok) {
      throw new Error(
        (data as { error?: string }).error || `Request failed: ${res.status}`,
      );
    }

    return data as T;
  }

  async getParcel(cadastral: string): Promise<Parcel> {
    return this.request("GET", `/api/v1/parcels/${encodeURIComponent(cadastral)}`);
  }

  async getCreditProfile(cadastral: string): Promise<CreditProfile> {
    return this.request(
      "GET",
      `/api/v1/parcels/${encodeURIComponent(cadastral)}/profile`,
    );
  }

  async registerParcel(input: RegisterParcelInput): Promise<Parcel> {
    return this.request("POST", "/api/v1/parcels", input);
  }

  async listCertificates(cadastral: string): Promise<NDVICertificate[]> {
    return this.request(
      "GET",
      `/api/v1/parcels/${encodeURIComponent(cadastral)}/certificates`,
    );
  }

  async listLiens(cadastral: string): Promise<Encumbrance[]> {
    return this.request(
      "GET",
      `/api/v1/parcels/${encodeURIComponent(cadastral)}/liens`,
    );
  }

  async registerLien(input: RegisterLienInput): Promise<Encumbrance> {
    return this.request("POST", "/api/v1/liens", input);
  }

  async releaseLien(lienId: string): Promise<{ status: string }> {
    return this.request(
      "POST",
      `/api/v1/liens/${encodeURIComponent(lienId)}/release`,
      {},
    );
  }
}
