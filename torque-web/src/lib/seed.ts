/**
 * Seed data for visual fidelity. All static, no backend. PT-BR industrial context.
 */

export type Stage =
  | 'novo'
  | 'abordado'
  | 'qualificado'
  | 'agendado'
  | 'proposta'
  | 'vendido'
  | 'perdido'

export const stageMeta: Record<Stage, { label: string; color: string; order: number }> = {
  novo: { label: 'Novo', color: 'hsl(var(--stage-1))', order: 1 },
  abordado: { label: 'Abordado', color: 'hsl(var(--stage-2))', order: 2 },
  qualificado: { label: 'Qualificado', color: 'hsl(var(--stage-3))', order: 3 },
  agendado: { label: 'Agendado', color: 'hsl(var(--stage-4))', order: 4 },
  proposta: { label: 'Proposta', color: 'hsl(var(--stage-5))', order: 5 },
  vendido: { label: 'Vendido', color: 'hsl(var(--stage-6))', order: 6 },
  perdido: { label: 'Perdido', color: 'hsl(var(--stage-7))', order: 7 },
}

export type Lead = {
  id: string
  name: string
  company: string
  value: number
  score: number
  stage: Stage
  tags: string[]
  owner: { name: string; initials: string }
  lastTouch: Date
  channel: 'whatsapp' | 'messenger' | 'email'
  unread?: number
  source: string
  notesCount?: number
  tasksDue?: number
}

const daysAgo = (d: number) => new Date(Date.now() - d * 86_400_000)
const hoursAgo = (h: number) => new Date(Date.now() - h * 3_600_000)
const minAgo = (m: number) => new Date(Date.now() - m * 60_000)

export const leads: Lead[] = [
  {
    id: 'l1',
    name: 'Lucas Arantes',
    company: 'Metalúrgica Kaizen',
    value: 48_000,
    score: 82,
    stage: 'qualificado',
    tags: ['BANT ok', 'ICP fit'],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: minAgo(14),
    channel: 'whatsapp',
    source: 'Meta Ads',
    notesCount: 3,
    tasksDue: 1,
    unread: 2,
  },
  {
    id: 'l2',
    name: 'Aline Pereira',
    company: 'Tecnoferro SP',
    value: 120_000,
    score: 91,
    stage: 'proposta',
    tags: ['Hot', 'Decisor'],
    owner: { name: 'Rafael Bento', initials: 'RB' },
    lastTouch: hoursAgo(3),
    channel: 'whatsapp',
    source: 'Indicação',
    notesCount: 7,
    tasksDue: 0,
  },
  {
    id: 'l3',
    name: 'Bruno Tavares',
    company: 'Distribuidora Horizonte',
    value: 22_000,
    score: 64,
    stage: 'novo',
    tags: ['Inbound'],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: minAgo(3),
    channel: 'whatsapp',
    source: 'Formulário site',
    unread: 1,
  },
  {
    id: 'l4',
    name: 'Camila Esteves',
    company: 'Tubos Andrade',
    value: 86_500,
    score: 74,
    stage: 'agendado',
    tags: ['Reunião 18/04'],
    owner: { name: 'Rafael Bento', initials: 'RB' },
    lastTouch: hoursAgo(9),
    channel: 'whatsapp',
    source: 'Campanha Outbound',
    tasksDue: 1,
  },
  {
    id: 'l5',
    name: 'Diogo Nakamura',
    company: 'Forjaria Boreal',
    value: 34_000,
    score: 58,
    stage: 'abordado',
    tags: ['Aguardando', 'Follow-up'],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: daysAgo(1),
    channel: 'whatsapp',
    source: 'Meta Ads',
  },
  {
    id: 'l6',
    name: 'Érica Vilhena',
    company: 'Aços União',
    value: 210_000,
    score: 95,
    stage: 'proposta',
    tags: ['Enterprise', 'Hot'],
    owner: { name: 'Rafael Bento', initials: 'RB' },
    lastTouch: hoursAgo(1),
    channel: 'whatsapp',
    source: 'Evento',
    notesCount: 12,
    tasksDue: 2,
    unread: 3,
  },
  {
    id: 'l7',
    name: 'Fábio Tonin',
    company: 'Indústria Mercúrio',
    value: 15_000,
    score: 42,
    stage: 'abordado',
    tags: [],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: daysAgo(2),
    channel: 'messenger',
    source: 'Instagram',
  },
  {
    id: 'l8',
    name: 'Gabriela Luz',
    company: 'Plástica Litoral',
    value: 62_000,
    score: 79,
    stage: 'qualificado',
    tags: ['Tech fit'],
    owner: { name: 'Rafael Bento', initials: 'RB' },
    lastTouch: hoursAgo(5),
    channel: 'whatsapp',
    source: 'Indicação',
    notesCount: 2,
  },
  {
    id: 'l9',
    name: 'Henrique Seixas',
    company: 'Logis Transportes',
    value: 44_000,
    score: 68,
    stage: 'agendado',
    tags: ['Reunião amanhã'],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: hoursAgo(8),
    channel: 'whatsapp',
    source: 'Meta Ads',
    tasksDue: 1,
  },
  {
    id: 'l10',
    name: 'Isabela Ponte',
    company: 'Fundição Oceanus',
    value: 155_000,
    score: 88,
    stage: 'vendido',
    tags: ['Fechado', 'Q2'],
    owner: { name: 'Rafael Bento', initials: 'RB' },
    lastTouch: daysAgo(1),
    channel: 'whatsapp',
    source: 'Outbound',
  },
  {
    id: 'l11',
    name: 'João Menezes',
    company: 'Cerâmica Atlante',
    value: 28_000,
    score: 35,
    stage: 'perdido',
    tags: ['Sem orçamento'],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: daysAgo(3),
    channel: 'whatsapp',
    source: 'Formulário site',
  },
  {
    id: 'l12',
    name: 'Kauê Moretti',
    company: 'Tubos & Conexões BR',
    value: 73_000,
    score: 71,
    stage: 'novo',
    tags: ['Novo'],
    owner: { name: 'Maíra Duarte', initials: 'MD' },
    lastTouch: minAgo(28),
    channel: 'whatsapp',
    source: 'Meta Ads',
    unread: 5,
  },
]

