import {
  type Address,
  address,
  AccountRole,
  type Instruction,
  getProgramDerivedAddress,
  getAddressEncoder,
  getBase58Decoder,
  getBase58Encoder,
} from '@solana/kit'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const TERRA_TOKEN_PROGRAM_ID: Address = address(
  '2eAqpJ7yjso7FDA4sDQLJQioNCRuoYSUeha2Y88NRRMX',
)
export const LIEN_REGISTRY_PROGRAM_ID: Address = address(
  '3qYHSTPeRLRDfWmtzEhiaHpT2kchgW8GqaYcwmDbKnq4',
)
export const SYSTEM_PROGRAM: Address = address(
  '11111111111111111111111111111111',
)

const RPC_URL =
  import.meta.env.VITE_SOLANA_RPC_URL || 'https://api.devnet.solana.com'

// ---------------------------------------------------------------------------
// Discriminators (from IDL)
// ---------------------------------------------------------------------------

const REGISTER_PARCEL_DISCRIMINATOR = new Uint8Array([170, 232, 221, 44, 109, 149, 104, 207])
const MINT_CERTIFICATE_DISCRIMINATOR = new Uint8Array([53, 2, 104, 84, 51, 197, 179, 10])
const SEASONAL_CHECK_DISCRIMINATOR = new Uint8Array([161, 143, 168, 146, 218, 14, 59, 80])
const VERIFY_PARCEL_DISCRIMINATOR = new Uint8Array([45, 47, 217, 191, 140, 31, 95, 178])
const REGISTER_ENCUMBRANCE_DISCRIMINATOR = new Uint8Array([5, 82, 66, 155, 31, 178, 204, 252])
const RELEASE_ENCUMBRANCE_DISCRIMINATOR = new Uint8Array([144, 232, 132, 141, 47, 237, 85, 194])

// ---------------------------------------------------------------------------
// PDA derivation helpers
// ---------------------------------------------------------------------------

const addressEncoder = getAddressEncoder()

/**
 * Derive parcel config PDA from cadastral number.
 * Seeds: ["parcel", cadastral_bytes]
 */
export async function getParcelPda(
  cadastralNumber: string,
): Promise<[Address, number]> {
  const cadastralBytes = new TextEncoder().encode(cadastralNumber)
  const [pda, bump] = await getProgramDerivedAddress({
    programAddress: TERRA_TOKEN_PROGRAM_ID,
    seeds: ['parcel', cadastralBytes],
  })
  return [pda, bump]
}

/**
 * Derive encumbrance PDA.
 * Seeds: ["encumbrance", parcel_pda, lender]
 */
export async function getEncumbrancePda(
  parcelPda: Address,
  lender: Address,
): Promise<[Address, number]> {
  const [pda, bump] = await getProgramDerivedAddress({
    programAddress: LIEN_REGISTRY_PROGRAM_ID,
    seeds: [
      'encumbrance',
      addressEncoder.encode(parcelPda),
      addressEncoder.encode(lender),
    ],
  })
  return [pda, bump]
}

/**
 * Derive lien index PDA.
 * Seeds: ["lien_index", parcel_pda]
 */
export async function getLienIndexPda(
  parcelPda: Address,
): Promise<[Address, number]> {
  const [pda, bump] = await getProgramDerivedAddress({
    programAddress: LIEN_REGISTRY_PROGRAM_ID,
    seeds: ['lien_index', addressEncoder.encode(parcelPda)],
  })
  return [pda, bump]
}

// ---------------------------------------------------------------------------
// Encoding helpers
// ---------------------------------------------------------------------------

function encodeU64LE(value: bigint): Uint8Array {
  const buf = new Uint8Array(8)
  const view = new DataView(buf.buffer)
  view.setBigUint64(0, value, true)
  return buf
}

function encodeU32LE(value: number): Uint8Array {
  const buf = new Uint8Array(4)
  new DataView(buf.buffer).setUint32(0, value, true)
  return buf
}

function encodeU16LE(value: number): Uint8Array {
  const buf = new Uint8Array(2)
  new DataView(buf.buffer).setUint16(0, value, true)
  return buf
}

