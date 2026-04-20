import { Suspense, lazy } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'

import { ProtectedRoute } from '@/components/ProtectedRoute'
import { ManagerModeGate } from '@/components/ManagerModeGate'
import { OnboardingGate } from '@/components/OnboardingGate'
import { RouteSkeleton } from '@/shell/RouteSkeleton'
import { EmptyState } from '@/ui/empty-state'
import { lazyRetry } from '@/lib/lazyRetry'
import { Construction } from 'lucide-react'

// ---------------------------------------------------------------------------
// Lazy routes
// ---------------------------------------------------------------------------
//
// All page-level components are dynamically imported through `lazyRetry` so
// the initial bundle stays minimal and a stale chunk right after a deploy
// retries with exponential backoff before surfacing to the user.
//
// The shells (`AppShell`, `CockpitShell`) are also lazy — they carry enough
// sidebar + provider weight that splitting them saves ~40 KB on the login
// path.
// ---------------------------------------------------------------------------

const AppShell = lazy(() => lazyRetry(() => import('@/shell/AppShell').then((m) => ({ default: m.AppShell }))))
const CockpitShell = lazy(() => lazyRetry(() => import('@/features/cockpit/CockpitShell').then((m) => ({ default: m.CockpitShell }))))
const CockpitView = lazy(() => lazyRetry(() => import('@/features/cockpit/CockpitView').then((m) => ({ default: m.CockpitView }))))

const LoginPage = lazy(() => lazyRetry(() => import('@/features/auth/LoginPage').then((m) => ({ default: m.LoginPage }))))
const DashboardPage = lazy(() => lazyRetry(() => import('@/features/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage }))))
const KanbanPage = lazy(() => lazyRetry(() => import('@/features/pipeline/KanbanPage').then((m) => ({ default: m.KanbanPage }))))
const InboxPage = lazy(() => lazyRetry(() => import('@/features/inbox/InboxPage').then((m) => ({ default: m.InboxPage }))))
const WorkflowBuilderPage = lazy(() => lazyRetry(() => import('@/features/workflows/WorkflowBuilderPage').then((m) => ({ default: m.WorkflowBuilderPage }))))
const CampaignsPage = lazy(() => lazyRetry(() => import('@/features/campaigns/CampaignsPage').then((m) => ({ default: m.CampaignsPage }))))
const AgentListPage = lazy(() => lazyRetry(() => import('@/features/copilot/AgentListPage').then((m) => ({ default: m.AgentListPage }))))
const AgentPlaygroundPage = lazy(() => lazyRetry(() => import('@/features/copilot/AgentPlaygroundPage').then((m) => ({ default: m.AgentPlaygroundPage }))))
const AnalyticsPage = lazy(() => lazyRetry(() => import('@/features/analytics/AnalyticsPage').then((m) => ({ default: m.AnalyticsPage }))))
const SettingsPage = lazy(() => lazyRetry(() => import('@/features/settings/SettingsPage').then((m) => ({ default: m.SettingsPage }))))
const CheckoutPage = lazy(() => lazyRetry(() => import('@/features/billing/CheckoutPage').then((m) => ({ default: m.CheckoutPage }))))
const MasterPage = lazy(() => lazyRetry(() => import('@/features/master/MasterPage').then((m) => ({ default: m.MasterPage }))))
const OnboardingPage = lazy(() => lazyRetry(() => import('@/features/onboarding/OnboardingPage').then((m) => ({ default: m.OnboardingPage }))))
const ProductsPage = lazy(() => lazyRetry(() => import('@/features/products/ProductsPage').then((m) => ({ default: m.ProductsPage }))))
const NotFoundPage = lazy(() => lazyRetry(() => import('@/features/errors/NotFoundPage').then((m) => ({ default: m.NotFoundPage }))))
const ForbiddenPage = lazy(() => lazyRetry(() => import('@/features/errors/ForbiddenPage').then((m) => ({ default: m.ForbiddenPage }))))

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

// Every lazy route is rendered inside a single `<Suspense>` at the
// top of the authenticated tree, so the skeleton flashes once per
// navigation instead of per-nested-route.
function withSuspense(node: React.ReactNode) {
  return <Suspense fallback={<RouteSkeleton />}>{node}</Suspense>
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

export const router = createBrowserRouter([
  {
    path: '/login',
    element: withSuspense(<LoginPage />),
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      // Onboarding wizard — fora do AppShell, sem gate de modo.
      {
        path: 'onboarding',
        element: withSuspense(<OnboardingPage />),
      },
      // Modo Vendedor — layout próprio (sem AppShell)
      {
        path: 'cockpit',
        element: <OnboardingGate />,
        children: [
          {
            element: withSuspense(<CockpitShell />),
            children: [{ index: true, element: withSuspense(<CockpitView />) }],
          },
        ],
      },
      // Modo Gerente — AppShell + gate de permissão + gate de onboarding
      {
        element: <OnboardingGate />,
        children: [
          {
            element: <ManagerModeGate />,
            children: [
              {
                element: withSuspense(<AppShell />),
                children: [
                  { index: true, element: withSuspense(<DashboardPage />) },
                  { path: 'pipeline', element: withSuspense(<KanbanPage />) },
                  { path: 'inbox', element: withSuspense(<InboxPage />) },
                  { path: 'workflows', element: withSuspense(<WorkflowBuilderPage />) },
                  { path: 'campaigns', element: withSuspense(<CampaignsPage />) },
                  { path: 'copilot', element: withSuspense(<AgentListPage />) },
                  { path: 'copilot/:id', element: withSuspense(<AgentPlaygroundPage />) },
                  { path: 'analytics', element: withSuspense(<AnalyticsPage />) },
                  { path: 'settings', element: withSuspense(<SettingsPage />) },
                  { path: 'billing', element: withSuspense(<CheckoutPage />) },
                  { path: 'master', element: withSuspense(<MasterPage />) },
                  { path: 'products', element: withSuspense(<ProductsPage />) },
                  { path: 'help', element: <Navigate to="/settings" replace /> },
                  { path: 'follow-ups', element: <ComingSoonPage /> },
                  { path: 'team', element: <ComingSoonPage /> },
                  { path: 'forbidden', element: withSuspense(<ForbiddenPage />) },
                  { path: '*', element: withSuspense(<NotFoundPage />) },
                ],
              },
            ],
          },
        ],
      },
    ],
  },
])
