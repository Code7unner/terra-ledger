import { lazy, Suspense } from 'react'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { AppLayout } from './layouts/AppLayout'
import { LenderDashboard } from './pages/LenderDashboard/LenderDashboard'
import { Skeleton } from './components/Skeleton/Skeleton'

const ParcelDetail = lazy(() => import('./pages/ParcelDetail/ParcelDetail'))
const LienManagement = lazy(() => import('./pages/LienManagement/LienManagement'))
const FarmerPortal = lazy(() => import('./pages/FarmerPortal/FarmerPortal'))
const ConsentDashboard = lazy(() => import('./pages/ConsentDashboard/ConsentDashboard'))

function PageFallback() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
        <Skeleton width="280px" height="38px" />
        <Skeleton width="200px" height="20px" />
      </div>
      <Skeleton width="100%" height="160px" />
      <Skeleton width="100%" height="120px" />
    </div>
  )
}

const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { index: true, element: <LenderDashboard /> },
      { path: 'parcel/:cadastral', element: <Suspense fallback={<PageFallback />}><ParcelDetail /></Suspense> },
      { path: 'liens', element: <Suspense fallback={<PageFallback />}><LienManagement /></Suspense> },
      { path: 'farmer', element: <Suspense fallback={<PageFallback />}><FarmerPortal /></Suspense> },
      { path: 'farmer/consent', element: <Suspense fallback={<PageFallback />}><ConsentDashboard /></Suspense> },
    ],
  },
])

export function App() {
  return <RouterProvider router={router} />
}