export const kpis = [
  {
    label: 'Leads novos',
    value: 142,
    delta: 0.18,
    spark: [12, 19, 14, 18, 22, 17, 24, 20, 26, 23, 28, 31],
  },
  {
    label: 'Taxa de resposta',
    value: 68,
    suffix: '%',
    delta: 0.04,
    spark: [55, 58, 57, 60, 62, 61, 63, 64, 65, 66, 67, 68],
  },
  {
    label: 'Tempo médio 1ª resposta',
    value: 3.4,
    suffix: 'min',
    delta: -0.22,
    spark: [9, 8, 7, 6.5, 6, 5.4, 5, 4.5, 4, 3.8, 3.6, 3.4],
  },
  {
    label: 'Receita funil',
    value: 1_248_000,
    currency: true,
    delta: 0.09,
    spark: [800, 820, 870, 900, 930, 960, 1020, 1080, 1110, 1150, 1200, 1248],
  },
]

export const activity = [
  {
    at: minAgo(2),
    type: 'reply',
    who: 'Aline Pereira',
    text: 'respondeu com confirmação da reunião',
    meta: 'Tecnoferro SP · Proposta',
  },
  {
    at: minAgo(14),
    type: 'stage',
    who: 'Lucas Arantes',
    text: 'moveu para Qualificado',
    meta: 'por Maíra Duarte',
  },
  {
    at: minAgo(33),
    type: 'ai',
    who: 'Copilot · Mila',
    text: 'enviou apresentação e qualificou BANT',
    meta: 'Forjaria Boreal',
  },
  {
    at: hoursAgo(1),
    type: 'new',
    who: 'Érica Vilhena',
    text: 'entrou via Evento Expomáquinas',
    meta: 'Aços União · R$ 210k',
  },
  {
    at: hoursAgo(2),
    type: 'workflow',
    who: "Fluxo 'Aquecer após 72h'",
    text: 'disparou follow-up automático',
    meta: '3 leads afetados',
  },
  {
    at: hoursAgo(3),
    type: 'won',
    who: 'Isabela Ponte',
    text: 'fechou venda',
    meta: 'Fundição Oceanus · R$ 155k',
  },
]

export const conversations = [
  {
    id: 'c1',
    leadId: 'l6',
    name: 'Érica Vilhena',
    company: 'Aços União',
    preview: 'Perfeito, vou revisar o termo e te respondo ainda hoje.',
    at: minAgo(4),
    unread: 3,
    channel: 'whatsapp' as const,
    online: true,
  },
  {
    id: 'c2',
    leadId: 'l1',
    name: 'Lucas Arantes',
    company: 'Metalúrgica Kaizen',
    preview: 'Qual a capacidade mensal que vocês conseguem atender?',
    at: minAgo(14),
    unread: 2,
    channel: 'whatsapp' as const,
    online: true,
  },
  {
    id: 'c3',
    leadId: 'l12',
    name: 'Kauê Moretti',
    company: 'Tubos & Conexões BR',
    preview: 'Oi! Vi o anúncio. Trabalham com projeto sob medida?',
    at: minAgo(28),
    unread: 5,
    channel: 'whatsapp' as const,
    online: false,
  },
  {
    id: 'c4',
    leadId: 'l3',
    name: 'Bruno Tavares',
    company: 'Distribuidora Horizonte',
    preview: 'Pode me mandar um PDF com a ficha técnica?',
    at: hoursAgo(1),
    unread: 1,
    channel: 'whatsapp' as const,
    online: true,
  },
  {
    id: 'c5',
    leadId: 'l2',
    name: 'Aline Pereira',
    company: 'Tecnoferro SP',
    preview: 'Fechado. Nos falamos quinta às 10h.',
    at: hoursAgo(3),
    channel: 'whatsapp' as const,
    online: false,
  },
  {
    id: 'c6',
    leadId: 'l8',
    name: 'Gabriela Luz',
    company: 'Plástica Litoral',
    preview: 'Ficou dentro do orçamento. Vamos seguir.',
    at: hoursAgo(5),
    channel: 'whatsapp' as const,
    online: false,
  },
  {
    id: 'c7',
    leadId: 'l7',
    name: 'Fábio Tonin',
    company: 'Indústria Mercúrio',
    preview: 'Bom dia, gostaria de saber mais.',
    at: daysAgo(1),
    channel: 'messenger' as const,
    online: false,
  },
]

