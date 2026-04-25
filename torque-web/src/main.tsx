import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import * as Sentry from '@sentry/react'
import { QueryProvider } from '@/providers/QueryProvider'
import { AuthProvider } from '@/providers/AuthProvider'
import { IntlProvider } from '@/providers/IntlProvider'
import { WSProvider } from '@/providers/WSProvider'
import { ThemeProvider } from '@/providers/ThemeProvider'
import { UiModeProvider } from '@/providers/UiModeProvider'
import { RootErrorBoundary } from '@/components/RootErrorBoundary'
import { router } from '@/routes'
import './styles/globals.css'

// ---------------------------------------------------------------------------
// Sentry — initialize only when DSN is provided
// ---------------------------------------------------------------------------

const sentryDsn = import.meta.env.VITE_SENTRY_DSN as string | undefined

if (sentryDsn) {
  Sentry.init({
    dsn: sentryDsn,
    environment: import.meta.env.MODE,
    tracesSampleRate: 0.2,
    replaysSessionSampleRate: 0,
    replaysOnErrorSampleRate: 1.0,
  })
}

// ---------------------------------------------------------------------------
// Fallback UI for Sentry.ErrorBoundary
// ---------------------------------------------------------------------------

function SentryFallback() {
  return (
    <div className="bg-bg flex h-screen w-screen flex-col items-center justify-center px-6 text-center">
      <h1 className="font-display tracking-tightest text-ink text-3xl">Algo deu errado</h1>
      <p className="text-ink-muted mt-3 max-w-sm text-sm leading-relaxed">
        Um erro inesperado impediu a aplicacao de continuar. Tente recarregar a pagina.
      </p>
      <button
        type="button"
        onClick={() => window.location.reload()}
        className="bg-accent text-bg mt-8 rounded-md px-5 py-2.5 text-sm font-semibold transition-opacity duration-150 ease-[var(--ease-out-soft)] hover:opacity-90"
      >
        Recarregar pagina
      </button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Mount
// ---------------------------------------------------------------------------

const root = document.getElementById('root')
if (!root) throw new Error('Root element not found')

createRoot(root).render(
  <StrictMode>
    <Sentry.ErrorBoundary fallback={<SentryFallback />}>
      <RootErrorBoundary>
        <ThemeProvider>
          <QueryProvider>
            <AuthProvider>
              <UiModeProvider>
                <IntlProvider>
                  <WSProvider>
                    <RouterProvider router={router} />
                  </WSProvider>
                </IntlProvider>
              </UiModeProvider>
            </AuthProvider>
          </QueryProvider>
        </ThemeProvider>
      </RootErrorBoundary>
    </Sentry.ErrorBoundary>
  </StrictMode>
)
