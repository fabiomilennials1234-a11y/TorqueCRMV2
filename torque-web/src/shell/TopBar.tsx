import { Bell, Search, Plus, Sun, Moon, Menu } from 'lucide-react'
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

export function TopBar({
  onOpenCommand,
  onOpenSidebar,
}: {
  onOpenCommand: () => void
  onOpenSidebar?: () => void
}) {
  const { theme, toggle } = useTheme()

  return (
    <header className="bg-bg/80 shadow-hairline-b sticky top-0 z-30 flex h-14 shrink-0 items-center gap-2 px-3 backdrop-blur-xl sm:gap-3 sm:px-4 lg:px-6">
      {/* S56 — hamburger trigger, mobile-only. Hidden at >=lg where Sidebar is fixed. */}
      {onOpenSidebar && (
        <Tooltip content="Abrir menu">
          <Button
            variant="ghost"
            size="icon"
            onClick={onOpenSidebar}
            aria-label="Abrir menu de navegacao"
            className="h-11 w-11 lg:hidden"
          >
            <Menu className="h-5 w-5" strokeWidth={1.75} />
          </Button>
        </Tooltip>
      )}

      <button
        onClick={onOpenCommand}
        aria-label="Buscar"
        className="group bg-surface/80 shadow-hairline flex h-9 min-w-0 flex-1 items-center gap-2.5 rounded-md pr-2 pl-3 text-left transition-shadow duration-150 hover:shadow-[inset_0_0_0_1px_hsl(var(--ink-dim)/0.4)] sm:flex-initial sm:w-[260px] md:w-[340px]"
      >
        <Search className="text-ink-dim h-3.5 w-3.5 shrink-0" strokeWidth={2} />
        <span className="text-ink-dim hidden flex-1 truncate text-sm sm:inline">
          Buscar leads, conversas, acoes...
        </span>
        <span className="text-ink-dim flex-1 truncate text-sm sm:hidden">Buscar...</span>
        <span className="hidden items-center gap-1 sm:flex">
          <Kbd>⌘</Kbd>
          <Kbd>K</Kbd>
        </span>
      </button>

      <div className="ml-auto flex items-center gap-1.5">
        {/* S56 — primary CTA visible on >=md; collapses to icon on <md. */}
        <Tooltip content="Novo lead" shortcut="N">
          <Button variant="secondary" size="sm" className="hidden gap-1.5 md:inline-flex">
            <Plus className="h-3.5 w-3.5" />
            Novo lead
          </Button>
        </Tooltip>
        <Tooltip content="Novo lead">
          <Button
            variant="secondary"
            size="icon"
            aria-label="Novo lead"
            className="h-11 w-11 md:hidden"
          >
            <Plus className="h-4 w-4" strokeWidth={2} />
          </Button>
        </Tooltip>

        {/* S56 — secondary actions hidden on <md, surfaced via UserMenu/kebab. */}
        <div className="bg-hairline/60 mx-1 hidden h-5 w-px md:block" aria-hidden />

        <div className="hidden md:block">
          <UiModeToggle variant="appshell" />
        </div>

        <Tooltip content={theme === 'dark' ? 'Tema claro' : 'Tema escuro'}>
          <Button
            variant="ghost"
            size="icon"
            onClick={toggle}
            aria-label="Alternar tema"
            className="hidden md:inline-flex"
          >
            {theme === 'dark' ? (
              <Sun className="h-4 w-4" strokeWidth={1.75} />
            ) : (
              <Moon className="h-4 w-4" strokeWidth={1.75} />
            )}
          </Button>
        </Tooltip>

        <Tooltip content="Notificacoes">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Notificacoes"
            className="relative h-11 w-11 md:h-8 md:w-8"
          >
            <Bell className="h-4 w-4" strokeWidth={1.75} />
            <span className="bg-accent ring-bg absolute top-1.5 right-1.5 h-1.5 w-1.5 rounded-full ring-2" />
          </Button>
        </Tooltip>

        <UserMenu themeToggle={toggle} theme={theme} />
      </div>
    </header>
  )
}

function UserMenu({ themeToggle, theme }: { themeToggle: () => void; theme: 'light' | 'dark' }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label="Abrir menu do usuario"
          className="focus-visible:ring-accent ml-1 flex min-h-[44px] min-w-[44px] items-center justify-center rounded-full outline-none focus-visible:ring-2 md:min-h-0 md:min-w-0"
        >
          <Avatar size="md" fallback="FM" ring />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[220px]">
        <div className="px-2 py-2">
          <div className="text-ink text-sm font-medium">Fabio Milennials</div>
          <div className="text-ink-dim text-xs">fabio@milennials.com</div>
        </div>
        <DropdownMenuSeparator />
        {/* S56 — theme toggle exposed in user menu on mobile (icon hidden in TopBar <md). */}
        <DropdownMenuItem className="md:hidden" onSelect={themeToggle}>
          {theme === 'dark' ? 'Tema claro' : 'Tema escuro'}
        </DropdownMenuItem>
        <div className="md:hidden">
          <DropdownMenuSeparator />
        </div>
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
