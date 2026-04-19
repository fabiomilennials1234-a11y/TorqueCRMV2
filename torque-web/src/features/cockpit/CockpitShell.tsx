import { useEffect, useState } from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import { TooltipProvider } from '@/ui/tooltip'
import { TopBar } from '@/shell/TopBar'
import { CommandPalette } from '@/shell/CommandPalette'
import { useUiMode } from '@/providers/UiModeProvider'

/**
 * Layout raiz do modo Vendedor (rota /cockpit).
 * TopBar idêntica ao AppShell (inclui o UiModeToggle, que é o único escape).
 * Aplica .cockpit-theme para ativar os tokens claymorphism no conteúdo.
 */
export function CockpitShell() {
  const { mode } = useUiMode()
  const navigate = useNavigate()
  const [commandOpen, setCommandOpen] = useState(false)

  // Atalho G M → / (G V não existe aqui: já está no cockpit)
  useEffect(() => {
    let lastG = 0
    function handler(e: KeyboardEvent) {
      if (e.target instanceof HTMLElement) {
        const tag = e.target.tagName
        if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable) {
          return
        }
      }
      if (e.key.toLowerCase() === 'g') {
        lastG = Date.now()
        return
      }
      if (Date.now() - lastG < 900 && e.key.toLowerCase() === 'm') {
        e.preventDefault()
        navigate('/')
        lastG = 0
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [navigate])

  useEffect(() => {
    if (mode === 'manager') {
      navigate('/', { replace: true })
    }
  }, [mode, navigate])

  return (
    <TooltipProvider delayDuration={200} skipDelayDuration={300}>
      <div className="flex h-screen w-screen flex-col overflow-hidden bg-bg text-ink">
        <TopBar onOpenCommand={() => setCommandOpen(true)} />
        <main className="cockpit-theme flex-1 overflow-hidden">
          <Outlet />
        </main>
        <CommandPalette open={commandOpen} onOpenChange={setCommandOpen} />
      </div>
    </TooltipProvider>
  )
}
