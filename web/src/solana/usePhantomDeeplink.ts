import { useState, useCallback, useRef, useEffect } from 'react'
import { post, get } from '../api/client'
import {
  getDappKeyPair,
  buildConnectUrl,
  decryptConnectResponse,
  buildSignAndSendUrl,
  decryptSignResponse,
} from './phantom-deeplink'

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
  const [status, setStatus] = useState<DeeplinkStatus>('idle')
  const [qrUrl, setQrUrl] = useState<string | null>(null)
  const [walletAddress, setWalletAddress] = useState<string | null>(null)
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

  /**
   * Sign and send a serialized transaction via Phantom deeplink.
   * Opens Phantom on mobile, user approves, tx is submitted by Phantom,
   * callback stores signature, we poll and return it.
   */
  const signAndSendTransaction = useCallback(
    async (transactionB58: string): Promise<string> => {
      const session = localStorage.getItem('phantom_deeplink_session')
      const storedWallet = localStorage.getItem('phantom_deeplink_wallet')
      if (!session || !storedWallet) {
        throw new Error('Phantom deeplink not connected. Please reconnect wallet.')
      }

      // 1. Create a sign relay session on backend
      const { session_id } = await post<SessionResponse>('/api/v1/phantom/sign-session', {})

      // 2. Build deeplink URL
      const dappKeyPair = getDappKeyPair()
      // We need the phantom encryption public key — stored during connect
      // Actually for signAndSendTransaction, Phantom uses the same shared secret from connect
      // The phantomPubKey was used during connect decrypt — we need to store it
      const phantomPubKey = localStorage.getItem('phantom_deeplink_pubkey')
      if (!phantomPubKey) {
        throw new Error('Missing Phantom encryption key. Please reconnect wallet.')
      }

      const url = buildSignAndSendUrl(
        dappKeyPair,
        session,
        phantomPubKey,
        transactionB58,
        session_id,
      )

      // 3. Open deeplink
      window.open(url, '_blank')

      // 4. Poll for result
      return new Promise<string>((resolve, reject) => {
        const start = Date.now()
        const interval = setInterval(async () => {
          if (Date.now() - start > MAX_POLL_TIME) {
            clearInterval(interval)
            reject(new Error('Transaction signing timed out'))
            return
          }

          try {
            const resp = await get<PollResponse>(`/api/v1/phantom/poll/${session_id}`)
            if (resp.status === 'pending') return

            clearInterval(interval)

            if (resp.status === 'error') {
              reject(new Error(resp.error_message || 'Transaction rejected'))
              return
            }

            if (resp.status === 'connected' && resp.nonce && resp.data) {
              try {
                const result = decryptSignResponse(
                  resp.phantom_encryption_public_key!,
                  resp.nonce,
                  resp.data,
                  dappKeyPair,
                )
                resolve(result.signature)
              } catch (err) {
                reject(new Error('Failed to decrypt sign response'))
              }
              return
            }

            reject(new Error('Unexpected response from Phantom'))
          } catch {
            // Network error during poll — keep trying
          }
        }, POLL_INTERVAL)
      })
    },
    [],
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
    qrUrl,
    status,
    walletAddress,
    error,
    disconnect,
  }
}
