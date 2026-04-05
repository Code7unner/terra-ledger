import { type Address, getAddressDecoder } from '@solana/kit'
import {
  TERRA_TOKEN_PROGRAM_ID,
  LIEN_REGISTRY_PROGRAM_ID,
  getParcelPda,
  getEncumbrancePda,
  getLienIndexPda,
} from './program'

// ---------------------------------------------------------------------------
// Deserialization helpers
// ---------------------------------------------------------------------------

const addressDecoder = getAddressDecoder()

const ANCHOR_DISCRIMINATOR_SIZE = 8

function readAddress(data: Uint8Array, offset: number): Address {
  return addressDecoder.decode(data.slice(offset, offset + 32)) as Address
}

function readU8(data: Uint8Array, offset: number): number {
  return data[offset]
}

function readU16LE(data: Uint8Array, offset: number): number {
  return new DataView(data.buffer, data.byteOffset).getUint16(offset, true)
}

function readU32LE(data: Uint8Array, offset: number): number {
  return new DataView(data.buffer, data.byteOffset).getUint32(offset, true)
}

function readU64LE(data: Uint8Array, offset: number): bigint {
  return new DataView(data.buffer, data.byteOffset).getBigUint64(offset, true)
}

function readI64LE(data: Uint8Array, offset: number): bigint {
  return new DataView(data.buffer, data.byteOffset).getBigInt64(offset, true)
}

function readBool(data: Uint8Array, offset: number): boolean {
  return data[offset] !== 0
}

function readBytes(data: Uint8Array, offset: number, len: number): Uint8Array {
  return data.slice(offset, offset + len)
}

/**
 * Read a Borsh-encoded string (4-byte LE length prefix + UTF-8 bytes).
 * Returns [value, bytesConsumed].
 */
function readString(data: Uint8Array, offset: number): [string, number] {
  const len = readU32LE(data, offset)
  const strBytes = data.slice(offset + 4, offset + 4 + len)
  const value = new TextDecoder().decode(strBytes)
  return [value, 4 + len]
}

// ---------------------------------------------------------------------------
// Base64 helpers
// ---------------------------------------------------------------------------

function bytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

function base64ToBytes(base64: string): Uint8Array {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes
}

// ---------------------------------------------------------------------------
// RPC helper
// ---------------------------------------------------------------------------

/**
 * Fetch raw account data from an RPC endpoint.
 */
async function getAccountData(
  rpcUrl: string,
  addr: Address,
): Promise<Uint8Array | null> {
  const resp = await fetch(rpcUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'getAccountInfo',
      params: [addr, { encoding: 'base64' }],
    }),
  })
  const json = (await resp.json()) as {
    result: { value: { data: [string, string] } | null }
  }
  if (!json.result.value) return null
  return base64ToBytes(json.result.value.data[0])
}

// ---------------------------------------------------------------------------
// ParcelConfig account
// ---------------------------------------------------------------------------

export interface ParcelConfigAccount {
  owner: Address
  mintAuthority: Address
  cadastralStr: string
  areaHa: number
  landClass: number
  kycVerified: boolean
  kycMethod: number
  lastCertEpoch: bigint
  certCount: number
  ndviSubmissionsThisSeason: number
  dormantSeasons: number
  egissSnapshotHash: Uint8Array
  riskFlag: number
  registeredAt: bigint
  bump: number
}

function deserializeParcelConfig(data: Uint8Array): ParcelConfigAccount {
  let off = ANCHOR_DISCRIMINATOR_SIZE

  const owner = readAddress(data, off); off += 32
  const mintAuthority = readAddress(data, off); off += 32

  const [cadastralStr, strSize] = readString(data, off); off += strSize

  const areaHa = readU32LE(data, off); off += 4
  const landClass = readU8(data, off); off += 1
  const kycVerified = readBool(data, off); off += 1
  const kycMethod = readU8(data, off); off += 1
  const lastCertEpoch = readU64LE(data, off); off += 8
  const certCount = readU16LE(data, off); off += 2
  const ndviSubmissionsThisSeason = readU8(data, off); off += 1
  const dormantSeasons = readU8(data, off); off += 1
  const egissSnapshotHash = readBytes(data, off, 32); off += 32
  const riskFlag = readU8(data, off); off += 1
  const registeredAt = readI64LE(data, off); off += 8
  const bump = readU8(data, off)

  return {
    owner,
    mintAuthority,
    cadastralStr,
    areaHa,
    landClass,
    kycVerified,
    kycMethod,
    lastCertEpoch,
    certCount,
    ndviSubmissionsThisSeason,
    dormantSeasons,
    egissSnapshotHash,
    riskFlag,
    registeredAt,
    bump,
  }
}

