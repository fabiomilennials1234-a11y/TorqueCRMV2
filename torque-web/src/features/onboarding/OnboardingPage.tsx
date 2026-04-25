/**
 * F13 Onboarding wizard.
 *
 * Linear flow: for each canonical step the user sees a short description
 * and a primary action to mark it done. Reaching `finish` stamps
 * completed_at server-side and the gate releases the user to the product.
 *
 * The step labels and copy live entirely client-side — the server is
 * authoritative about which step keys exist; labels are presentational.
 */

import { Check } from 'lucide-react'
import { Navigate } from 'react-router-dom'

import { Button } from '@/ui/button'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { StepProgress } from '@/ui/step-progress'
import {
  useCompleteStep,
  useDismissOnboarding,
  useOnboarding,
  type OnboardingStatus,
} from '@/hooks/useOnboarding'

interface StepCopy {
  title: string
  description: string
  action: string
}

const STEP_COPY: Record<string, StepCopy> = {
  welcome: {
    title: 'Boas-vindas ao Torque',
    description:
      'Você configurou uma organização. Vamos passar por 5 passos rápidos para deixar o time pronto para operar.',
    action: 'Começar',
  },
  organization_profile: {
    title: 'Perfil da organização',
    description:
      'Preencha razão social, fuso horário e um logo em Configurações → Organização para relatórios e notificações externas.',
    action: 'Feito',
  },
  invite_team: {
    title: 'Convide seu time',
    description:
      'Adicione seus SDRs e closers na aba Equipe. Admins gerenciam permissões finas por membro.',
    action: 'Feito',
  },
  connect_whatsapp: {
    title: 'Conecte o WhatsApp',
    description:
      'Ligue sua instância da Evolution API em Integrações. Sem isso o Inbox e as Campanhas não disparam.',
    action: 'Feito',
  },
  create_first_lead: {
    title: 'Crie seu primeiro lead',
    description:
      'Importe leads via webhook ou crie manualmente no funil. Uma vez que haja ao menos um, o funil ganha sentido.',
    action: 'Feito',
  },
  finish: {
    title: 'Tudo pronto',
    description: 'Sua organização está operacional. Bem-vindo ao Torque.',
    action: 'Entrar no produto',
  },
}

export function OnboardingPage() {
  const status = useOnboarding()
  const completeStep = useCompleteStep()
  const dismiss = useDismissOnboarding()

  if (status.isLoading) {
    return (
      <div className="mx-auto mt-24 max-w-xl space-y-4 px-8">
        <Skeleton className="h-6 w-1/2" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!status.data) {
    return (
      <div className="text-ink-muted mx-auto mt-24 max-w-xl px-8 text-sm">
        Não foi possível carregar o onboarding.
      </div>
    )
  }

  const s: OnboardingStatus = status.data

  // Gate is decided elsewhere (OnboardingGate); if the user lands here after
  // completing, push them to the dashboard.
  if (s.completed_at) {
    return <Navigate to="/" replace />
  }

  const currentIndex = s.all_steps.findIndex((k) => k === s.current_step)
  const completedIndexes = s.steps_completed
    .map((k) => s.all_steps.findIndex((x) => x === k))
    .filter((i) => i >= 0)
  const copy = STEP_COPY[s.current_step] ?? {
    title: s.current_step,
    description: 'Prossiga para o próximo passo.',
    action: 'Feito',
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-3xl flex-col justify-center px-8 py-16">
      <PageHeader eyebrow="Primeiros passos" title="Vamos preparar sua organização" />

      <div className="mt-10">
        <StepProgress
          steps={s.all_steps.map(humanize)}
          currentStep={currentIndex < 0 ? 0 : currentIndex}
          completedSteps={completedIndexes}
          variant="numbered"
        />
      </div>

      <div className="bg-surface shadow-elev-1 mt-12 rounded-lg p-8">
        <div className="text-2xs text-ink-dim mb-2 inline-flex items-center gap-2 tracking-[0.12em] uppercase">
          Passo {currentIndex + 1} de {s.all_steps.length}
        </div>
        <h2 className="font-display text-ink text-2xl">{copy.title}</h2>
        <p className="text-ink-muted mt-3 max-w-prose text-sm">{copy.description}</p>

        <div className="mt-8 flex flex-wrap items-center gap-3">
          <Button
            variant="primary"
            size="md"
            disabled={completeStep.isPending}
            onClick={() => void completeStep.mutateAsync({ step: s.current_step })}
          >
            {completeStep.isPending ? 'Salvando…' : copy.action}
            {!completeStep.isPending && <Check className="ml-1.5 h-3.5 w-3.5" />}
          </Button>
          <Button
            variant="ghost"
            size="md"
            disabled={dismiss.isPending}
            onClick={() => void dismiss.mutateAsync()}
          >
            Pular onboarding
          </Button>
        </div>
      </div>
    </div>
  )
}

function humanize(stepKey: string): string {
  switch (stepKey) {
    case 'welcome':
      return 'Boas-vindas'
    case 'organization_profile':
      return 'Perfil'
    case 'invite_team':
      return 'Equipe'
    case 'connect_whatsapp':
      return 'WhatsApp'
    case 'create_first_lead':
      return 'Primeiro lead'
    case 'finish':
      return 'Finalizar'
    default:
      return stepKey
  }
}
