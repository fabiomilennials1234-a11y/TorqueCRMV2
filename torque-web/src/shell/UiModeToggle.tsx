import { LayoutDashboard, UserRound } from 'lucide-react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useUiMode } from '@/providers/UiModeProvider'
import { Tooltip } from '@/ui/tooltip'
import { cn } from '@/lib/utils'
import type { UiMode } from '@/contracts/manual'

type Variant = 'appshell' | 'cockpit'

interface UiModeToggleProps {
  variant?: Variant
  className?: string
}

/**
 * Alternador entre modo Vendedor (salesperson) e Gerente (manager).
 * Presente no TopBar do AppShell (variant="appshell") e no CockpitHeader
 * (variant="cockpit", com linguagem claymorphism).
 *
 * Clicar alterna o modo global e navega para /cockpit ou /.
 */
export function UiModeToggle({ variant = 'appshell', className }: UiModeToggleProps) {
  const { mode, setMode, canSwitchToManager, canSwitchToSalesperson } = useUiMode()
  const navigate = useNavigate()
  const location = useLocation()

  async function choose(next: UiMode) {
    if (next === mode) return
    if (next === 'manager' && !canSwitchToManager) return
    if (next === 'salesperson' && !canSwitchToSalesperson) return
    await setMode(next)
    if (next === 'salesperson' && !location.pathname.startsWith('/cockpit')) {
      void navigate('/cockpit', { replace: false })
    } else if (next === 'manager' && location.pathname.startsWith('/cockpit')) {
      void navigate('/', { replace: false })
    }
  }

  const isClay = variant === 'cockpit'

  const containerClass = cn(
    'relative inline-flex items-center p-0.5 transition-all select-none',
    isClay
      ? 'rounded-full bg-[hsl(var(--clay-panel-down))] shadow-[var(--clay-shadow-sunken)]'
      : 'rounded-md bg-elevated/60 shadow-hairline',
    className
  )

  const itemBase = cn(
    'relative z-10 inline-flex items-center gap-1.5 whitespace-nowrap',
    'transition-colors duration-200',
    isClay
      ? 'h-8 px-3 text-[0.8125rem] font-medium rounded-full'
      : 'h-7 px-2.5 text-[0.75rem] font-medium rounded'
  )

  const pillBase = cn(
    'pointer-events-none absolute top-0.5 bottom-0.5 w-1/2',
    'transition-[transform,box-shadow] ease-[var(--ease-out-soft)]',
    isClay
      ? 'rounded-full bg-[hsl(var(--clay-panel-up))] shadow-[var(--clay-shadow-raised)]'
      : 'rounded bg-surface shadow-[0_1px_0_0_hsl(var(--hairline)/0.5),_inset_0_0_0_1px_hsl(var(--hairline)/0.6)]'
  )

  const salespersonActive = mode === 'salesperson'
  const pillTransform = salespersonActive ? 'translate-x-[2px]' : 'translate-x-[calc(100%-2px)]'
  const pillDuration = isClay ? 'duration-[380ms]' : 'duration-200'

  const activeText = isClay ? 'text-ink' : 'text-ink'
  const idleText = isClay
    ? 'text-[hsl(var(--ink-dim))] hover:text-ink'
    : 'text-ink-dim hover:text-ink'

  const managerDisabled = !canSwitchToManager

  return (
    <div role="radiogroup" aria-label="Modo de interface" className={containerClass}>
      <span aria-hidden className={cn(pillBase, pillDuration, pillTransform)} />

      <Tooltip content="Modo Vendedor · G V" side="bottom">
        <button
          type="button"
          role="radio"
          aria-checked={salespersonActive}
          disabled={!canSwitchToSalesperson}
          onClick={() => void choose('salesperson')}
          className={cn(
            itemBase,
            salespersonActive ? activeText : idleText,
            'disabled:cursor-not-allowed disabled:opacity-40'
          )}
        >
          <UserRound className="h-3.5 w-3.5" strokeWidth={salespersonActive ? 2 : 1.6} />
          <span>Vendedor</span>
        </button>
      </Tooltip>

      <Tooltip
        content={managerDisabled ? 'Seu acesso não permite o modo Gerente' : 'Modo Gerente · G M'}
        side="bottom"
      >
        <button
          type="button"
          role="radio"
          aria-checked={!salespersonActive}
          aria-disabled={managerDisabled}
          disabled={managerDisabled}
          onClick={() => void choose('manager')}
          className={cn(
            itemBase,
            !salespersonActive ? activeText : idleText,
            'disabled:cursor-not-allowed disabled:opacity-40'
          )}
        >
          <LayoutDashboard className="h-3.5 w-3.5" strokeWidth={!salespersonActive ? 2 : 1.6} />
          <span>Gerente</span>
        </button>
      </Tooltip>
    </div>
  )
}
