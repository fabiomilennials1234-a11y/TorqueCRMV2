import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { TooltipProvider } from '@/ui/tooltip'
import { Sheet, SheetContent } from '@/ui/sheet'
import { Sidebar, SidebarContent } from './Sidebar'
import { TopBar } from './TopBar'
import { CommandPalette } from './CommandPalette'

export function AppShell() {
  const [commandOpen, setCommandOpen] = useState(false)
  // S56 — mobile sidebar drawer state. Closed by default; controlled at shell level so
  // TopBar hamburger can open and a route change auto-closes (matches expected mobile UX).
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const location = useLocation()

  useEffect(() => {
    setSidebarOpen(false)
  }, [location.pathname])

  return (
    <TooltipProvider delayDuration={200} skipDelayDuration={300}>
      <div className="bg-bg text-ink flex h-[100dvh] w-screen overflow-hidden">
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <TopBar
            onOpenCommand={() => setCommandOpen(true)}
            onOpenSidebar={() => setSidebarOpen(true)}
          />
          <main className="flex-1 overflow-y-auto">
            <Outlet />
          </main>
        </div>
        <CommandPalette open={commandOpen} onOpenChange={setCommandOpen} />

        {/* S56 — mobile-only drawer. lg:hidden suppresses if user resizes during open state. */}
        <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
          <SheetContent
            side="left"
            className="bg-surface flex w-[280px] flex-col p-0 lg:hidden"
            aria-label="Menu de navegacao"
          >
            <SidebarContent onNavigate={() => setSidebarOpen(false)} />
          </SheetContent>
        </Sheet>
      </div>
    </TooltipProvider>
  )
}
