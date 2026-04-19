import { Bell, Search, Plus, Sun, Moon } from 'lucide-react'
import { Button } from '@/ui/button'
import { Kbd } from '@/ui/kbd'
import { Avatar } from '@/ui/avatar'
import { Tooltip } from '@/ui/tooltip'
import { useTheme } from '@/providers/ThemeProvider'
import { UiModeToggle } from './UiModeToggle'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/ui/dropdown'

export function TopBar({ onOpenCommand }: { onOpenCommand: () => void }) {
  const { theme, toggle } = useTheme()

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 bg-bg/80 px-6 shadow-hairline-b backdrop-blur-xl">
      <button
        onClick={onOpenCommand}
        className="group flex h-9 w-[340px] items-center gap-2.5 rounded-md bg-surface/80 pl-3 pr-2 text-left shadow-hairline transition-shadow duration-150 hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim)/0.4)]"
      >
        <Search className="h-3.5 w-3.5 text-ink-dim" strokeWidth={2} />
        <span className="flex-1 truncate text-sm text-ink-dim">
          Buscar leads, conversas, acoes...
        </span>
        <Kbd>⌘</Kbd>
        <Kbd>K</Kbd>
      </button>

      <div className="ml-auto flex items-center gap-1.5">
        <Tooltip content="Novo lead" shortcut="N">
          <Button variant="secondary" size="sm" className="gap-1.5">
            <Plus className="h-3.5 w-3.5" />
            Novo lead
          </Button>
        </Tooltip>

        <div className="mx-1 h-5 w-px bg-hairline/60" aria-hidden />

        <UiModeToggle variant="appshell" />

        <Tooltip content={theme === 'dark' ? 'Tema claro' : 'Tema escuro'}>
          <Button variant="ghost" size="icon" onClick={toggle}>
            {theme === 'dark' ? (
              <Sun className="h-4 w-4" strokeWidth={1.75} />
            ) : (
              <Moon className="h-4 w-4" strokeWidth={1.75} />
            )}
          </Button>
        </Tooltip>

        <Tooltip content="Notificacoes">
          <Button variant="ghost" size="icon" className="relative">
            <Bell className="h-4 w-4" strokeWidth={1.75} />
            <span className="absolute right-1.5 top-1.5 h-1.5 w-1.5 rounded-full bg-accent ring-2 ring-bg" />
          </Button>
        </Tooltip>

        <UserMenu />
      </div>
    </header>
  )
}

function UserMenu() {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="ml-1 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-accent">
          <Avatar size="md" fallback="FM" ring />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[220px]">
        <div className="px-2 py-2">
          <div className="text-sm font-medium text-ink">Fabio Milennials</div>
          <div className="text-xs text-ink-dim">fabio@milennials.com</div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Conta</DropdownMenuLabel>
        <DropdownMenuItem>
          Perfil
          <DropdownMenuShortcut>⌘ ,</DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuItem>
          Preferencias
          <DropdownMenuShortcut>⌘ ⇧ P</DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuItem>Dispositivos conectados</DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Organizacao</DropdownMenuLabel>
        <DropdownMenuItem>Time e permissoes</DropdownMenuItem>
        <DropdownMenuItem>Integracoes</DropdownMenuItem>
        <DropdownMenuItem>Faturamento</DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem className="text-danger data-[highlighted]:text-danger">
          Sair
          <DropdownMenuShortcut>⌘ ⇧ Q</DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
