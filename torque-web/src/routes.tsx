import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppShell } from '@/shell/AppShell'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { ManagerModeGate } from '@/components/ManagerModeGate'
import { CockpitShell } from '@/features/cockpit/CockpitShell'
import { CockpitView } from '@/features/cockpit/CockpitView'
import { LoginPage } from '@/features/auth/LoginPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { KanbanPage } from '@/features/pipeline/KanbanPage'
import { InboxPage } from '@/features/inbox/InboxPage'
import { WorkflowBuilderPage } from '@/features/workflows/WorkflowBuilderPage'
import { CampaignsPage } from '@/features/campaigns/CampaignsPage'
import { AgentsPage } from '@/features/copilot/AgentsPage'
import { AnalyticsPage } from '@/features/analytics/AnalyticsPage'
import { CheckoutPage } from '@/features/billing/CheckoutPage'
import { OnboardingPage } from '@/features/onboarding/OnboardingPage'
import { OnboardingGate } from '@/components/OnboardingGate'
import { ProductsPage } from '@/features/products/ProductsPage'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { NotFoundPage } from '@/features/errors/NotFoundPage'
import { ForbiddenPage } from '@/features/errors/ForbiddenPage'
import { EmptyState } from '@/ui/empty-state'
import { Construction } from 'lucide-react'

// ---------------------------------------------------------------------------
// Placeholder for routes under construction
// ---------------------------------------------------------------------------

function ComingSoonPage() {
  return (
    <div className="flex flex-1 items-center justify-center py-16">
      <EmptyState
        icon={Construction}
        title="Em breve"
        description="Esta area esta sendo construida."
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      // Onboarding wizard — fora do AppShell, sem gate de modo.
      {
        path: 'onboarding',
        element: <OnboardingPage />,
      },
      // Modo Vendedor — layout próprio (sem AppShell)
      {
        path: 'cockpit',
        element: <OnboardingGate />,
        children: [
          {
            element: <CockpitShell />,
            children: [{ index: true, element: <CockpitView /> }],
          },
        ],
      },
      // Modo Gerente — AppShell + gate de permissão + gate de onboarding
      {
        element: <OnboardingGate />,
        children: [{
        element: <ManagerModeGate />,
        children: [
          {
            element: <AppShell />,
            children: [
              { index: true, element: <DashboardPage /> },
              { path: 'pipeline', element: <KanbanPage /> },
              { path: 'inbox', element: <InboxPage /> },
              { path: 'workflows', element: <WorkflowBuilderPage /> },
              { path: 'campaigns', element: <CampaignsPage /> },
              { path: 'copilot', element: <AgentsPage /> },
              { path: 'analytics', element: <AnalyticsPage /> },
              { path: 'settings', element: <SettingsPage /> },
              { path: 'billing', element: <CheckoutPage /> },
              { path: 'help', element: <Navigate to="/settings" replace /> },
              { path: 'follow-ups', element: <ComingSoonPage /> },
              { path: 'team', element: <ComingSoonPage /> },
              { path: 'products', element: <ProductsPage /> },
              { path: 'forbidden', element: <ForbiddenPage /> },
              { path: '*', element: <NotFoundPage /> },
            ],
          },
        ],
        }],
      },
    ],
  },
])
