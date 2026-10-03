import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { Toast, ToastContainer } from 'react-bootstrap'
import { api, setUnauthorizedHandler, type Status, type User } from './api'

interface AuthState {
  status: Status | null
  user: User | null
  refresh: () => Promise<void>
  signedIn: (user: User) => void
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthState>(null!)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status | null>(null)

  const refresh = useCallback(async () => {
    setStatus(await api.status())
  }, [])

  useEffect(() => {
    refresh()
    setUnauthorizedHandler(() => setStatus((s) => (s ? { ...s, user: null } : s)))
  }, [refresh])

  const value: AuthState = {
    status,
    user: status?.user ?? null,
    refresh,
    signedIn: (user) => setStatus({ setupRequired: false, user }),
    signOut: async () => {
      await api.logout()
      setStatus((s) => (s ? { ...s, user: null } : s))
    },
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)

type FlashKind = 'success' | 'danger' | 'info'
interface FlashMessage {
  id: number
  kind: FlashKind
  text: string
}

const FlashContext = createContext<(kind: FlashKind, text: string) => void>(() => {})

let nextFlashId = 1

export function FlashProvider({ children }: { children: ReactNode }) {
  const [messages, setMessages] = useState<FlashMessage[]>([])

  const flash = useCallback((kind: FlashKind, text: string) => {
    setMessages((m) => [...m, { id: nextFlashId++, kind, text }])
  }, [])

  const dismiss = (id: number) => setMessages((m) => m.filter((x) => x.id !== id))

  return (
    <FlashContext.Provider value={flash}>
      {children}
      <ToastContainer position="top-center" className="p-3 flash-container">
        {messages.map((m) => (
          <Toast key={m.id} bg={m.kind} onClose={() => dismiss(m.id)} delay={4000} autohide>
            <Toast.Body className="d-flex align-items-center text-white">
              <span className="flex-grow-1">{m.text}</span>
              <button
                type="button"
                className="btn-close btn-close-white ms-2"
                aria-label="Close"
                onClick={() => dismiss(m.id)}
              />
            </Toast.Body>
          </Toast>
        ))}
      </ToastContainer>
    </FlashContext.Provider>
  )
}

export const useFlash = () => useContext(FlashContext)
