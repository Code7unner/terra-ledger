import nacl from 'tweetnacl'
import { getBase58Encoder, getBase58Decoder } from '@solana/kit'

const STORAGE_KEY = 'phantom_dapp_keypair'

// @solana/kit codec: encoder = string→bytes, decoder = bytes→string
const b58ToBytes = getBase58Encoder()
const bytesToB58 = getBase58Decoder()

function b58encode(bytes: Uint8Array): string {
  return bytesToB58.decode(bytes)
}

function b58decode(str: string): Uint8Array {
  return new Uint8Array(b58ToBytes.encode(str))
}

export interface DappKeyPair {
  publicKey: Uint8Array
  secretKey: Uint8Array
}

export interface ConnectResponse {
  public_key: string
  session: string
}

/**
 * Generate or load a persistent x25519 keypair for Phantom deeplink encryption.
 */
export function getDappKeyPair(): DappKeyPair {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored) {
    const parsed = JSON.parse(stored)
    return {
      publicKey: new Uint8Array(parsed.publicKey),
      secretKey: new Uint8Array(parsed.secretKey),
    }
  }
  const kp = nacl.box.keyPair()
  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify({
      publicKey: Array.from(kp.publicKey),
      secretKey: Array.from(kp.secretKey),
    }),
  )
  return kp
}

/**
 * Build a Phantom deeplink connect URL.
 */
export function buildConnectUrl(
  dappKeyPair: DappKeyPair,
  sessionId: string,
  cluster: string = 'devnet',
): string {
  const apiBase = import.meta.env.VITE_API_URL || window.location.origin
  const redirectLink = `${apiBase}/api/v1/phantom/callback?session=${sessionId}`

  const params = new URLSearchParams({
    dapp_encryption_public_key: b58encode(dappKeyPair.publicKey),
    cluster,
    app_url: window.location.origin,
    redirect_link: redirectLink,
  })

  return `https://phantom.app/ul/v1/connect?${params.toString()}`
}

/**
 * Decrypt the Phantom connect response using x25519 Diffie-Hellman.
 */
/**
 * Build a Phantom deeplink signAndSendTransaction URL.
 */
/**
 * Build a Phantom deeplink signTransaction URL.
 * Phantom signs the transaction and returns it via callback (does NOT submit).
 */
export function buildSignTransactionUrl(
  dappKeyPair: DappKeyPair,
  sessionToken: string,
  phantomPubKeyB58: string,
  transactionB58: string,
  sessionId: string,
): string {
  const apiBase = import.meta.env.VITE_API_URL || window.location.origin
  const redirectLink = `${apiBase}/api/v1/phantom/callback?session=${sessionId}`

  const phantomPubKey = b58decode(phantomPubKeyB58)
  const sharedSecret = nacl.box.before(phantomPubKey, dappKeyPair.secretKey)

  const payload = JSON.stringify({
    transaction: transactionB58,
    session: sessionToken,
  })

  const nonce = nacl.randomBytes(24)
  const encrypted = nacl.box.after(
    new TextEncoder().encode(payload),
    nonce,
    sharedSecret,
  )

  const params = new URLSearchParams({
    dapp_encryption_public_key: b58encode(dappKeyPair.publicKey),
    nonce: b58encode(nonce),
    redirect_link: redirectLink,
    payload: b58encode(encrypted),
  })

  return `https://phantom.app/ul/v1/signTransaction?${params.toString()}`
}

/**
 * Decrypt the Phantom signTransaction response.
 * Returns the signed transaction as base58.
 */
export function decryptSignResponse(
  phantomPubKeyB58: string,
  nonceB58: string,
  dataB58: string,
  dappKeyPair: DappKeyPair,
): { transaction: string } {
  const phantomPubKey = b58decode(phantomPubKeyB58)
  const nonce = b58decode(nonceB58)
  const encryptedData = b58decode(dataB58)

  const sharedSecret = nacl.box.before(phantomPubKey, dappKeyPair.secretKey)

  const decrypted = nacl.box.open.after(encryptedData, nonce, sharedSecret)
  if (!decrypted) {
    throw new Error('Failed to decrypt Phantom sign response')
  }

  const json = JSON.parse(new TextDecoder().decode(decrypted))
  return { transaction: json.transaction }
}

export function decryptConnectResponse(
  phantomPubKeyB58: string,
  nonceB58: string,
  dataB58: string,
  dappKeyPair: DappKeyPair,
): ConnectResponse {
  const phantomPubKey = b58decode(phantomPubKeyB58)
  const nonce = b58decode(nonceB58)
  const encryptedData = b58decode(dataB58)

  const sharedSecret = nacl.box.before(phantomPubKey, dappKeyPair.secretKey)

  const decrypted = nacl.box.open.after(encryptedData, nonce, sharedSecret)
  if (!decrypted) {
    throw new Error('Failed to decrypt Phantom response')
  }

  const json = JSON.parse(new TextDecoder().decode(decrypted))
  return {
    public_key: json.public_key,
    session: json.session,
  }
}
