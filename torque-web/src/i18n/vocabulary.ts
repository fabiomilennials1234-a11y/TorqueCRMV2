/**
 * Vocabulário canônico do Torque CRM (PT-BR).
 *
 * Fonte única de verdade para os termos que aparecem em UI, erros,
 * logs de produto e comunicação com o usuário. Qualquer string que
 * mencione uma entidade ou conceito do produto DEVE usar a palavra
 * declarada aqui — nunca um sinônimo, nunca uma tradução livre.
 *
 * Glossário crítico (ver também CLAUDE.md e `Torque-dir-new/01 - Produto/Glossario.md`).
 */

export const VOCABULARY = {
  // ---- Entidades de vendas -------------------------------------------------
  lead: 'Lead',
  leads: 'Leads',
  funil: 'Funil',
  funis: 'Funis',
  stage: 'Stage',
  stages: 'Stages',
  calor: 'Calor',

  // Stages canônicos do funil WhatsApp (qualificação)
  stage_novo: 'Novo',
  stage_abordado: 'Abordado',
  stage_esfriamento: 'Esfriamento',
  stage_agendado: 'Agendado',
  stage_respondeu: 'Respondeu',
  stage_qualificado: 'Qualificado',
  stage_proposta: 'Proposta',
  stage_vendido: 'Vendido',
  stage_perdido: 'Perdido',

  // Ações do vendedor
  abordagem: 'Abordagem',
  esfriamento: 'Esfriamento',

  // ---- Tasks (ADR-007 unificou Follow-up em Task) --------------------------
  task: 'Task',
  tasks: 'Tasks',
  fila: 'Fila',
  aFazer: 'A fazer',
  emAberto: 'Em aberto',
  fazendo: 'Fazendo agora',
  concluida: 'Concluída',
  cancelada: 'Cancelada',
  perdida: 'Perdida',

  // ---- Copilot / IA --------------------------------------------------------
  copilot: 'Copilot',
  agenteIa: 'Agente IA',

  // ---- Organização e times -------------------------------------------------
  org: 'Organização',
  organizacao: 'Organização',
  time: 'Time',
  membro: 'Membro',
  admin: 'Admin',
  master: 'Master',

  // ---- Modos de UI (ADR-007) -----------------------------------------------
  modoVendedor: 'Vendedor',
  modoGerente: 'Gerente',

  // ---- Comunicação ---------------------------------------------------------
  conversa: 'Conversa',
  conversas: 'Conversas',
  mensagem: 'Mensagem',
  mensagens: 'Mensagens',
  template: 'Template',
  templates: 'Templates',
  nota: 'Nota interna',

  // ---- Canais --------------------------------------------------------------
  canal_whatsapp: 'WhatsApp',
  canal_instagram: 'Instagram',
  canal_messenger: 'Messenger',
  canal_sz: 'SZ.Chat',

  // ---- Automação -----------------------------------------------------------
  workflow: 'Fluxo',
  workflows: 'Fluxos',
  campanha: 'Campanha',
  campanhas: 'Campanhas',
  regraDePipe: 'Regra de funil',

  // ---- Analytics -----------------------------------------------------------
  dashboard: 'Dashboard',
  metrica: 'Métrica',
  meta: 'Meta',
  ranking: 'Ranking',
  comissao: 'Comissão',
  premiacao: 'Premiação',

  // ---- Admin ---------------------------------------------------------------
  configuracoes: 'Configurações',
  permissoes: 'Permissões',
  planos: 'Planos',
  faturamento: 'Faturamento',
  webhook: 'Webhook',
} as const

export type VocabularyKey = keyof typeof VOCABULARY

/**
 * Termos proibidos em UI e código. Se um destes aparecer em um PR,
 * reprova — o revisor deve apontar o termo canônico correspondente.
 *
 * Chaves = termo proibido (lowercase). Valores = termo canônico a usar.
 */
export const FORBIDDEN_TERMS: Record<string, string> = {
  // Lead
  prospect: 'Lead',
  contato: 'Lead',
  cliente: 'Lead',
  oportunidade: 'Lead',

  // Funil
  pipeline: 'Funil',
  funnel: 'Funil',

  // Stage
  coluna: 'Stage',
  etapa: 'Stage',
  fase: 'Stage',
  step: 'Stage',

  // Calor
  temperatura: 'Calor',
  score: 'Calor',
  heat: 'Calor',

  // Copilot
  bot: 'Copilot',
  chatbot: 'Copilot',
  assistente: 'Copilot',

  // Org
  tenant: 'Organização (em UI)',
  workspace: 'Organização',
  conta: 'Organização',

  // Tasks
  'follow-up': 'Task',
  followup: 'Task',
}
