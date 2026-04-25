import { Bell, LogOut, Search, Settings2 } from 'lucide-react'
import { UiModeToggle } from '@/shell/UiModeToggle'
import { Avatar } from '@/ui/avatar'
import { Tooltip } from '@/ui/tooltip'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown'
import { useAuth } from '@/providers/AuthProvider'
import { useSession } from '@/hooks/useSession'
import { initials } from '@/lib/utils'

import torqueLogoDark from '@/assets/torque-logo.png'

export function CockpitHeader() {
  return (
    <header className="relative z-20 flex h-16 shrink-0 items-center gap-4 px-6" role="banner">
      <div className="flex items-center gap-3">
        <img
          src={torqueLogoDark}
          alt="Torque"
          className="h-7 w-auto drop-shadow-[0_1px_2px_rgb(0_0_0_/_0.5)]"
        />
        <div className="hidden h-5 w-px bg-[hsl(var(--clay-rim)/0.5)] md:block" />
        <span className="text-ink-dim hidden text-[0.6875rem] font-medium tracking-[0.18em] uppercase md:block">
          Cockpit
        </span>
      </div>

      <CockpitSearch />

      <div className="ml-auto flex items-center gap-2">
        <UiModeToggle variant="cockpit" />

        <Tooltip content="Notificações" side="bottom">
          <button
            type="button"
            aria-label="Notificações"
            className="text-ink-muted hover:text-ink relative flex h-9 w-9 items-center justify-center rounded-full bg-[hsl(var(--clay-panel))] shadow-[var(--clay-shadow-raised)] transition-[box-shadow,transform] hover:-translate-y-[1px] active:translate-y-[0.5px] active:shadow-[var(--clay-shadow-sunken)]"
          >
            <Bell className="h-4 w-4" strokeWidth={1.75} />
            <span className="absolute top-2 right-2 h-1.5 w-1.5 rounded-full bg-[hsl(var(--accent))] shadow-[0_0_6px_0_hsl(var(--accent))]" />
          </button>
        </Tooltip>

        <CockpitUserMenu />
      </div>
    </header>
  )
}

function CockpitSearch() {
  return (
    <button
      type="button"
      className="group flex h-9 max-w-[380px] min-w-0 flex-1 items-center gap-2.5 rounded-full bg-[hsl(var(--clay-panel-down))] px-3.5 text-left shadow-[var(--clay-shadow-sunken)] transition-[box-shadow] hover:shadow-[var(--clay-ring-focus)]"
      aria-label="Abrir busca"
    >
      <Search className="text-ink-dim h-3.5 w-3.5" strokeWidth={2} />
      <span className="text-ink-dim flex-1 truncate text-[0.8125rem]">
        Buscar leads, conversas, ações…
      </span>
      <kbd className="text-ink-dim hidden items-center rounded-md bg-[hsl(var(--clay-panel))] px-1.5 py-0.5 font-mono text-[10px] shadow-[inset_0_1px_0_0_hsl(var(--clay-rim)/0.5)] sm:inline-flex">
        ⌘K
      </kbd>
    </button>
  )
}

function CockpitUserMenu() {
  const { logout } = useAuth()
  const { user } = useSession()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label="Menu do usuário"
          className="ml-1 rounded-full outline-none focus-visible:shadow-[var(--clay-ring-focus)]"
        >
          <Avatar size="md" fallback={initials(user.displayName || '?')} ring />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[220px]">
        <div className="px-2 py-2">
          <div className="text-ink text-sm font-medium">{user.displayName || '—'}</div>
          <div className="text-ink-dim text-xs">{user.email}</div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Conta</DropdownMenuLabel>
        <DropdownMenuItem>
          <Settings2 className="mr-2 h-3.5 w-3.5" strokeWidth={1.75} />
          Preferências
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="text-danger data-[highlighted]:text-danger"
          onClick={() => void logout()}
        >
          <LogOut className="mr-2 h-3.5 w-3.5" strokeWidth={1.75} />
          Sair
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