// ---------------------------------------------------------------------------
// EncumbranceAccount
// ---------------------------------------------------------------------------

export type EncumbranceStatus = 'Active' | 'Released' | 'Disputed'

export interface EncumbranceAccountData {
  parcelPda: Address
  lender: Address
  amount: bigint
  notarySigHash: Uint8Array
  notaryCertHash: Uint8Array
  egissSnapshotHash: Uint8Array
  registeredAt: bigint
  releasedAt: bigint
  status: EncumbranceStatus
  bump: number
}

const ENCUMBRANCE_STATUS_MAP: Record<number, EncumbranceStatus> = {
  0: 'Active',
  1: 'Released',
  2: 'Disputed',
}

function deserializeEncumbrance(data: Uint8Array): EncumbranceAccountData {
  let off = ANCHOR_DISCRIMINATOR_SIZE

  const parcelPda = readAddress(data, off); off += 32
  const lender = readAddress(data, off); off += 32
  const amount = readU64LE(data, off); off += 8
  const notarySigHash = readBytes(data, off, 32); off += 32
  const notaryCertHash = readBytes(data, off, 32); off += 32
  const egissSnapshotHash = readBytes(data, off, 32); off += 32
  const registeredAt = readI64LE(data, off); off += 8
  const releasedAt = readI64LE(data, off); off += 8
  const status = ENCUMBRANCE_STATUS_MAP[readU8(data, off)] ?? 'Active'; off += 1
  const bump = readU8(data, off)

  return {
    parcelPda,
    lender,
    amount,
    notarySigHash,
    notaryCertHash,
    egissSnapshotHash,
    registeredAt,
    releasedAt,
    status,
    bump,
  }
}

// ---------------------------------------------------------------------------
// LienIndex account
// ---------------------------------------------------------------------------

export interface LienIndexData {
  parcelPda: Address
  activeLienCount: number
  totalLienCount: number
  bump: number
}

function deserializeLienIndex(data: Uint8Array): LienIndexData {
  let off = ANCHOR_DISCRIMINATOR_SIZE

  const parcelPda = readAddress(data, off); off += 32
  const activeLienCount = readU8(data, off); off += 1
  const totalLienCount = readU16LE(data, off); off += 2
  const bump = readU8(data, off)

  return { parcelPda, activeLienCount, totalLienCount, bump }
}

// ---------------------------------------------------------------------------
// Public fetch functions
// ---------------------------------------------------------------------------

/**
 * Fetch a ParcelConfig account by cadastral number.
 */
export async function fetchParcelConfig(
  rpcUrl: string,
  cadastralNumber: string,
): Promise<ParcelConfigAccount | null> {
  const [pda] = await getParcelPda(cadastralNumber)
  const data = await getAccountData(rpcUrl, pda)
  if (!data) return null
  return deserializeParcelConfig(data)
}

/**
 * Fetch a ParcelConfig account by its PDA address directly.
 */
export async function fetchParcelConfigByPda(
  rpcUrl: string,
  parcelPda: Address,
): Promise<ParcelConfigAccount | null> {
  const data = await getAccountData(rpcUrl, parcelPda)
  if (!data) return null
  return deserializeParcelConfig(data)
}

/**
 * Fetch an EncumbranceAccount for a given parcel and lender.
 */
