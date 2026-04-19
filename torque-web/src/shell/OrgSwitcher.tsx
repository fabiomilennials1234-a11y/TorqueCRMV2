import { ChevronsUpDown, Check, Plus } from 'lucide-react'
import { useState } from 'react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/dropdown'
import { cn } from '@/lib/utils'

const orgs = [
  { id: 'o1', name: 'Siderúrgica Aurora', plan: 'Growth', initial: 'SA' },
  { id: 'o2', name: 'Distribuidora Vértice', plan: 'Scale', initial: 'DV' },
  { id: 'o3', name: 'Metalúrgica Côndor', plan: 'Starter', initial: 'MC' },
]

export function OrgSwitcher() {
  const [active, setActive] = useState(orgs[0]!)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          'group flex w-full items-center gap-2.5 rounded-md bg-elevated px-2.5 py-2',
          'shadow-hairline hover:bg-elevated/80 focus-visible:outline-none',
          'text-left transition-colors'
        )}
      >
        <div className="font-metric flex h-7 w-7 shrink-0 items-center justify-center rounded-sm bg-accent/15 text-[0.7rem] font-medium text-accent shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.3)]">
          {active.initial}
        </div>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[0.8125rem] font-medium text-ink">{active.name}</div>
          <div className="text-2xs uppercase tracking-[0.1em] text-ink-dim">
            Plano {active.plan}
          </div>
        </div>
        <ChevronsUpDown className="h-3.5 w-3.5 shrink-0 text-ink-dim group-hover:text-ink-muted" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[260px]">
        <DropdownMenuLabel>Organizações</DropdownMenuLabel>
        {orgs.map((org) => (
          <DropdownMenuItem
            key={org.id}
            onSelect={() => setActive(org)}
            className="flex items-center gap-2.5"
          >
            <div className="font-metric flex h-6 w-6 shrink-0 items-center justify-center rounded-sm bg-elevated text-[0.65rem] text-ink-muted shadow-hairline">
              {org.initial}
            </div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-[0.8125rem] text-ink">{org.name}</div>
              <div className="text-2xs uppercase tracking-[0.1em] text-ink-dim">{org.plan}</div>
            </div>
            {active.id === org.id && <Check className="h-3.5 w-3.5 text-accent" />}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem className="gap-2.5">
          <div className="flex h-6 w-6 items-center justify-center rounded-sm bg-elevated shadow-hairline">
            <Plus className="h-3.5 w-3.5" />
          </div>
          <span>Nova organização</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
