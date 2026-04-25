import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

/**
 * Top-level error boundary. Catches unhandled React errors and renders
 * a full-screen recovery UI matching the Torque design language.
 */
export class RootErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Log to console in dev; Sentry captures automatically via its integration
    console.error('[RootErrorBoundary]', error, info.componentStack)
  }

  private handleReload = () => {
    window.location.reload()
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="bg-bg flex h-screen w-screen flex-col items-center justify-center px-6 text-center">
          <h1 className="font-display tracking-tightest text-ink text-3xl">Algo deu errado</h1>
          <p className="text-ink-muted mt-3 max-w-sm text-sm leading-relaxed">
            Um erro inesperado impediu a aplicacao de continuar. Tente recarregar a pagina.
          </p>
          <button
            type="button"
            onClick={this.handleReload}
            className="bg-accent text-bg focus-visible:ring-accent focus-visible:ring-offset-bg mt-8 rounded-md px-5 py-2.5 text-sm font-semibold transition-opacity duration-150 ease-[var(--ease-out-soft)] hover:opacity-90 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
          >
            Recarregar pagina
          </button>
        </div>
      )
    }

    return this.props.children
  }
}
