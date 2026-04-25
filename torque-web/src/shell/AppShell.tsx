import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import { TooltipProvider } from '@/ui/tooltip'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { CommandPalette } from './CommandPalette'

export function AppShell() {
  const [commandOpen, setCommandOpen] = useState(false)

  return (
    <TooltipProvider delayDuration={200} skipDelayDuration={300}>
      <div className="bg-bg text-ink flex h-screen w-screen overflow-hidden">
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <TopBar onOpenCommand={() => setCommandOpen(true)} />
          <main className="flex-1 overflow-y-auto">
            <Outlet />
          </main>
        </div>
        <CommandPalette open={commandOpen} onOpenChange={setCommandOpen} />
      </div>
    </TooltipProvider>
  )
}