export async function fetchEncumbrance(
  rpcUrl: string,
  parcelPda: Address,
  lender: Address,
): Promise<EncumbranceAccountData | null> {
  const [pda] = await getEncumbrancePda(parcelPda, lender)
  const data = await getAccountData(rpcUrl, pda)
  if (!data) return null
  return deserializeEncumbrance(data)
}

/**
 * Fetch the LienIndex for a given parcel.
 */
export async function fetchLienIndex(
  rpcUrl: string,
  parcelPda: Address,
): Promise<LienIndexData | null> {
  const [pda] = await getLienIndexPda(parcelPda)
  const data = await getAccountData(rpcUrl, pda)
  if (!data) return null
  return deserializeLienIndex(data)
}

/**
 * Fetch all ParcelConfig accounts owned by the terra_token program.
 * Uses getProgramAccounts with discriminator filter.
 */
export async function fetchAllParcels(
  rpcUrl: string,
): Promise<{ pda: string; account: ParcelConfigAccount }[]> {
  const discriminator = bytesToBase64(new Uint8Array([13, 4, 49, 115, 238, 91, 158, 26]))

  const resp = await fetch(rpcUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'getProgramAccounts',
      params: [
        TERRA_TOKEN_PROGRAM_ID,
        {
          encoding: 'base64',
          filters: [
            {
              memcmp: {
                offset: 0,
                bytes: discriminator,
                encoding: 'base64',
              },
            },
          ],
        },
      ],
    }),
  })
  const json = (await resp.json()) as {
    result: Array<{
      pubkey: string
      account: { data: [string, string] }
    }> | null
  }

  if (!json.result) return []

  return json.result.map((entry) => ({
    pda: entry.pubkey,
    account: deserializeParcelConfig(base64ToBytes(entry.account.data[0])),
  }))
}

/**
 * Fetch all EncumbranceAccount entries from the lien_registry program.
 */
export async function fetchAllEncumbrances(
  rpcUrl: string,
): Promise<{ pda: string; account: EncumbranceAccountData }[]> {
  const discriminator = bytesToBase64(new Uint8Array([81, 214, 219, 18, 223, 107, 108, 14]))

  const resp = await fetch(rpcUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'getProgramAccounts',
      params: [
        LIEN_REGISTRY_PROGRAM_ID,
        {
          encoding: 'base64',
          filters: [
            {
              memcmp: {
                offset: 0,
                bytes: discriminator,
                encoding: 'base64',
              },
            },
          ],
        },
      ],
    }),
  })
  const json = (await resp.json()) as {
    result: Array<{
      pubkey: string
      account: { data: [string, string] }
    }> | null
  }

  if (!json.result) return []

  return json.result.map((entry) => ({
    pda: entry.pubkey,
    account: deserializeEncumbrance(base64ToBytes(entry.account.data[0])),
  }))
}

/**
 * Fetch encumbrances for a specific parcel by filtering on the parcel_pda field
 * (offset 8, first 32 bytes after discriminator).
 */
export async function fetchEncumbrancesByParcel(
  rpcUrl: string,
  parcelPda: Address,
): Promise<EncumbranceAccountData[]> {
  const resp = await fetch(rpcUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'getProgramAccounts',
      params: [
        LIEN_REGISTRY_PROGRAM_ID,
        {
          encoding: 'base64',
          filters: [
            {
              memcmp: {
                offset: ANCHOR_DISCRIMINATOR_SIZE,
                bytes: parcelPda,
              },
            },
          ],
        },
      ],
    }),
  })
  const json = (await resp.json()) as {
    result: Array<{
      pubkey: string
      account: { data: [string, string] }
    }> | null
  }

  if (!json.result) return []

  return json.result.map((entry) =>
    deserializeEncumbrance(base64ToBytes(entry.account.data[0])),
  )
}

/**
 * Check if an account exists on-chain.
 */
export async function accountExists(
  rpcUrl: string,
  addr: Address,
): Promise<boolean> {
  const resp = await fetch(rpcUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'getAccountInfo',
      params: [addr, { encoding: 'base64' }],
    }),
  })
  const json = (await resp.json()) as { result: { value: unknown } }
  return json.result.value !== null
}
