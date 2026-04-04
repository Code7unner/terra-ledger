import { useCallback, useState } from 'react'
import { type Instruction, type Address, type Signature } from '@solana/kit'
import { useSendTransaction, useWalletConnection } from '@solana/react-hooks'

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
        const message =
          err instanceof Error ? err.message : 'Transaction failed'
        setTxError(message)
        setTxStatus('error')
        throw err
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
