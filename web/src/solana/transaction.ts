import { useCallback, useState } from 'react'
import { type Instruction, type Address, type Signature } from '@solana/kit'
import { useSendTransaction, useWalletConnection } from '@solana/react-hooks'

// ---------------------------------------------------------------------------
// Anchor Error Map — maps custom error codes to user-friendly messages
// ---------------------------------------------------------------------------

// Anchor error names → user-friendly messages
const ANCHOR_ERROR_NAMES: Record<string, string> = {
  // terra_token
  CadastralTooLong: 'Cadastral number exceeds maximum length',
  ParcelNotVerified: 'Parcel is not KYC verified',
  TooEarlyForSeasonalCheck: 'Too early for seasonal check (must wait 90 days)',
  RiskFlagSet: 'Risk flag is set on this parcel',
  InvalidLandClass: 'Invalid land class (must be 1-8)',
  InvalidArea: 'Area must be greater than zero',
  // lien_registry
  ActiveLienExists: 'Active lien already exists on this parcel (double-pledge prevented)',
  UnauthorizedRelease: 'Only the lender can release this encumbrance',
  EncumbranceNotActive: 'Encumbrance is not active',
  InvalidAmount: 'Lien amount must be greater than zero',
  // Anchor built-in
  AccountNotInitialized: 'Account not found on-chain (parcel may not be registered)',
  ConstraintHasOne: 'Account ownership check failed',
}

function parseTransactionError(err: unknown): string {
  // Recursively search for useful error info in nested error objects
  const raw = extractErrorString(err)

  // 1. Match "Error Code: XYZ. Error Number: N. Error Message: ..."
  const anchorMatch = raw.match(/Error Code: (\w+)\.\s*Error Number: \d+\.\s*Error Message: ([^.]+)/)
  if (anchorMatch) {
    const name = anchorMatch[1]
    return ANCHOR_ERROR_NAMES[name] ?? `${name}: ${anchorMatch[2]}`
  }

  // 2. Match known error names anywhere in the string
  for (const [name, message] of Object.entries(ANCHOR_ERROR_NAMES)) {
    if (raw.includes(name)) {
      return message
    }
  }

  // 3. Match "custom program error: 0xNNNN" with known codes
  const hexMatch = raw.match(/custom program error: 0x([0-9a-fA-F]+)/)
  if (hexMatch) {
    const code = parseInt(hexMatch[1], 16)
    const knownCodes: Record<number, string> = {
      6000: 'Active lien already exists on this parcel (double-pledge prevented)',
      6001: 'Parcel is not KYC verified',
      6002: 'Only the lender can release this encumbrance',
      6003: 'Encumbrance is not active',
      6004: 'Lien amount must be greater than zero',
    }
    return knownCodes[code] ?? `Program error (code ${code})`
  }

  // 4. Common wallet/network errors
  if (raw.includes('already in use')) {
    return 'This parcel is already registered on-chain. Try a different cadastral number.'
  }
  if (raw.includes('User rejected') || raw.includes('user rejected')) {
    return 'Transaction was rejected by the wallet'
  }
  if (raw.includes('insufficient') || raw.includes('Insufficient')) {
    return 'Insufficient SOL balance for transaction fees'
  }
  if (raw.includes('blockhash') || raw.includes('Blockhash')) {
    return 'Transaction expired. Please try again'
  }

  // 5. Truncate long messages
  const msg = err instanceof Error ? err.message : String(err)
  return msg.length > 200 ? msg.slice(0, 200) + '...' : msg
}

/** JSON.stringify that handles BigInt values (common in @solana/kit responses) */
function safeStringify(val: unknown): string {
  try {
    return JSON.stringify(val, (_key, v) => typeof v === 'bigint' ? v.toString() : v)
  } catch {
    return String(val)
  }
}

