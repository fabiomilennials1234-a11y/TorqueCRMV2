import { Link } from 'react-router-dom'
import { TorqueMark } from '@/shell/TorqueMark'
import { Button } from '@/ui/button'

export function NotFoundPage() {
  return (
    <div className="bg-bg relative flex min-h-[calc(100vh-3.5rem)] w-full items-center justify-center">
      {/* subtle dot-grid background */}
      <div className="bg-dot-grid pointer-events-none absolute inset-0 [background-size:16px_16px] [background-position:center]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_0%,hsl(var(--bg))_70%)]" />

      <div className="relative z-10 flex max-w-md flex-col items-center px-6 text-center">
        {/* Animated TorqueMark */}
        <div className="animate-torque-tick mb-6">
          <TorqueMark size={48} />
        </div>

        {/* 404 editorial heading */}
        <h1 className="font-display tracking-tightest text-ink text-6xl">404</h1>

        {/* Description */}
        <p className="text-ink-muted mt-3 text-sm leading-relaxed">
          A pagina que voce procura nao existe ou foi movida.
        </p>

        {/* CTA */}
        <div className="mt-6">
          <Button variant="secondary" size="md" asChild>
            <Link to="/">Voltar ao inicio</Link>
          </Button>
        </div>
      </div>
    </div>
  )
}
