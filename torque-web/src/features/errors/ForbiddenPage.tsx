import { Link } from 'react-router-dom'
import { TorqueMark } from '@/shell/TorqueMark'
import { Button } from '@/ui/button'
import { useAuth } from '@/providers/AuthProvider'

export function ForbiddenPage() {
  const { logout } = useAuth()

  return (
    <div className="relative flex min-h-[calc(100vh-3.5rem)] w-full items-center justify-center bg-bg">
      {/* subtle dot-grid background */}
      <div className="pointer-events-none absolute inset-0 bg-dot-grid [background-position:center] [background-size:16px_16px]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_0%,hsl(var(--bg))_70%)]" />

      <div className="relative z-10 flex max-w-md flex-col items-center px-6 text-center">
        {/* TorqueMark */}
        <div className="mb-5">
          <TorqueMark size={32} />
        </div>

        {/* Heading */}
        <h1 className="font-display text-3xl tracking-tightest text-ink">Acesso restrito</h1>

        {/* Description */}
        <p className="mt-3 text-sm leading-relaxed text-ink-muted">
          Voce nao tem permissao para acessar esta area.
        </p>

        {/* Actions */}
        <div className="mt-6 flex items-center gap-3">
          <Button variant="secondary" size="md" asChild>
            <Link to="/">Voltar</Link>
          </Button>
          <Button variant="danger" size="md" onClick={() => void logout()}>
            Sair da conta
          </Button>
        </div>
      </div>
    </div>
  )
}
