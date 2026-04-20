/**
 * Rascunho tipado das 16 entidades canonicas do Torque CRM.
 * Sera substituido por api.gen.ts quando o backend Go expuser OpenAPI.
 * Campos em camelCase — transformer converte do snake_case do wire.
 */

export interface Lead {
  id: string
  name: string
  company: string | null
  phone: string | null
  email: string | null
  origin: string | null
  rating: string | null
  score: number | null
  tags: string[]
  responsibleId: string | null
  sdrId: string | null
  closerId: string | null
  isShadow: boolean
  organizationId: string
  createdAt: string
  updatedAt: string
}

export interface Organization {
  id: string
  name: string
  slug: string
  planId: string | null
  paymentStatus: 'active' | 'overdue' | 'suspended' | 'cancelled'
  logoUrl: string | null
}

export interface TeamMember {
  id: string
  userId: string
  organizationId: string
  role: 'admin' | 'membro' | 'master'
  displayName: string
  avatarUrl: string | null
  isActive: boolean
}

export interface PipeRecord {
  id: string
  leadId: string
  stageId: string
  responsibleId: string | null
  organizationId: string
  updatedAt: string
}

export interface PipeWhatsApp extends PipeRecord {
  sdrId: string | null
}

export interface PipeConfirmacao extends PipeRecord {
  meetingAt: string | null
  noShow: boolean
}

export interface PipeProposta extends PipeRecord {
  heat: 1 | 2 | 3 | 4 | 5
  proposalValue: number | null
  erpSynced: boolean
  commitmentDate: string | null
}

export interface Conversation {
  id: string
  leadId: string
  agentId: string | null
  status: 'open' | 'closed' | 'archived'
  channel: 'whatsapp' | 'messenger' | 'instagram' | 'sz_chat'
  lastMessageAt: string | null
  organizationId: string
}

export interface ChannelMessage {
  id: string
  conversationId: string
  channel: Conversation['channel']
  direction: 'inbound' | 'outbound'
  content: string
  mediaUrl: string | null
  status: 'pending' | 'sent' | 'delivered' | 'read' | 'failed'
  timestamp: string
}

export interface Workflow {
  id: string
  name: string
  triggerType: string
  isActive: boolean
  definition: { nodes: unknown[]; edges: unknown[] }
  organizationId: string
}

export interface WorkflowExecution {
  id: string
  workflowId: string
  leadId: string | null
  status: 'pending' | 'running' | 'completed' | 'failed'
  currentNodeId: string | null
  error: string | null
  createdAt: string
}

export interface Campaign {
  id: string
  name: string
  status: 'draft' | 'active' | 'paused' | 'ended'
  objective: string | null
  deadline: string | null
  teamGoal: number | null
  individualGoal: number | null
  organizationId: string
}

export interface CopilotAgent {
  id: string
  templateType: string
  isActive: boolean
  isDefault: boolean
  personalityTone: string | null
  skills: string[]
  organizationId: string
}

/**
 * @deprecated Mantido apenas enquanto legados referenciam. Nova modelagem
 * unificada é Task (ADR-007). Follow-up passa a ser task.kind === 'followup'.
 */
export interface FollowUp {
  id: string
  leadId: string
  title: string
  dueDate: string
  priority: 'low' | 'medium' | 'high' | 'urgent'
  isCompleted: boolean
  assignedTo: string | null
  sourcePipe: string | null
}

// ---------------------------------------------------------------------------
// UI Mode & Task cockpit (ADR-007 / F17)
// ---------------------------------------------------------------------------

export type UiMode = 'manager' | 'salesperson'

export interface UiPreferences {
  mode: UiMode
}

export type TaskKind =
  | 'followup'
  | 'call'
  | 'qualification'
  | 'send_proposal'
  | 'confirm_meeting'
  | 'objection'
  | 'generic'

export type TaskPriority = 'low' | 'normal' | 'high' | 'urgent'

export type TaskStatus = 'pending' | 'in_progress' | 'done' | 'cancelled' | 'missed'

export type TaskOrigin = 'manual' | 'workflow' | 'agent' | 'rule' | 'system'

export interface Task {
  id: string
  organizationId: string
  leadId: string | null
  assignedTo: string
  createdBy: string | null
  kind: TaskKind
  title: string
  description: string | null
  priority: TaskPriority
  status: TaskStatus
  inQueue: boolean
  queuePosition: number | null
  dueAt: string | null
  startedAt: string | null
  completedAt: string | null
  completedBy: string | null
  cancelledAt: string | null
  cancelledReason: string | null
  missedReason: string | null
  origin: TaskOrigin
  context: Record<string, unknown> | null
  resultNote: string | null
  createdAt: string
  updatedAt: string
}

export interface LeadSummary {
  id: string
  name: string
  phone: string | null
  pipeId: string
  pipeName: string
  stageId: string
  stageName: string
  heat: 1 | 2 | 3 | 4 | 5
  channel: 'whatsapp' | 'messenger' | 'instagram' | 'sz_chat'
}

export interface PipeStageSnapshot {
  id: string
  name: string
  position: number
  leadCount: number
  colorToken: string
  isActive: boolean
}

export interface TaskCockpitBundle {
  inProgress: Task | null
  queue: Task[]
  backlog: Task[]
  missed: Task[]
  activeLead: LeadSummary | null
  pipeStages: PipeStageSnapshot[]
  counts: {
    queue: number
    backlog: number
    missed: number
    completedToday: number
  }
}

export interface Product {
  id: string
  name: string
  type: 'mrr' | 'projeto' | 'unitario'
  price: number | null
  ticket: number | null
  ticketMin: number | null
  organizationId: string
}

export interface Commission {
  id: string
  dealId: string
  memberId: string
  value: number
  status: 'pending' | 'approved' | 'paid'
  paidAt: string | null
}

export interface OrgQuota {
  resourceKey: string
  planBase: number
  purchasedAddons: number
  adminAdjustment: number
  effectiveLimit: number
  currentUsage: number
  canAdd: boolean
}

export interface Operation {
  id: string
  type: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: number | null
  startedAt: string | null
  endedAt: string | null
  error: { code: string; message: string } | null
  result: unknown
}

export interface CursorPage<T> {
  items: T[]
  nextCursor: string | null
  total?: number
}

export interface SessionBundle {
  user: {
    id: string
    email: string
    displayName: string
    uiPreferences?: UiPreferences
    // S36: team_member_id exposto em /me para a UI distinguir "assumir"
    // vs "já é minha" sem roundtrip extra a /members.
    teamMemberId: string
  }
  org: Organization
  role: TeamMember['role']
  isMaster: boolean
  featurePermissions: Record<string, boolean>
  quotas: Record<string, Pick<OrgQuota, 'effectiveLimit' | 'currentUsage' | 'canAdd'>>
}
