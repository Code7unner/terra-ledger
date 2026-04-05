import { useState, useCallback, useRef, useEffect } from 'react'
import { post, get } from '../api/client'
import { getDappKeyPair, buildConnectUrl, decryptConnectResponse } from './phantom-deeplink'

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

  const disconnect = useCallback(() => {
    stopPolling()
    setStatus('idle')
    setQrUrl(null)
    setWalletAddress(null)
    setError(null)
    localStorage.removeItem('phantom_deeplink_wallet')
    localStorage.removeItem('phantom_deeplink_session')
  }, [stopPolling])

  return {
    connectViaQR,
    qrUrl,
    status,
    walletAddress,
    error,
    disconnect,
  }
}
