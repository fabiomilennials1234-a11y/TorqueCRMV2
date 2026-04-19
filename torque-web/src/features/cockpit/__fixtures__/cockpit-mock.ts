import type { LeadSummary, PipeStageSnapshot, Task, TaskCockpitBundle } from '@/contracts/manual'

const ORG = 'org_mock_001'
const ME = 'mbr_mock_001'
const NOW = new Date()

function minutesAgo(min: number): string {
  return new Date(NOW.getTime() - min * 60_000).toISOString()
}
function hoursAhead(h: number): string {
  return new Date(NOW.getTime() + h * 3_600_000).toISOString()
}
function hoursAgo(h: number): string {
  return new Date(NOW.getTime() - h * 3_600_000).toISOString()
}

function makeTask(partial: Partial<Task> & Pick<Task, 'id' | 'title'>): Task {
  return {
    organizationId: ORG,
    leadId: null,
    assignedTo: ME,
    createdBy: 'mbr_mgr_007',
    kind: 'generic',
    description: null,
    priority: 'normal',
    status: 'pending',
    inQueue: false,
    queuePosition: null,
    dueAt: null,
    startedAt: null,
    completedAt: null,
    completedBy: null,
    cancelledAt: null,
    cancelledReason: null,
    missedReason: null,
    origin: 'manual',
    context: null,
    resultNote: null,
    createdAt: minutesAgo(240),
    updatedAt: minutesAgo(30),
    ...partial,
  }
}

const ACTIVE_LEAD: LeadSummary = {
  id: 'lead_mock_042',
  name: 'Marina Falcão · Construtora Atlas',
  phone: '+55 11 98444-1223',
  pipeId: 'pipe_whatsapp',
  pipeName: 'Pipe WhatsApp',
  stageId: 'stage_3',
  stageName: 'Qualificado',
  heat: 4,
  channel: 'whatsapp',
}

export const PIPE_STAGES: PipeStageSnapshot[] = [
  {
    id: 'stage_1',
    name: 'Novo',
    position: 1,
    leadCount: 38,
    colorToken: 'stage-1',
    isActive: false,
  },
  {
    id: 'stage_2',
    name: 'Em contato',
    position: 2,
    leadCount: 22,
    colorToken: 'stage-2',
    isActive: false,
  },
  {
    id: 'stage_3',
    name: 'Qualificado',
    position: 3,
    leadCount: 14,
    colorToken: 'stage-3',
    isActive: true,
  },
  {
    id: 'stage_4',
    name: 'Agendado',
    position: 4,
    leadCount: 9,
    colorToken: 'stage-4',
    isActive: false,
  },
  {
    id: 'stage_5',
    name: 'Proposta',
    position: 5,
    leadCount: 6,
    colorToken: 'stage-5',
    isActive: false,
  },
  {
    id: 'stage_6',
    name: 'Fechamento',
    position: 6,
    leadCount: 3,
    colorToken: 'stage-6',
    isActive: false,
  },
]

const IN_PROGRESS: Task = makeTask({
  id: 'task_active_001',
  leadId: ACTIVE_LEAD.id,
  kind: 'qualification',
  title: 'Qualificar Marina — entender porte do projeto',
  description:
    'Marina respondeu ao anúncio da campanha outbound com interesse em orçamento para condomínio residencial de 120 unidades. Confirmar: metragem total, prazo de entrega, cidade, orçamento alocado, decisor.',
  priority: 'high',
  status: 'in_progress',
  inQueue: false,
  queuePosition: null,
  dueAt: hoursAhead(2),
  startedAt: minutesAgo(6),
  origin: 'manual',
})

