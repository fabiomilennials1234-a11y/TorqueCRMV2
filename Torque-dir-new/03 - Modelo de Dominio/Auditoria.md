---
tipo: dominio
entidade: Auditoria
---

# Auditoria (Audit Log / Histórico)

Sistema de registro imutável append-only de ações relevantes. Existe em múltiplas granularidades:

1. **Lead History**: eventos sobre lead específico.
2. **Execution Steps**: eventos sobre execução de workflow.
3. **Admin Audit Log**: ações administrativas (login, mudança de permissão, master actions).
4. **Integration Audit**: chamadas a integrações externas (sucesso/falha, código HTTP).
5. **Domain Event Log**: todos eventos de domínio emitidos (fonte para replay/reprocessamento).

## Princípios

1. **Append-only**: nunca UPDATE, nunca DELETE (exceto retenção automática por política).
2. **Imutável**: correção cria novo evento, não altera antigo.
3. **Rastreável**: quem, quando, o quê, contexto.
4. **Per-tenant**: toda entrada pertence a uma organização.
5. **Retenção mínima**: 1 ano para audit admin; 6 meses para lead history; conforme LGPD.

## Estrutura Comum

Todo evento de auditoria carrega:

- `id`: UUID.
- `organization_id`.
- `actor_type`: `user` | `agent` | `system` | `workflow` | `webhook` | `master`.
- `actor_id`: id específico (user_id, agent_id, workflow_id, etc.; null para sistema).
- `action`: enum da ação.
- `entity_type`: tipo da entidade afetada.
- `entity_id`: id da entidade.
- `timestamp`.
- `context`: JSON com detalhe (IP, user-agent, origem, razão quando relevante).
- `before`: estado antes (opcional).
- `after`: estado depois (opcional).
- `correlation_id`: para vincular múltiplos eventos de uma mesma operação.

## Lead History

### Ações registradas
- `lead.created`
- `lead.updated` (com delta de campos)
- `lead.assigned` (responsible/sdr/closer)
- `lead.tag_added`, `lead.tag_removed`
- `lead.stage_changed` (com pipeline, from, to)
- `lead.entered_pipe`, `lead.left_pipe`
- `lead.score_changed`
- `lead.note_added` (notas internas)
- `lead.message_sent`, `lead.message_received`
- `lead.followup_created`, `lead.followup_completed`
- `lead.deleted`

### Visualização
- Timeline no drawer do lead: mostra tudo em ordem cronológica.
- Filtros por tipo, por ator, por período.
- Expansível: cada evento pode mostrar detalhes completos.

## Admin Audit Log

### Ações registradas
- `auth.login`, `auth.logout`, `auth.failed_login`.
- `auth.password_changed`, `auth.mfa_enabled`, `auth.mfa_disabled`.
- `member.invited`, `member.activated`, `member.role_changed`, `member.removed`.
- `permission.granted`, `permission.revoked`.
- `organization.plan_changed`, `organization.suspended`, `organization.reactivated`.
- `integration.connected`, `integration.disconnected`, `integration.token_refreshed`.
- `apikey.created`, `apikey.revoked`.
- `export.requested`, `export.downloaded`.
- `master.impersonation_started`, `master.impersonation_ended` (com justificativa).
- `master.hard_delete_organization`.

### Visualização
- Apenas admin da própria org vê ações da sua org.
- Master admin vê tudo.
- Logs não-filtrados não acessíveis a membros normais.

## Integration Audit

### Ações
- Chamadas HTTP a providers externos.
- `integration.call_started`, `integration.call_succeeded`, `integration.call_failed`.
- Detalhes: endpoint, método, duração, código HTTP, tentativa N/M.

### Uso
- Diagnóstico de falha de integração.
- Analytics de latência e disponibilidade por provider.

## Domain Event Log

- Todo evento de domínio (ver [[09 - Referências/Catálogo de Eventos]]) é persistido.
- Source of truth para replay em caso de perda de estado downstream.
- Permite reconstruir estado agregado (ex.: recalcular contadores).

### Formato
Exatamente o formato padrão de evento (ver [[01 - Arquitetura Conceitual/Padrões Transversais]]#8-eventos-de-domínio).

### Retenção
- Eventos críticos: retidos permanentemente (ou até remoção legítima LGPD).
- Eventos verbose (ex.: step-by-step de workflow): retidos conforme política.

## Execution Steps (workflows)

Ver [[Workflow]] para detalhe. Cada step de execução é registro imutável com input, output, status e erro.

## Consulta de Audit

### Casos de uso
1. **Admin** audita ações do time.
2. **Master** investiga incidente em uma org.
3. **Titular de dados** (LGPD) solicita histórico de acessos ao seu dado pessoal.
4. **Compliance**: exportar audit de período específico.
5. **Debugging**: rastrear o que aconteceu antes de um bug.

### API
- Filtros: ator, ação, entidade, período, texto livre.
- Exportação em CSV/JSON para compliance.

## Privacidade

- Log pode conter dados pessoais (nome de lead, email).
- Mascaramento quando exibido em logs operacionais (ex.: email parcial).
- Retenção respeita LGPD: se titular pede exclusão, eventos relacionados são anonimizados (não deletados — preservam integridade operacional).
- Direito de acesso: titular pode solicitar exportação de eventos relativos a si.

## Segurança do Próprio Audit

- Log não pode ser modificado ou apagado por usuário do sistema.
- Master admin tem acesso de leitura; não tem acesso de escrita direta (só via ações auditadas).
- Integridade verificável: hash encadeado opcional (Merkle log) para detectar tampering.

## Retenção

- **Legal**: LGPD exige no mínimo 6 meses de logs de acesso a dado pessoal; recomendado 1-5 anos.
- **Default Torque**: 1 ano para audit admin, 6 meses para lead history, 30 dias para logs verbose.
- Arquivamento para storage frio após janela quente.
- Hard-delete conforme política (retenção legal máxima).

## Performance

- Índices por `(organization_id, timestamp DESC)`, `(organization_id, entity_type, entity_id)`, `(organization_id, actor_id, timestamp)`.
- Particionamento por mês para tabelas grandes.
- Query de timeline de lead paginada e lazy.

## Alertas

- Ações sensíveis geram alerta imediato:
  - `master.impersonation_started` → notifica time de segurança interno.
  - `apikey.created` → notifica admin da org por email.
  - `export.requested` de grande volume → notifica admin.
  - Falha de integração crítica após X tentativas.

## Eventos Não-logados

- Leitura de entidade (GET comum) não entra em audit — volume excessivo.
- Exceções: acesso a dado marcado como **sensível** (configurável), leituras cross-org por master.
