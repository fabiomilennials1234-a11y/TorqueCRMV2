import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useAuth } from '@/providers/AuthProvider'
import type { UiMode } from '@/contracts/manual'

const STORAGE_KEY = 'torque.ui_mode'

interface UiModeContextValue {
  mode: UiMode
  setMode: (mode: UiMode) => Promise<void>
  canSwitchToManager: boolean
  canSwitchToSalesperson: boolean
  isPersisting: boolean
}

const UiModeContext = createContext<UiModeContextValue | null>(null)

function readStorage(): UiMode | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === 'manager' || raw === 'salesperson') return raw
  } catch {
    // ignore (SSR / blocked storage)
  }
  return null
}

function writeStorage(mode: UiMode): void {
  try {
    localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // ignore
  }
}

function roleDefault(role: 'admin' | 'membro' | 'master' | null): UiMode {
  if (role === 'membro') return 'salesperson'
  return 'manager'
}

export function UiModeProvider({ children }: { children: ReactNode }) {
  const { session } = useAuth()

  const role = session?.role ?? null
  const isMaster = session?.isMaster ?? false
  const backendPreference = session?.user.uiPreferences?.mode ?? null

  // Stabilize reference so the memo deps below don't flip every render.
  const featurePermissions = useMemo(
    () => session?.featurePermissions ?? {},
    [session?.featurePermissions]
  )

  const canSwitchToManager = useMemo(() => {
    if (isMaster) return true
    if (role === 'admin') return true
    if (featurePermissions['ui.view_manager_mode'] === true) return true
    return false
  }, [isMaster, role, featurePermissions])

  const canSwitchToSalesperson = useMemo(() => {
    if (isMaster) return false // master sempre manager
    return featurePermissions['ui.view_salesperson_mode'] !== false
  }, [isMaster, featurePermissions])

  const [mode, setModeState] = useState<UiMode>(() => {
    // Hint inicial antes do AuthProvider hidratar — evita flash.
    return readStorage() ?? 'manager'
  })
  const [isPersisting, setIsPersisting] = useState(false)

  // Reconciliação quando a sessão chega do backend.
  useEffect(() => {
    if (!session) return

    // 1) Preferência do backend ganha prioridade.
    if (backendPreference) {
      if (backendPreference !== mode) {
        setModeState(backendPreference)
        writeStorage(backendPreference)
      }
      return
    }

    // 2) localStorage (render inicial) — se existe e é válido, mantém.
    const stored = readStorage()
    if (stored) {
      // Mas se stored=salesperson e master, corrige para manager.
      if (isMaster && stored === 'salesperson') {
        setModeState('manager')
        writeStorage('manager')
      } else if (stored === 'manager' && !canSwitchToManager) {
        setModeState('salesperson')
        writeStorage('salesperson')
      } else if (stored !== mode) {
        setModeState(stored)
      }
      return
    }

    // 3) Fallback por role.
    const derived = roleDefault(role)
    if (derived !== mode) {
      setModeState(derived)
      writeStorage(derived)
    }
  }, [session, backendPreference, isMaster, role, canSwitchToManager]) // eslint-disable-line react-hooks/exhaustive-deps

  const setMode = useCallback(
    async (next: UiMode) => {
      if (next === 'manager' && !canSwitchToManager) return
      if (next === 'salesperson' && !canSwitchToSalesperson) return
      if (next === mode) return

      setIsPersisting(true)
      setModeState(next)
      writeStorage(next)

      // TODO: quando o backend existir:
      // await patch("/me/preferences", { ui_mode: next });
      // queryClient.invalidateQueries(['session']);

      setIsPersisting(false)
    },
    [canSwitchToManager, canSwitchToSalesperson, mode]
  )

  const value: UiModeContextValue = {
    mode,
    setMode,
    canSwitchToManager,
    canSwitchToSalesperson,
    isPersisting,
  }

  return <UiModeContext.Provider value={value}>{children}</UiModeContext.Provider>
}

export function useUiMode(): UiModeContextValue {
  const ctx = useContext(UiModeContext)
  if (!ctx) {
    throw new Error('useUiMode must be used within <UiModeProvider>')
  }
  return ctx
}
