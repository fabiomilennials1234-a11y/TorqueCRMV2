import * as RD from '@radix-ui/react-dialog'
import { Command } from 'cmdk'
import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Columns3,
  MessagesSquare,
  LayoutDashboard,
  Workflow,
  Megaphone,
  Sparkles,
  BarChart3,
  Settings2,
  Plus,
  User,
  Tag,
  Moon,
  ArrowRight,
} from 'lucide-react'
import { Kbd } from '@/ui/kbd'
import { cn } from '@/lib/utils'

interface Props {
  open: boolean
  onOpenChange: (v: boolean) => void
}

const nav = [
  { label: 'Visão geral', icon: LayoutDashboard, to: '/' },
  { label: 'Funis', icon: Columns3, to: '/pipeline' },
  { label: 'Conversas', icon: MessagesSquare, to: '/inbox' },
  { label: 'Fluxos', icon: Workflow, to: '/workflows' },
  { label: 'Campanhas', icon: Megaphone, to: '/campaigns' },
  { label: 'Agentes IA', icon: Sparkles, to: '/copilot' },
  { label: 'Analytics', icon: BarChart3, to: '/analytics' },
  { label: 'Configurações', icon: Settings2, to: '/settings' },
]

const actions = [
  { label: 'Novo lead', icon: Plus, shortcut: 'N' },
  { label: 'Novo contato', icon: User },
  { label: 'Adicionar tag', icon: Tag },
  { label: 'Alternar tema', icon: Moon, shortcut: '⌘ ⇧ L' },
]

const recent = [
  'Lucas Arantes — Metalúrgica Kaizen',
  'Aline Pereira — Tecnoferro SP',
  'Bruno Tavares — Distribuidora Horizonte',
  'Camila Esteves — Tubos Andrade',
]

export function CommandPalette({ open, onOpenChange }: Props) {
  const navigate = useNavigate()

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        onOpenChange(!open)
      }
      if (e.key === 'Escape' && open) onOpenChange(false)
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onOpenChange])

  return (
    <RD.Root open={open} onOpenChange={onOpenChange}>
      <RD.Portal>
        <RD.Overlay className="bg-bg/60 data-[state=open]:animate-fade-in fixed inset-0 z-50 backdrop-blur-[16px]" />
        <RD.Content
          className={cn(
            'fixed top-[22%] left-1/2 z-50 w-full max-w-[640px] -translate-x-1/2',
            'bg-elevated/90 shadow-elev-3 shadow-hairline overflow-hidden rounded-lg backdrop-blur-xl',
            'data-[state=open]:animate-scale-in'
          )}
        >
          <RD.Title className="sr-only">Command palette</RD.Title>
          <Command loop className="flex flex-col">
            <div className="shadow-hairline-b flex items-center gap-3 px-4 pt-4 pb-3">
              <div className="bg-accent h-1.5 w-1.5 animate-pulse rounded-full" />
              <Command.Input
                // Command palette is modal and opens via keyboard shortcut —
                // autofocus is the primary UX contract here.
                // eslint-disable-next-line jsx-a11y/no-autofocus
                autoFocus
                placeholder="Busque ou digite um comando…"
                className="font-display text-ink placeholder:text-ink-dim flex-1 bg-transparent text-[0.9375rem] tracking-tight focus:outline-none"
              />
              <Kbd>ESC</Kbd>
            </div>

            <Command.List className="max-h-[420px] overflow-y-auto p-2">
              <Command.Empty className="text-ink-dim px-3 py-8 text-center text-sm">
                Nenhum resultado.
              </Command.Empty>

              <Command.Group
                heading="Navegação"
                className="[&_[cmdk-group-heading]]:text-2xs [&_[cmdk-group-heading]]:text-ink-dim [&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:pt-2 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:tracking-[0.14em] [&_[cmdk-group-heading]]:uppercase"
              >
                {nav.map((item) => {
                  const Icon = item.icon
                  return (
                    <Command.Item
                      key={item.to}
                      onSelect={() => {
                        void navigate(item.to)
                        onOpenChange(false)
                      }}
                      className="group text-ink-muted data-[selected=true]:bg-surface data-[selected=true]:text-ink flex cursor-pointer items-center gap-3 rounded-sm px-2 py-2 text-sm"
                    >
                      <Icon
                        className="text-ink-dim group-data-[selected=true]:text-accent h-4 w-4"
                        strokeWidth={1.75}
                      />
                      <span className="flex-1">{item.label}</span>
                      <ArrowRight className="text-ink-dim h-3 w-3 opacity-0 group-data-[selected=true]:opacity-100" />
                    </Command.Item>
                  )
                })}
              </Command.Group>

              <Command.Group
                heading="Ações rápidas"
                className="[&_[cmdk-group-heading]]:text-2xs [&_[cmdk-group-heading]]:text-ink-dim [&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:tracking-[0.14em] [&_[cmdk-group-heading]]:uppercase"
              >
                {actions.map((a) => {
                  const Icon = a.icon
                  return (
                    <Command.Item
                      key={a.label}
                      className="text-ink-muted data-[selected=true]:bg-surface data-[selected=true]:text-ink flex cursor-pointer items-center gap-3 rounded-sm px-2 py-2 text-sm"
                    >
                      <Icon className="text-ink-dim h-4 w-4" strokeWidth={1.75} />
                      <span className="flex-1">{a.label}</span>
                      {a.shortcut && <Kbd>{a.shortcut}</Kbd>}
                    </Command.Item>
                  )
                })}
              </Command.Group>

              <Command.Group
                heading="Leads recentes"
                className="[&_[cmdk-group-heading]]:text-2xs [&_[cmdk-group-heading]]:text-ink-dim [&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:pt-3 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:tracking-[0.14em] [&_[cmdk-group-heading]]:uppercase"
              >
                {recent.map((name) => (
                  <Command.Item
                    key={name}
                    className="text-ink-muted data-[selected=true]:bg-surface data-[selected=true]:text-ink flex cursor-pointer items-center gap-3 rounded-sm px-2 py-2 text-sm"
                  >
                    <div className="font-metric bg-accent/10 text-accent flex h-5 w-5 items-center justify-center rounded-sm text-[0.6rem]">
                      {name.slice(0, 2).toUpperCase()}
                    </div>
                    <span className="flex-1 truncate">{name}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            </Command.List>

            <div className="text-2xs text-ink-dim flex items-center gap-3 px-3 py-2 shadow-[inset_0_1px_0_0_hsl(var(--hairline))]">
              <div className="flex items-center gap-1.5">
                <Kbd>↑</Kbd>
                <Kbd>↓</Kbd>
                <span>navegar</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Kbd>↵</Kbd>
                <span>abrir</span>
              </div>
              <div className="ml-auto flex items-center gap-1.5">
                <span>powered by</span>
                <span className="font-metric text-ink-muted">Torque Command</span>
              </div>
            </div>
          </Command>
        </RD.Content>
      </RD.Portal>
    </RD.Root>
  )
}
