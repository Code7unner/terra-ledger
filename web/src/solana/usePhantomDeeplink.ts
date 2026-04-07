import { useState, useCallback, useRef, useEffect } from 'react'
import { post, get } from '../api/client'
import {
  getDappKeyPair,
  buildConnectUrl,
  decryptConnectResponse,
  buildSignTransactionUrl,
  decryptSignResponse,
} from './phantom-deeplink'
import { submitSignedTransaction } from './program'

type DeeplinkStatus = 'idle' | 'waiting' | 'connected' | 'error'

const POLL_INTERVAL = 2000
const MAX_POLL_TIME = 5 * 60 * 1000 // 5 minutes

interface SessionResponse {
  session_id: string
}

interface PollResponse {
  status: 'pending' | 'connected' | 'error' | 'expired'
  phantom_encryption_public_key?: string
  nonce?: string
  data?: string
  error_code?: string
  error_message?: string
}

export function usePhantomDeeplink() {
  // Restore from localStorage if previously connected
  const storedWallet = typeof window !== 'undefined' ? localStorage.getItem('phantom_deeplink_wallet') : null
  const [status, setStatus] = useState<DeeplinkStatus>(storedWallet ? 'connected' : 'idle')
  const [qrUrl, setQrUrl] = useState<string | null>(null)
  const [walletAddress, setWalletAddress] = useState<string | null>(storedWallet)
  const [error, setError] = useState<string | null>(null)
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const startTimeRef = useRef<number>(0)

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current)
      pollingRef.current = null
    }
  }, [])

  // Cleanup on unmount
  useEffect(() => {
    return () => stopPolling()
  }, [stopPolling])

  const connectViaQR = useCallback(async () => {
    setError(null)
    setWalletAddress(null)
    setQrUrl(null)
    stopPolling()

    try {
      // 1. Create relay session on backend
      const { session_id } = await post<SessionResponse>('/api/v1/phantom/session', {})

      // 2. Generate deeplink URL
      const dappKeyPair = getDappKeyPair()
      const url = buildConnectUrl(dappKeyPair, session_id)
      setQrUrl(url)
      setStatus('waiting')

      // 3. Start polling
      startTimeRef.current = Date.now()
      pollingRef.current = setInterval(async () => {
        // Timeout check
        if (Date.now() - startTimeRef.current > MAX_POLL_TIME) {
          stopPolling()
          setStatus('error')
          setError('Connection timed out. Please try again.')
          return
        }

        try {
          const resp = await get<PollResponse>(`/api/v1/phantom/poll/${session_id}`)

          if (resp.status === 'pending') return

          stopPolling()

          if (resp.status === 'connected' && resp.phantom_encryption_public_key && resp.nonce && resp.data) {
            try {
              const result = decryptConnectResponse(
                resp.phantom_encryption_public_key,
                resp.nonce,
                resp.data,
                dappKeyPair,
              )
              setWalletAddress(result.public_key)
              setStatus('connected')
              localStorage.setItem('phantom_deeplink_wallet', result.public_key)
              localStorage.setItem('phantom_deeplink_session', result.session)
              localStorage.setItem('phantom_deeplink_pubkey', resp.phantom_encryption_public_key!)
            } catch (decryptErr) {
              void decryptErr
              setError('Failed to decrypt wallet response')
              setStatus('error')
            }
          } else if (resp.status === 'error') {
            setError(resp.error_message || 'Connection denied')
            setStatus('error')
          } else if (resp.status === 'expired') {
            setError('Session expired. Please try again.')
            setStatus('error')
          }
        } catch {
          // Network error during poll — keep trying
        }
      }, POLL_INTERVAL)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start connection')
      setStatus('error')
    }
  }, [stopPolling])

  const [signQrUrl, setSignQrUrl] = useState<string | null>(null)
  const [signStatus, setSignStatus] = useState<'idle' | 'waiting' | 'done' | 'error'>('idle')
  const signPollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopSignPolling = useCallback(() => {
    if (signPollRef.current) {
      clearInterval(signPollRef.current)
      signPollRef.current = null
    }
  }, [])

  useEffect(() => {
    return () => stopSignPolling()
  }, [stopSignPolling])

  /**
   * Sign and send a serialized transaction via Phantom deeplink.
   * Shows a QR code for the user to scan with Phantom mobile,
   * polls for the signature, and returns it.
   */
  const signAndSendTransaction = useCallback(
    async (transactionB58: string): Promise<string> => {
      const session = localStorage.getItem('phantom_deeplink_session')
      if (!session) {
        throw new Error('Phantom deeplink not connected. Please reconnect wallet.')
      }

      const phantomPubKey = localStorage.getItem('phantom_deeplink_pubkey')
      if (!phantomPubKey) {
        throw new Error('Missing Phantom encryption key. Please reconnect wallet.')
      }

      // 1. Create sign relay session
      const { session_id } = await post<SessionResponse>('/api/v1/phantom/sign-session', {})

      // 2. Build deeplink URL and show as QR
      const dappKeyPair = getDappKeyPair()
      const url = buildSignTransactionUrl(dappKeyPair, session, phantomPubKey, transactionB58, session_id)
      setSignQrUrl(url)
      setSignStatus('waiting')

      // 3. Poll for result
      return new Promise<string>((resolve, reject) => {
        const start = Date.now()
        signPollRef.current = setInterval(async () => {
          if (Date.now() - start > MAX_POLL_TIME) {
            stopSignPolling()
            setSignStatus('error')
            setSignQrUrl(null)
            reject(new Error('Transaction signing timed out'))
            return
          }

          try {
            const resp = await get<PollResponse>(`/api/v1/phantom/poll/${session_id}`)
            if (resp.status === 'pending') return

            stopSignPolling()
            setSignQrUrl(null)

            if (resp.status === 'error') {
              setSignStatus('error')
              reject(new Error(resp.error_message || 'Transaction rejected'))
              return
            }

            if (resp.nonce && resp.data && resp.phantom_encryption_public_key) {
              try {
                const result = decryptSignResponse(
                  resp.phantom_encryption_public_key,
                  resp.nonce,
                  resp.data,
                  dappKeyPair,
                )
                // Submit the signed transaction to devnet ourselves
                const signature = await submitSignedTransaction(result.transaction)
                setSignStatus('done')
                resolve(signature)
              } catch {
                setSignStatus('error')
                reject(new Error('Failed to submit signed transaction'))
              }
              return
            }

            setSignStatus('error')
            reject(new Error('Unexpected response from Phantom'))
          } catch {
            // Network error — keep trying
          }
        }, POLL_INTERVAL)
      })
    },
    [stopSignPolling],
  )

  const disconnect = useCallback(() => {
    stopPolling()
    setStatus('idle')
    setQrUrl(null)
    setWalletAddress(null)
    setError(null)
    localStorage.removeItem('phantom_deeplink_wallet')
    localStorage.removeItem('phantom_deeplink_session')
    localStorage.removeItem('phantom_deeplink_pubkey')
  }, [stopPolling])

  return {
    connectViaQR,
    signAndSendTransaction,
    signQrUrl,
    signStatus,
    qrUrl,
    status,
    walletAddress,
    error,
    disconnect,
  }
}
