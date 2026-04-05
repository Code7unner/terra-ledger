import { createContext, useContext } from 'react'

type ToastType = 'error' | 'success' | 'info'

export interface ToastContextValue {
  toast: (message: string, type?: ToastType) => void
}

export const ToastContext = createContext<ToastContextValue>({ toast: () => {} })

export function useToast() {
  return useContext(ToastContext)
}
