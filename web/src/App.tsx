import { lazy, Suspense } from 'react'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { AppLayout } from './layouts/AppLayout'
import { NotFound } from './pages/NotFound/NotFound'
import { Skeleton } from './components/Skeleton/Skeleton'
import { ToastProvider } from './components/Toast/Toast'

const WizardFlow = lazy(() => import('./pages/WizardFlow/WizardFlow'))
const ParcelDetail = lazy(() => import('./pages/ParcelDetail/ParcelDetail'))

function PageFallback() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
      <Skeleton width="280px" height="38px" />
      <Skeleton width="100%" height="160px" />
    </div>
  )
}

const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <Suspense fallback={<PageFallback />}><WizardFlow /></Suspense> },
      { path: 'parcel/:cadastral', element: <Suspense fallback={<PageFallback />}><ParcelDetail /></Suspense> },
      { path: '*', element: <NotFound /> },
    ],
  },
])

export function App() {
  return (
    <ToastProvider>
      <RouterProvider router={router} />
    </ToastProvider>
  )
}
