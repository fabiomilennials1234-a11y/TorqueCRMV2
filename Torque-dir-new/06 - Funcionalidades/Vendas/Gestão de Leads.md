---
tipo: feature
dominio: vendas
---

# Gestão de Leads

## Propósito

CRUD e operações cotidianas sobre a entidade Lead. É a interface primária dos membros do time comercial com a oportunidade comercial — criar, editar, atribuir, marcar tags, procurar, ver histórico, anexar follow-ups.

## Atores e Permissões

- **Admin**: tudo — ver todos, editar qualquer campo, reatribuir, exportar, deletar.
- **Membro (SDR/Closer/Prospectador)**: ver próprios e sem-responsável, criar, editar os seus, atribuir a si, adicionar tags.
- **Agente IA**: pode alterar alguns campos via AI Actions (tag, score, custom fields) conforme Kanban Rules.
- **Webhook externo**: cria leads via endpoint público de ingestão (com API key).
- **Sistema**: pode criar leads via conversa inbound (quando número desconhecido escreve).

Ações relevantes: `lead.view`, `lead.create`, `lead.update`, `lead.assign`, `lead.tag`, `lead.delete`, `lead.export`. Ver [[03 - Identidade e Permissões/Permissões por Feature]].

## Dados Envolvidos

Entidade Lead completa em [[02 - Modelo de Domínio/Lead]]. Campos principais:

- Identificação: nome, empresa, telefone (E.164), email.
- Classificação: rating (1-5, manual), qualification_score (0-100, automático), origin, segment.
- Atribuição: responsible, sdr, closer.
- UTMs: source, medium, campaign, term, content.
- Custom fields: schema dinâmico por organização.
- Relações: tags (N:N), pipeline entries (N), conversas (N), follow-ups (N).

## Estados e Transições