const QUEUE: Task[] = [
  makeTask({
    id: 'task_q_001',
    leadId: 'lead_101',
    kind: 'call',
    title: 'Ligar para Rafael (Eletro Norte) após almoço',
    priority: 'urgent',
    inQueue: true,
    queuePosition: 1,
    dueAt: hoursAhead(1),
  }),
  makeTask({
    id: 'task_q_002',
    leadId: 'lead_102',
    kind: 'send_proposal',
    title: 'Enviar proposta revisada — Studio Vértice',
    description: 'Inclui desconto de 8% e prazo estendido a pedido do cliente.',
    priority: 'high',
    inQueue: true,
    queuePosition: 2,
    dueAt: hoursAhead(4),
  }),
  makeTask({
    id: 'task_q_003',
    leadId: 'lead_103',
    kind: 'followup',
    title: 'Retomar contato com Douglas (Cimento Leste)',
    priority: 'normal',
    inQueue: true,
    queuePosition: 3,
    dueAt: hoursAhead(22),
  }),
  makeTask({
    id: 'task_q_004',
    leadId: 'lead_104',
    kind: 'confirm_meeting',
    title: 'Confirmar reunião de amanhã 14h — Helena',
    priority: 'high',
    inQueue: true,
    queuePosition: 4,
    dueAt: hoursAhead(26),
  }),
  makeTask({
    id: 'task_q_005',
    leadId: 'lead_105',
    kind: 'objection',
    title: 'Responder objeção de preço — Vitor (Metal Sul)',
    priority: 'normal',
    inQueue: true,
    queuePosition: 5,
    dueAt: hoursAhead(28),
  }),
]

const BACKLOG: Task[] = [
  makeTask({
    id: 'task_b_001',
    leadId: 'lead_201',
    kind: 'followup',
    title: 'Aquecer lead frio — Luciana (Construtora Planalto)',
    priority: 'low',
    dueAt: hoursAhead(72),
  }),
  makeTask({
    id: 'task_b_002',
    leadId: 'lead_202',
    kind: 'call',
    title: 'Tentar segundo contato — Eduardo (Indústria Apex)',
    priority: 'normal',
    dueAt: hoursAhead(48),
  }),
  makeTask({
    id: 'task_b_003',
    leadId: 'lead_203',
    kind: 'qualification',
    title: 'Qualificar Sophia — veio do Instagram',
    priority: 'high',
    dueAt: hoursAhead(36),
  }),
  makeTask({
    id: 'task_b_004',
    leadId: 'lead_204',
    kind: 'send_proposal',
    title: 'Revisar proposta antes de enviar — Oliveira & Filhos',
    priority: 'normal',
    dueAt: hoursAhead(18),
  }),
  makeTask({
    id: 'task_b_005',
    leadId: 'lead_205',
    kind: 'generic',
    title: 'Atualizar dados cadastrais — Ferraz Metalúrgica',
    priority: 'low',
  }),
  makeTask({
    id: 'task_b_006',
    leadId: 'lead_206',
    kind: 'followup',
    title: 'Retorno prometido para quinta — Ana (Clínica Horizonte)',
    priority: 'high',
    dueAt: hoursAhead(56),
  }),
  makeTask({
    id: 'task_b_007',
    leadId: 'lead_207',
    kind: 'call',
    title: 'Ligar no fim do dia — Bruno (Papéis Reais)',
    priority: 'normal',
    dueAt: hoursAhead(10),
  }),
]

const MISSED: Task[] = [
  makeTask({
    id: 'task_m_001',
    leadId: 'lead_301',
    kind: 'followup',
    title: 'Recuperar atraso — Cesar (Transportadora Lima)',
    priority: 'urgent',
    status: 'missed',
    missedReason: 'SLA vencido sem ação',
    dueAt: hoursAgo(8),
  }),
]

export const COCKPIT_MOCK: TaskCockpitBundle = {
  inProgress: IN_PROGRESS,
  queue: QUEUE,
  backlog: BACKLOG,
  missed: MISSED,
  activeLead: ACTIVE_LEAD,
  pipeStages: PIPE_STAGES,
  counts: {
    queue: QUEUE.length,
    backlog: BACKLOG.length,
    missed: MISSED.length,
    completedToday: 7,
  },
}

export const COCKPIT_ME_MEMBER_ID = ME
