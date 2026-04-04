import { Outlet } from 'react-router-dom'
import { SolanaProvider } from '@solana/react-hooks'
import { createClient, autoDiscover } from '@solana/client'
import { TopBar } from '../components/TopBar/TopBar'
import styles from './AppLayout.module.css'

const endpoint = import.meta.env.VITE_SOLANA_RPC_URL || 'https://api.devnet.solana.com'
const wsEndpoint = endpoint.replace('https://', 'wss://').replace('http://', 'ws://')

const solanaClient = createClient({
  endpoint,
  websocketEndpoint: wsEndpoint,
  walletConnectors: autoDiscover(),
})

export function AppLayout() {
  return (
    <SolanaProvider client={solanaClient}>
      <div className={styles.layout}>
        <TopBar />
        <main className={styles.content}>
          <Outlet />
        </main>
      </div>
    </SolanaProvider>
  )
}
