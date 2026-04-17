---
tipo: feature
dominio: admin
criticidade: alta
---

# Master Admin

## Propósito

Painel da **empresa-dona** do produto. Permite provisionar organizações, suportar clientes, investigar problemas cross-org, gerenciar planos globais, rotacionar secrets, impersonar admins em casos autorizados.

> **Área crítica**: ações master afetam clientes. Todo ato é auditado, alguns geram alerta imediato ao time de segurança.

## Atores e Permissões

- **Master Admin** (empresa-dona). Vínculo separado da tabela de membros de org cliente.
- **MFA obrigatório** para master.

Ações: `master.*`.

## Telas Principais

### 1. Lista de Organizações
- Todas as orgs com: nome, plano, membros ativos, último acesso, saúde (alertas), total de leads, total de vendas.
- Filtros: plano, status (ativa/suspensa/expirada), onboarded/não-onboarded, último login.
- Ordenação.
- Click → detalhe da org.

### 2. Detalhe da Organização
- Dados completos.
- Plano, histórico de pagamentos.
- Uso: quota consumida × limite.
- Integrações: quais estão conectadas, status.
- Time: lista de membros.
- Saúde: integrações falhando, fila de jobs acumulada.
- Ações:
  - Impersonar admin.
  - Suspender.
  - Reativar.
  - Mudar plano.
  - Override de quota.
  - Forçar sync (integrações).
  - Deletar (soft, hard com carência).

### 3. Provisionamento
- Wizard para criar nova organização:
  - Dados da empresa.
  - Plano inicial.
  - Admin inicial (email + nome).
  - Opcional: override de features.
- Após criar: envia email ao admin com credenciais/link.

### 4. Métricas Cross-Org
- Dashboard global: MRR, churn, novo MRR, upgrades/downgrades, orgs ativas, orgs suspensas, NPS.
- Tendências.

### 5. Gestão de Planos
- Catálogo de planos (features, quotas, preços).
- Criar novo plano, editar existente.
- Ao editar: afeta novas assinaturas; atuais seguem plano antigo até renovação.

### 6. Gestão de Features Globais
- Catálogo de features que compõem planos.
- Feature flags experimentais (orgs beta).

### 7. Audit Cross-Org
- Logs de auditoria todas as orgs.
- Filtros: ator, ação, org, período.
- Alertas de eventos críticos.

### 8. Saúde do Sistema
- Fila de jobs: ver, reprocessar, esvaziar.
- Webhook deliveries falhadas.
- Integrações com saúde degradada por org.
- Alertas ativos.

### 9. Suporte
- Tickets (se integrado).
- Notas de cada organização (interno, não visível a cliente).
- Histórico de atendimentos.

## Impersonação

### Fluxo
1. Master clica "Impersonar admin da Org X".
2. **Justificativa obrigatória** (texto livre, min 20 chars).
3. Confirma MFA.
4. Sistema emite token com flag `impersonation`, `target_user_id`, `target_org_id`, `expires_at` (30 min default).
5. Master entra na UI como se fosse admin.
6. **Banner vermelho persistente**: "Você está impersonando [admin name] da [org name]. Sessão expira em X min. [Encerrar]".
7. Toda ação é executada em nome do admin mas registrada com flag `via_master`.
8. Ao encerrar ou expirar: volta ao painel master.

### Limites
- Algumas ações permanentes (hard-delete, rotacionar API key) podem ser bloqueadas durante impersonação (configurável).
- Máxima duração: 1h, extensível.
- Cliente pode ver no audit (visibilidade configurável) que master acessou — por transparência contratual.

## Provisionamento Automático (via Checkout)

- Cliente compra plano no checkout → webhook → sistema cria org automaticamente.
- Master é notificado de nova organização.
- Admin recebe email de ativação.

## Regras de Negócio

1. Master é identidade separada de admin de org cliente.
2. Toda ação master fica em audit crítico com justificativa.
3. Impersonação é tempo-limitada.
4. Alguns dados (ex.: conteúdo completo de conversa de cliente específico) podem exigir aprovação cross-master (2-person rule) para garantir privacidade do cliente.
5. Rotação de secret global: exige MFA + confirmação dupla + alerta imediato.
6. Hard-delete: janela de carência ≥ 30 dias após desativação.

## Fluxos do Usuário

### Provisionar Nova Org
1. Master → `Organizações → Nova`.
2. Form preenchido.
3. Confirma → cria + envia email.
4. Admin recebe, completa setup.

### Investigar Problema de Cliente
1. Cliente abre ticket "workflow não dispara".
2. Master abre detalhe da org → Saúde.
3. Vê: fila de workflow com X jobs em falha.
4. Click em job → detalhe do erro.
5. Identifica problema (ex.: template removeu).
6. Impersona admin → corrige workflow.
7. Encerra impersonação.
8. Responde ticket.

### Investigar Bug Cross-Org
1. Relato: "várias orgs reportam lentidão".
2. Master → Saúde do Sistema → Métricas globais.
3. Identifica: fila de `process-webhook-deliveries` acumulada em 3 orgs.
4. Rastreia causa (ex.: um provider externo lento).
5. Aplica mitigação (reprocess, ajuste de timeout).

## Automações e Eventos

### Emite
- `MasterAction(actor, org, action, reason, timestamp)`.
- `OrganizationProvisioned`, `OrganizationHardDeleted`, `OrganizationSuspended`, `OrganizationReactivated`.
- `ImpersonationStarted`, `ImpersonationEnded`.

### Reage
- Alertas em ações críticas (email ao time de segurança).

## Integrações

- **Todas as features** (master pode atuar em qualquer).
- **Audit log**.
- **Provedor de pagamento** (checkout, webhooks).
- **Email** (notificações, ativação, alertas).
- **Analytics internos** (métricas do produto).

## Edge Cases

- **Master auditando master**: ação de um master é visível para outros masters.
- **Master removido** (ex.: funcionário desligado): acesso revogado imediatamente.
- **Impersonação de admin que foi removido**: bloqueada.
- **Organização sem admin ativo**: master pode designar novo via impersonação (mas só via impersonação, não cria admin em nome próprio sem fluxo explícito).

## Validações

- MFA sempre exigido.
- Justificativa não-vazia.
- Ação dentro do escopo master (não ações que só o admin da org pode pedir ativamente, ex.: mudanças em ToS).

## Métricas

- Ações master por período.
- Distribuição de tipo de ação.
- Impersonações: duração, ações feitas.
- Provisionamento automático vs manual.

## Segurança

- MFA obrigatório.
- IP allowlist opcional para painel master.
- Rate limit alto mas existente.
- Alertas: provisionamento em massa, hard-delete, rotação de secret.
- Revisão periódica de lista de masters.