function extractErrorString(err: unknown): string {
  if (err == null) return ''
  if (typeof err === 'string') return err

  const parts: string[] = []

  if (err instanceof Error) {
    parts.push(err.message)
    if ('cause' in err && err.cause) {
      parts.push(extractErrorString(err.cause))
    }
  }

  if (typeof err === 'object') {
    const obj = err as Record<string, unknown>

    // @solana/errors SolanaError has a `context` property with logs and causeMessage
    if (obj.context && typeof obj.context === 'object') {
      const ctx = obj.context as Record<string, unknown>
      if (ctx.causeMessage && typeof ctx.causeMessage === 'string') {
        parts.push(ctx.causeMessage)
      }
      if (ctx.logs && Array.isArray(ctx.logs)) {
        parts.push((ctx.logs as string[]).join(' '))
      }
      if (ctx.preflightData && typeof ctx.preflightData === 'object') {
        const preflight = ctx.preflightData as Record<string, unknown>
        if (preflight.logs && Array.isArray(preflight.logs)) {
          parts.push((preflight.logs as string[]).join(' '))
        }
      }
      if (ctx.transactionPlanResult) {
        parts.push(safeStringify(ctx.transactionPlanResult))
      }
    }

    if (obj.logs && Array.isArray(obj.logs)) {
      parts.push(obj.logs.join(' '))
    }
    if (obj.transactionPlanResult) {
      parts.push(safeStringify(obj.transactionPlanResult))
    }
    if (obj.data) {
      parts.push(typeof obj.data === 'string' ? obj.data : safeStringify(obj.data))
    }
  }

  return parts.join(' ')
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type OnChainTxStatus =
  | 'idle'
  | 'building'
  | 'signing'
  | 'confirming'
  | 'done'
  | 'error'

export interface SignAndSendResult {
  signature: string
}

// ---------------------------------------------------------------------------
// Hook: useSignAndSend
// ---------------------------------------------------------------------------

/**
 * Hook that provides a `signAndSend` function for submitting on-chain
 * transactions using the connected wallet.
 *
 * Internally uses `useSendTransaction` from `@solana/react-hooks` which
 * handles blockhash fetching, transaction building, wallet signing, and
 * submission through the SolanaProvider context.
 *
 * @example
 * ```tsx
 * const { signAndSend, txStatus, txSignature, txError, reset } = useSignAndSend()
 *
 * async function handleClick() {
 *   const ix = await buildSomeInstruction(...)
 *   await signAndSend([ix])
 * }
 * ```
 */
export function useSignAndSend() {
  const { wallet } = useWalletConnection()
  const { send, isSending } = useSendTransaction()

  const [txStatus, setTxStatus] = useState<OnChainTxStatus>('idle')
  const [txSignature, setTxSignature] = useState<string | null>(null)
  const [txError, setTxError] = useState<string | null>(null)

  const reset = useCallback(() => {
    setTxStatus('idle')
    setTxSignature(null)
    setTxError(null)
  }, [])

  const signAndSend = useCallback(
    async (instructions: Instruction[]): Promise<string> => {
      if (!wallet) {
        const msg = 'Wallet not connected'
        setTxError(msg)
        setTxStatus('error')
        throw new Error(msg)
      }

      try {
        setTxError(null)
        setTxStatus('building')

        const feePayer: Address = wallet.account.address

        setTxStatus('signing')

        // `send` from useSendTransaction handles:
        // - fetching latest blockhash
        // - building the transaction message
        // - signing via the connected wallet
        // - submitting to the cluster
        const signature: Signature = await send(
          { instructions, feePayer },
        )

        setTxStatus('confirming')
        const sig = String(signature)
        setTxSignature(sig)
        setTxStatus('done')

        return sig
      } catch (err: unknown) {
        const message = parseTransactionError(err)
        setTxError(message)
        setTxStatus('error')
        throw new Error(message)
      }
    },
    [wallet, send],
  )

  return {
    signAndSend,
    txStatus,
    txSignature,
    txError,
    isSending,
    reset,
    connected: !!wallet,
    walletAddress: wallet?.account.address ?? null,
  } as const
}
