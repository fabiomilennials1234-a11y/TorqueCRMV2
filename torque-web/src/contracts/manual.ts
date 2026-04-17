/**
 * Rascunho tipado das 16 entidades canonicas do Torque CRM.
 * Sera substituido por api.gen.ts quando o backend Go expuser OpenAPI.
 * Campos em camelCase — transformer converte do snake_case do wire.
 */

export interface Lead {
  id: string;
  name: string;
  company: string | null;
  phone: string | null;
  email: string | null;
  origin: string | null;
  rating: string | null;
  score: number | null;
  tags: string[];
  responsibleId: string | null;
  sdrId: string | null;
  closerId: string | null;
  isShadow: boolean;
  organizationId: string;
  createdAt: string;
  updatedAt: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  planId: string | null;
  paymentStatus: "active" | "overdue" | "suspended" | "cancelled";
  logoUrl: string | null;
}

export interface TeamMember {
  id: string;
  userId: string;
  organizationId: string;
  role: "admin" | "membro" | "master";
  displayName: string;
  avatarUrl: string | null;
  isActive: boolean;
}

export interface PipeRecord {
  id: string;
  leadId: string;
  stageId: string;
  responsibleId: string | null;
  organizationId: string;
  updatedAt: string;
}

export interface PipeWhatsApp extends PipeRecord {
  sdrId: string | null;
}

export interface PipeConfirmacao extends PipeRecord {
  meetingAt: string | null;
  noShow: boolean;
}

export interface PipeProposta extends PipeRecord {
  heat: 1 | 2 | 3 | 4 | 5;
  proposalValue: number | null;
  erpSynced: boolean;
  commitmentDate: string | null;
}

export interface Conversation {
  id: string;
  leadId: string;
  agentId: string | null;
  status: "open" | "closed" | "archived";
  channel: "whatsapp" | "messenger" | "instagram" | "sz_chat";
  lastMessageAt: string | null;
  organizationId: string;
}

export interface ChannelMessage {
  id: string;
  conversationId: string;
  channel: Conversation["channel"];
  direction: "inbound" | "outbound";
  content: string;
  mediaUrl: string | null;
  status: "pending" | "sent" | "delivered" | "read" | "failed";
  timestamp: string;
}

export interface Workflow {
  id: string;
  name: string;
  triggerType: string;
  isActive: boolean;
  definition: { nodes: unknown[]; edges: unknown[] };
  organizationId: string;
}

export interface WorkflowExecution {
  id: string;
  workflowId: string;
  leadId: string | null;
  status: "pending" | "running" | "completed" | "failed";
  currentNodeId: string | null;
  error: string | null;
  createdAt: string;
}

export interface Campaign {
  id: string;
  name: string;
  status: "draft" | "active" | "paused" | "ended";
  objective: string | null;
  deadline: string | null;
  teamGoal: number | null;
  individualGoal: number | null;
  organizationId: string;
}

export interface CopilotAgent {
  id: string;
  templateType: string;
  isActive: boolean;
  isDefault: boolean;
  personalityTone: string | null;
  skills: string[];
  organizationId: string;
}

export interface FollowUp {
  id: string;
  leadId: string;
  title: string;
  dueDate: string;
  priority: "low" | "medium" | "high" | "urgent";
  isCompleted: boolean;
  assignedTo: string | null;
  sourcePipe: string | null;
}

export interface Product {
  id: string;
  name: string;
  type: "mrr" | "projeto" | "unitario";
  price: number | null;
  ticket: number | null;
  ticketMin: number | null;
  organizationId: string;
}

export interface Commission {
  id: string;
  dealId: string;
  memberId: string;
  value: number;
  status: "pending" | "approved" | "paid";
  paidAt: string | null;
}

export interface OrgQuota {
  resourceKey: string;
  planBase: number;
  purchasedAddons: number;
  adminAdjustment: number;
  effectiveLimit: number;
  currentUsage: number;
  canAdd: boolean;
}

export interface Operation {
  id: string;
  type: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  progress: number | null;
  startedAt: string | null;
  endedAt: string | null;
  error: { code: string; message: string } | null;
  result: unknown | null;
}

export interface CursorPage<T> {
  items: T[];
  nextCursor: string | null;
  total?: number;
}

export interface SessionBundle {
  user: { id: string; email: string; displayName: string };
  org: Organization;
  role: TeamMember["role"];
  isMaster: boolean;
  featurePermissions: Record<string, boolean>;
  quotas: Record<string, Pick<OrgQuota, "effectiveLimit" | "currentUsage" | "canAdd">>;
}