function encodeString(value: string): Uint8Array {
  const strBytes = new TextEncoder().encode(value)
  const buf = new Uint8Array(4 + strBytes.length)
  new DataView(buf.buffer).setUint32(0, strBytes.length, true)
  buf.set(strBytes, 4)
  return buf
}

function concatBytes(...arrays: Uint8Array[]): Uint8Array {
  const total = arrays.reduce((s, a) => s + a.length, 0)
  const out = new Uint8Array(total)
  let offset = 0
  for (const a of arrays) {
    out.set(a, offset)
    offset += a.length
  }
  return out
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

/**
 * Check if an account exists on-chain.
 */
export async function accountExists(addr: Address): Promise<boolean> {
  const resp = await fetch(RPC_URL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'getAccountInfo',
      params: [addr, { encoding: 'base64' }],
    }),
  })
  const json = await resp.json() as { result: { value: unknown } }
  return json.result.value !== null
}

// ---------------------------------------------------------------------------
// Instruction builders — terra_token
// ---------------------------------------------------------------------------

/**
 * Build `register_parcel` instruction.
 */
export async function buildRegisterParcelInstruction(
  owner: Address,
  cadastralNumber: string,
  areaHa: number,
  landClass: number,
  egissSnapshotHash: Uint8Array,
): Promise<Instruction> {
  const [parcelPda] = await getParcelPda(cadastralNumber)

  const data = concatBytes(
    REGISTER_PARCEL_DISCRIMINATOR,
    encodeString(cadastralNumber),
    encodeU32LE(areaHa),
    new Uint8Array([landClass]),
    egissSnapshotHash,
  )

  return {
    programAddress: TERRA_TOKEN_PROGRAM_ID,
    accounts: [
      { address: parcelPda, role: AccountRole.WRITABLE },
      { address: owner, role: AccountRole.WRITABLE_SIGNER },
      { address: SYSTEM_PROGRAM, role: AccountRole.READONLY },
    ],
    data,
  }
}

/**
 * Build `mint_certificate` instruction.
 */
export async function buildMintCertificateInstruction(
  mintAuthority: Address,
  cadastralNumber: string,
  season: string,
  ndviScore: number,
): Promise<Instruction> {
  const [parcelPda] = await getParcelPda(cadastralNumber)

  const data = concatBytes(
    MINT_CERTIFICATE_DISCRIMINATOR,
    encodeString(cadastralNumber),
    encodeString(season),
    encodeU16LE(ndviScore),
  )

  return {
    programAddress: TERRA_TOKEN_PROGRAM_ID,
    accounts: [
      { address: parcelPda, role: AccountRole.WRITABLE },
      { address: mintAuthority, role: AccountRole.READONLY_SIGNER },
    ],
    data,
  }
}

/**
 * Build `seasonal_check` instruction.
 */
export async function buildSeasonalCheckInstruction(
  keeper: Address,
  cadastralNumber: string,
): Promise<Instruction> {
  const [parcelPda] = await getParcelPda(cadastralNumber)

  const data = concatBytes(
    SEASONAL_CHECK_DISCRIMINATOR,
    encodeString(cadastralNumber),
  )

  return {
    programAddress: TERRA_TOKEN_PROGRAM_ID,
    accounts: [
      { address: parcelPda, role: AccountRole.WRITABLE },
      { address: keeper, role: AccountRole.READONLY_SIGNER },
    ],
    data,
  }
}

/**
 * Build `verify_parcel` instruction.
 */
export async function buildVerifyParcelInstruction(
  cadastralNumber: string,
): Promise<Instruction> {
  const [parcelPda] = await getParcelPda(cadastralNumber)

  const data = concatBytes(
    VERIFY_PARCEL_DISCRIMINATOR,
    encodeString(cadastralNumber),
  )

  return {
    programAddress: TERRA_TOKEN_PROGRAM_ID,
    accounts: [
      { address: parcelPda, role: AccountRole.READONLY },
    ],
    data,
  }
}

// ---------------------------------------------------------------------------
// Instruction builders — lien_registry
// ---------------------------------------------------------------------------

/**
 * Build `register_encumbrance` instruction.
 */
