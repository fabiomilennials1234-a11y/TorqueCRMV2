export const VOCAB = {
  lead: "Lead",
  leads: "Leads",
  pipe: "Funil",
  pipes: "Funis",
  stage: "Estágio",
  stages: "Estágios",
  heat: "Calor",
  scheduled: "Agendado",
  attended: "Compareceu",
  sold: "Vendido",
  lost: "Perdido",
  copilot: "Copilot",
  oracle: "Oráculo Comercial",
  dispatchRule: "Regra de disparo",
  masterAdmin: "Master Admin",
  org: "Organização",
  member: "Membro",
  admin: "Admin",
  master: "Master",
  conversations: "Conversas",
  workflows: "Fluxos",
  campaigns: "Campanhas",
  agents: "Agentes IA",
  analytics: "Analytics",
  team: "Time",
  products: "Produtos",
  settings: "Configurações",
  followUps: "Follow-ups",
  dashboard: "Visão geral",
} as const;

export type VocabKey = keyof typeof VOCAB;

export const ROLES = {
  admin: "Admin",
  membro: "Membro",
  master: "Master",
} as const;

export const CHANNELS = {
  whatsapp: "WhatsApp",
  messenger: "Messenger",
  instagram: "Instagram",
  sz_chat: "SZ.Chat",
} as const;

export type Channel = keyof typeof CHANNELS;