Lead não tem "status" unificado — status é **derivado** dos pipeline entries (ver [[02 - Modelo de Domínio/Lead]]#estados-derivados). Para a UI:

- **Novo**: sem entries ou entry em stage inicial.
- **Em qualificação**: entry ativa em Pipeline WhatsApp.
- **Em reunião**: entry ativa em Pipeline Confirmação.
- **Em proposta**: entry ativa em Pipeline Propostas.
- **Vendido**: entry final positivo em Propostas.
- **Perdido**: entry final negativo sem positivo ativo.
- **Descartado**: tag específica + nenhum entry ativo.

## Regras de Negócio

1. Nome obrigatório, não-vazio após trim.
2. Ao menos um de telefone/email obrigatório.
3. Telefone normalizado para E.164 na entrada.
4. Email lowercased.
5. Qualification score em [0,100]; rating em [1,5].
6. Dedupe na ingestão: prefira atualizar a criar duplicado (configurável por fonte).
7. Lead criado sem atribuição recebe distribuição automática se configurada para o pipeline default.
8. Alteração de campo crítico (responsible, stage, custom field sensível) gera entrada em Lead History.
9. Soft-delete preserva histórico; hard-delete requer confirmação dupla e permissão master-nível.
10. Lead sem contato (telefone inválido E email inválido) é marcado `incontatável` — aparece em UI mas não recebe envios.

## Fluxos do Usuário

### Criar Lead (manual)
1. Usuário abre `Leads → Novo Lead`.
2. Preenche formulário: nome (obrigatório), telefone/email (obrigatório um), empresa, origem, tags, responsável.
3. Opcional: marcar "Adicionar ao Pipeline WhatsApp automaticamente" (default ligado).
4. Clica "Criar".
5. Sistema valida, normaliza, persiste, cria entry no pipeline default se marcado, emite eventos.
6. UI redireciona para drawer do lead criado.

### Listar / Filtrar Leads
1. Usuário abre `Leads` (lista).
2. Vê tabela com colunas: nome, empresa, telefone, origem, stage atual, responsável, rating, score, tags, created_at, última interação.
3. Filtros disponíveis:
   - Origem.
   - Tags (has_any / has_all).
   - Pipeline + stage.
   - Responsável (own, sem-resp, específico).
   - Período de criação.
   - Texto livre (nome, telefone, email, empresa).
   - Score range.
   - Segment.
   - Campos custom.
4. Ordenação por qualquer coluna.
5. Paginação (50 por página default).
6. Export CSV/JSON se permissão.

### Ver Detalhe (drawer)
1. Clica em card do lead.
2. Drawer lateral abre com:
   - **Header**: nome, empresa, rating, score, tags, botões de ação rápida.
   - **Tabs**: Dados, Conversas, Pipeline, Histórico, Follow-ups, Notas, Custom Fields, Anexos.
3. Edição inline em campos (clica, edita, salva).
4. Botões: ligar, abrir chat, criar follow-up, mover stage, adicionar tag, atribuir.

### Editar Lead
1. Drawer ou form dedicado.
2. Validação dos mesmos invariantes da criação.
3. Diff registrado em Lead History.
4. Evento `LeadUpdated` emitido.

### Atribuir
1. Menu no drawer "Atribuir".
2. Seleciona papéis: Responsável / SDR / Closer.
3. Busca membro por nome.
4. Salva.
5. Membro é notificado.

### Marcar Tag
1. Botão "+" em área de tags.
2. Autocomplete de tags existentes; pode criar nova se tem permissão.
3. Aplicação imediata.

### Deletar
1. Drawer → menu "..." → Deletar (soft).
2. Confirmação com motivo (opcional).
3. Lead marcado como `deleted_at`, sumido da lista default, visível em filtro "deletados" para admin.

### Exportar
1. Lista com filtros aplicados → botão Export.
2. Escolha de formato (CSV ou JSON) e de colunas.
3. Export assíncrono se volume grande: job gera arquivo, notifica por email com link temporário.

## Automações e Eventos

### Emite
- `LeadCreated`, `LeadUpdated`, `LeadAssigned`, `LeadTagAdded`, `LeadTagRemoved`, `LeadDeleted`.

### Consome (reage a)
- Evento de entrada de lead externo → cria lead.
- Evento de mensagem inbound de número desconhecido → cria lead + conversa.
- Evento de stage changed → atualiza last_interaction_at.
- Evento de payment succeeded → atualiza tags/segmento do lead se configurado.

### Dispara
- Distribuição automática em novos leads (ver [[07 - Processos Assíncronos/Distribuição de Leads]]).
- Workflows com trigger `lead_created`, `tag_added`, etc.
- Cálculo de qualification_score.

## Integrações

- Webhook externo de ingestão (ver [[05 - Integrações Externas/n8n (Orquestrador Externo)]]).
- Pipeline entries (ver [[02 - Modelo de Domínio/Pipeline]]).
- Conversas (ver [[Chat Multi-canal]]).
- Copilot (agentes podem modificar lead via AI Action).
- Campanhas (lead pode ser enrolado).
- Workflows (triggers e actions sobre lead).
- Analytics (agregações por origem, score, stage).
- ERP externo (lead pode ter `external_id` de sistema do cliente).

## Edge Cases

- **Telefone inválido após normalização**: aceita se email presente; marca `phone_invalid`.
- **Email duplicado mas telefone diferente**: duplicata? Configurável: dedupe por qualquer contato que match.
- **Lead sem responsável em pipeline com distribuição**: job pega e atribui.
- **Reatribuir lead em atendimento ativo**: novo responsável é notificado, antigo perde acesso (se `view:own`), audit registra.
- **Lead com 1000+ mensagens**: UI pagina histórico, busca full-text.
- **Custom field com schema alterado após criação**: lead mantém valor legado; UI mostra com warning.
- **Tag renomeada**: associações persistem; não quebra.
- **Deletar lead com campanha ativa**: remove da campanha + audit.

## Validações

- Nome: 2-200 chars, não só espaços.
- Telefone: após normalização E.164, regex `^\+\d{10,15}$`.
- Email: regex padrão + verificação de domínio sintático.
- Rating: 1-5 integer.
- Custom fields: conforme schema da org (type, max length, opções de enum).
- Origin: string livre, mas aceita enum sugerido para autocomplete.
- UTMs: strings ≤ 255 chars cada.

## Métricas

- Leads criados (por dia, semana, mês) filtrável por origem.
- Taxa de dedupe (criação que virou update).
- Tempo até primeira interação.
- Conversão por origem.
- Distribuição por responsável.
- Score médio por origem.
- Campos custom mais usados (para otimizar schema).
- Exports solicitados (audit + volume).