export async function buildRegisterEncumbranceInstruction(
  lender: Address,
  parcelPda: Address,
  cadastralNumber: string,
  amount: bigint,
  notarySigHash: Uint8Array,
  notaryCertHash: Uint8Array,
): Promise<Instruction> {
  const [encumbrancePda] = await getEncumbrancePda(parcelPda, lender)
  const [lienIndexPda] = await getLienIndexPda(parcelPda)

  const data = concatBytes(
    REGISTER_ENCUMBRANCE_DISCRIMINATOR,
    encodeString(cadastralNumber),
    encodeU64LE(amount),
    notarySigHash,
    notaryCertHash,
  )

  return {
    programAddress: LIEN_REGISTRY_PROGRAM_ID,
    accounts: [
      { address: encumbrancePda, role: AccountRole.WRITABLE },
      { address: lienIndexPda, role: AccountRole.WRITABLE },
      { address: parcelPda, role: AccountRole.READONLY },
      { address: lender, role: AccountRole.WRITABLE_SIGNER },
      { address: SYSTEM_PROGRAM, role: AccountRole.READONLY },
      { address: TERRA_TOKEN_PROGRAM_ID, role: AccountRole.READONLY },
    ],
    data,
  }
}

/**
 * Build `release_encumbrance` instruction.
 */
export async function buildReleaseEncumbranceInstruction(
  lender: Address,
  parcelPda: Address,
): Promise<Instruction> {
  const [encumbrancePda] = await getEncumbrancePda(parcelPda, lender)
  const [lienIndexPda] = await getLienIndexPda(parcelPda)

  const data = concatBytes(RELEASE_ENCUMBRANCE_DISCRIMINATOR)

  return {
    programAddress: LIEN_REGISTRY_PROGRAM_ID,
    accounts: [
      { address: encumbrancePda, role: AccountRole.WRITABLE },
      { address: lienIndexPda, role: AccountRole.WRITABLE },
      { address: parcelPda, role: AccountRole.READONLY },
      { address: lender, role: AccountRole.WRITABLE_SIGNER },
    ],
    data,
  }
}

// ---------------------------------------------------------------------------
// Transaction serialization for Phantom deeplink signing
// ---------------------------------------------------------------------------

const b58Decoder = getBase58Decoder()

/**
 * Build a fully serialized (unsigned) transaction as base58 string.
 * Uses @solana/web3.js for Phantom deeplink compatibility.
 */
export async function serializeTransactionB58(
  instructions: Instruction[],
  feePayer: Address,
): Promise<string> {
  // Use @solana/web3.js for serialization — Phantom deeplink expects this format
  const { Connection, Transaction, PublicKey, TransactionInstruction } = await import('@solana/web3.js')
  const connection = new Connection(RPC_URL, 'confirmed')
  const { blockhash } = await connection.getLatestBlockhash('confirmed')

  const tx = new Transaction()
  tx.recentBlockhash = blockhash
  tx.feePayer = new PublicKey(feePayer)

  for (const ix of instructions) {
    tx.add(new TransactionInstruction({
      programId: new PublicKey(ix.programAddress),
      keys: (ix.accounts ?? []).map((a) => ({
        pubkey: new PublicKey(a.address),
        isSigner: a.role === AccountRole.WRITABLE_SIGNER || a.role === AccountRole.READONLY_SIGNER,
        isWritable: a.role === AccountRole.WRITABLE || a.role === AccountRole.WRITABLE_SIGNER,
      })),
      data: Buffer.from(ix.data ?? new Uint8Array()),
    }))
  }

  const serialized = tx.serialize({ requireAllSignatures: false })
  return bs58Encode(serialized)
}

function bs58Encode(bytes: Uint8Array): string {
  return b58Decoder.decode(bytes)
}

/**
 * Submit a signed transaction (base58) to the cluster and return the signature.
 */
export async function submitSignedTransaction(signedTxB58: string): Promise<string> {
  const { Connection } = await import('@solana/web3.js')
  const connection = new Connection(RPC_URL, 'confirmed')

  const txBytes = Buffer.from(getBase58Encoder().encode(signedTxB58))
  const signature = await connection.sendRawTransaction(txBytes, {
    preflightCommitment: 'confirmed',
  })

  return signature
}
