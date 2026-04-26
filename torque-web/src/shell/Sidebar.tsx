import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Columns3,
  MessagesSquare,
  CheckSquare,
  Workflow,
  Megaphone,
  Sparkles,
  BarChart3,
  Users,
  Package,
  Settings2,
  LifeBuoy,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Tooltip } from '@/ui/tooltip'
import { cn } from '@/lib/utils'
import { useTheme } from '@/providers/ThemeProvider'
import { OrgSwitcher } from './OrgSwitcher'

import torqueLogoDark from '@/assets/torque-logo.png'
import torqueLogoLight from '@/assets/torque-logo-dark.png'

type NavItem = {
  to: string
  label: string
  icon: LucideIcon
  shortcut?: string
  badge?: string
}

const vendasNav: NavItem[] = [
  { to: '/', label: 'Visao geral', icon: LayoutDashboard, shortcut: 'G D' },
  { to: '/pipeline', label: 'Funis', icon: Columns3, shortcut: 'G P' },
  { to: '/inbox', label: 'Conversas', icon: MessagesSquare, shortcut: 'G I' },
  { to: '/follow-ups', label: 'Follow-ups', icon: CheckSquare, shortcut: 'G F' },
]

const automacaoNav: NavItem[] = [
  { to: '/workflows', label: 'Fluxos', icon: Workflow, shortcut: 'G W' },
  { to: '/campaigns', label: 'Campanhas', icon: Megaphone, shortcut: 'G C' },
]

const inteligenciaNav: NavItem[] = [
  { to: '/copilot', label: 'Agentes IA', icon: Sparkles, shortcut: 'G A' },
  { to: '/analytics', label: 'Analytics', icon: BarChart3, shortcut: 'G N' },
]

const equipeNav: NavItem[] = [
  { to: '/team', label: 'Time', icon: Users, shortcut: 'G T' },
  { to: '/products', label: 'Produtos', icon: Package },
]

const footerNav: NavItem[] = [
  { to: '/settings', label: 'Configuracoes', icon: Settings2, shortcut: 'G S' },
  { to: '/help', label: 'Ajuda', icon: LifeBuoy },
]

/**
 * Sidebar (desktop) — fixed aside in AppShell grid at >=lg breakpoint.
 * Below lg, AppShell renders SidebarContent inside a Sheet drawer (Sidebar.tsx itself stays hidden).
 */
export function Sidebar() {
  return (
    <aside
      className={cn(
        'hidden h-full w-[232px] shrink-0 flex-col lg:flex',
        'bg-surface/80 backdrop-blur-[8px]',
        'shadow-[inset_-1px_0_0_0_hsl(var(--hairline)/0.5)]'
      )}
    >
      <SidebarContent />
    </aside>
  )
}

/**
 * S56 — extracted nav body so the same content renders in:
 * (a) the desktop fixed aside (Sidebar above), and
 * (b) the mobile Sheet drawer (AppShell <lg).
 *
 * onNavigate fires after a nav link is clicked — used by the mobile drawer
 * to close itself on selection.
 */
export function SidebarContent({ onNavigate }: { onNavigate?: (() => void) | undefined } = {}) {
  const { theme } = useTheme()

  return (
    <>
      <div className="flex items-center gap-2.5 px-4 pt-4 pb-3">
        <img
          src={theme === 'dark' ? torqueLogoDark : torqueLogoLight}
          alt="Torque"
          className="h-7 w-auto"
        />
        <span className="font-metric text-2xs text-ink-dim ml-auto tabular-nums">v1.0</span>
      </div>

      <div className="px-3 pb-3">
        <OrgSwitcher />
      </div>

      <nav className="flex-1 overflow-y-auto px-2">
        <NavGroup label="Vendas">
          {vendasNav.map((item) => (
            <SidebarLink key={item.to} item={item} onNavigate={onNavigate} />
          ))}
        </NavGroup>
        <NavGroup label="Automacao">
          {automacaoNav.map((item) => (
            <SidebarLink key={item.to} item={item} onNavigate={onNavigate} />
          ))}
        </NavGroup>
        <NavGroup label="Inteligencia">
          {inteligenciaNav.map((item) => (
            <SidebarLink key={item.to} item={item} onNavigate={onNavigate} />
          ))}
        </NavGroup>
        <NavGroup label="Equipe">
          {equipeNav.map((item) => (
            <SidebarLink key={item.to} item={item} onNavigate={onNavigate} />
          ))}
        </NavGroup>
      </nav>

      <div className="px-2 py-3 shadow-[inset_0_1px_0_0_hsl(var(--hairline)/0.5)]">
        {footerNav.map((item) => (
          <SidebarLink key={item.to} item={item} onNavigate={onNavigate} />
        ))}
      </div>
    </>
  )
}

function NavGroup({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="mb-6">
      <div className="text-2xs text-ink-dim px-2 pt-2 pb-1.5 font-medium tracking-[0.14em] uppercase">
        {label}
      </div>
      <ul className="space-y-1">{children}</ul>
    </div>
  )
}

function SidebarLink({ item, onNavigate }: { item: NavItem; onNavigate?: (() => void) | undefined }) {
  const Icon = item.icon
  return (
    <li>
      <Tooltip content={item.label} shortcut={item.shortcut} side="right">
        <NavLink
          to={item.to}
          end={item.to === '/'}
          onClick={onNavigate}
          className={({ isActive }) =>
            cn(
              // S56 — h-11 (44px) on touch viewports, compact 36px from lg up.
              'group relative flex h-11 items-center gap-2.5 rounded-sm px-2 text-[0.8125rem] lg:h-9',
              'transition-colors duration-150',
              isActive
                ? 'bg-elevated text-ink shadow-hairline'
                : 'text-ink-muted hover:bg-elevated/50 hover:text-ink'
            )
          }
        >
          {({ isActive }) => (
            <>
              {isActive && (
                <span className="bg-accent absolute top-1/2 left-0 h-4 w-[2px] -translate-y-1/2 rounded-full" />
              )}
              <Icon
                className={cn(
                  'h-4 w-4 shrink-0',
                  isActive ? 'text-accent' : 'text-ink-dim group-hover:text-ink-muted'
                )}
                strokeWidth={1.75}
              />
              <span className="flex-1 truncate">{item.label}</span>
              {item.badge && (
                <span className="font-metric text-2xs text-ink-dim tabular-nums">{item.badge}</span>
              )}
            </>
          )}
        </NavLink>
      </Tooltip>
    </li>
  )
}