export const messages = [
  {
    id: 'm1',
    who: 'lead',
    name: 'Érica Vilhena',
    at: hoursAgo(2),
    text: 'Oi! Vi o seu material na Expomáquinas. Faz sentido marcarmos uma call rápida?',
  },
  {
    id: 'm2',
    who: 'agent',
    name: 'Mila (Copilot)',
    at: hoursAgo(2),
    text: 'Olá, Érica — que bom falar com você. Posso sim. Você tem preferência para amanhã ou sexta?',
    ai: true,
  },
  {
    id: 'm3',
    who: 'lead',
    name: 'Érica Vilhena',
    at: hoursAgo(1),
    text: 'Sexta de manhã seria ideal. 10h?',
  },
  {
    id: 'm4',
    who: 'agent',
    name: 'Rafael Bento',
    at: minAgo(42),
    text: 'Perfeito, Érica. Bloqueado na agenda. Vou te mandar o link do meet e uma prévia do escopo antes da conversa.',
  },
  {
    id: 'm5',
    who: 'lead',
    name: 'Érica Vilhena',
    at: minAgo(4),
    text: 'Perfeito, vou revisar o termo e te respondo ainda hoje.',
  },
]

export const workflowNodes = [
  { id: 'n1', type: 'trigger', title: 'Lead criado', x: 40, y: 60, note: 'Origem: Meta Ads' },
  { id: 'n2', type: 'condition', title: 'Score ≥ 60?', x: 280, y: 60 },
  {
    id: 'n3',
    type: 'action',
    title: 'Enviar mensagem inicial',
    x: 520,
    y: 20,
    note: 'Template · Boas-vindas',
  },
  { id: 'n4', type: 'wait', title: 'Aguardar 20min', x: 520, y: 140 },
  { id: 'n5', type: 'action', title: 'Atribuir SDR disponível', x: 760, y: 20 },
  { id: 'n6', type: 'action', title: "Arquivar com tag 'cold'", x: 760, y: 140 },
]

export const workflowEdges = [
  { from: 'n1', to: 'n2' },
  { from: 'n2', to: 'n3', label: 'sim' },
  { from: 'n2', to: 'n4', label: 'não' },
  { from: 'n3', to: 'n5' },
  { from: 'n4', to: 'n6' },
]

export const campaigns = [
  {
    id: 'cp1',
    name: 'Aquecimento ICP · Abril',
    status: 'running' as const,
    sent: 1_840,
    planned: 2_400,
    replied: 312,
    converted: 48,
    startedAt: daysAgo(5),
  },
  {
    id: 'cp2',
    name: 'Retomada de leads frios Q1',
    status: 'paused' as const,
    sent: 640,
    planned: 1_200,
    replied: 72,
    converted: 9,
    startedAt: daysAgo(12),
  },
  {
    id: 'cp3',
    name: 'Pós-Expomáquinas',
    status: 'running' as const,
    sent: 220,
    planned: 340,
    replied: 88,
    converted: 14,
    startedAt: daysAgo(2),
  },
  {
    id: 'cp4',
    name: 'Reengajar quem abriu proposta',
    status: 'draft' as const,
    sent: 0,
    planned: 180,
    replied: 0,
    converted: 0,
    startedAt: null,
  },
]

export const agents = [
  {
    id: 'a1',
    name: 'Mila',
    role: 'Qualificação inbound',
    model: 'Claude Sonnet 4.6',
    status: 'active' as const,
    conversations24h: 142,
    handoffRate: 0.14,
    tone: 'Editorial · Consultivo',
  },
  {
    id: 'a2',
    name: 'Otto',
    role: 'Follow-up 72h',
    model: 'Claude Haiku 4.5',
    status: 'active' as const,
    conversations24h: 86,
    handoffRate: 0.22,
    tone: 'Objetivo · Direto',
  },
  {
    id: 'a3',
    name: 'Selma',
    role: 'Confirmação de reunião',
    model: 'Claude Sonnet 4.6',
    status: 'paused' as const,
    conversations24h: 0,
    handoffRate: 0,
    tone: 'Formal · Breve',
  },
]
