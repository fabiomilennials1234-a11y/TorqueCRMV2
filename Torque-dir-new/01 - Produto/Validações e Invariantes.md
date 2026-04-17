---
tipo: requisitos
---

# Validações e Invariantes

Catálogo de validações mínimas e invariantes garantidos por cada entidade. Implementação deve respeitar todos.

## Por Entidade

### Lead
- Nome: ≥ 2 chars, não-vazio após trim.
- Ao menos um de `phone` ou `email`.
- `phone` em E.164 após normalização.
- `email` regex válido + lowercased.
- `rating` ∈ [1, 5].
- `qualification_score` ∈ [0, 100].
- `organization_id` imutável após criação.
- `external_id` único (se presente) dentro da org.

### Organização
- Slug único globalmente.
- Nome não-vazio.
- Ao menos 1 admin ativo em todo momento.
- Plano ativo referenciado.
- CNPJ validado quando preenchido.
- Timezone válido (IANA).

### Time Member
- `(organization_id, user_id)` único.
- `(organization_id, email)` único.
- Role ∈ {admin, membro}.
- Specialty ∈ {sdr, closer, prospectador, admin, outro}.
- Não rebaixar último admin.

### Pipeline
- Nome único por org.
- Tipos estruturais existem em toda org (não deletáveis).
- Custom: `is_active` toggleable.
- Ao menos 1 stage.

### Stage
- Nome único dentro do pipeline (case-insensitive).
- `order` único dentro do pipeline.
- `is_final` ∈ {`positive`, `negative`, `null`}.

### Pipeline Entry
- `(pipeline_id, lead_id)` único por entry ativo.
- `current_stage_id` pertence a `pipeline_id`.
- Entry finalizado (stage `is_final`) não pode ser "movido" para stage não-final sem ação especial.

### Tag
- Nome normalizado único na org (case-insensitive).
- Cor em formato válido (hex ou HSL).

### Produto
- Nome não-vazio.
- SKU único na org quando preenchido.
- `price >= 0`.
- Moeda válida.

### Conversa
- `(lead_id, channel, channel_instance_id)` único.
- `agent_id` pertence à mesma org (se preenchido).
- `human_takeover_until` timestamp válido (futuro quando ativo).

### Mensagem
- Append-only (exceto status).
- `external_id` único dentro do canal.
- `content_type` válido.
- Tamanho dentro de limite do canal.

### Workflow
- Grafo acíclico.
- Exatamente 1 trigger node.
- Edges válidos (portas compatíveis).
- Todos os nodes configurados (config obrigatório preenchido).
- `is_active` requer validação completa.

### Workflow Execution
- Status enum válido.
- `started_at <= completed_at` quando completa.
- `current_node_id` existe no workflow.
- `workflow_version` snapshot preservado.

### Agente IA
- Nome único na org.
- No máximo 1 `is_default=true` por org.
- Template válido.
- Business context não-vazio recomendado (warning).

### FAQ
- `question` e `answer` não-vazios.
- Embedding gerado ou marcado como pending.
- `agent_id` existe e pertence à org.

### Campanha
- Período: `start_date <= end_date` (quando end_date definido).
- Ao menos 1 stage.
- Templates referenciados existem.
- Agente referenciado (se aplicável) ativo.

### Commission Rule
- Scope válido.
- Value conforme calculation_type:
  - Percent: ∈ [0, 1].
  - Tiered: ranges não-sobrepostos, ordenados.
- Role split: soma ≤ 1.0.

### Follow-up
- `title` ≥ 3 chars.
- `assigned_to` membro ativo da org.
- `due_at` presente.

### Webhook Endpoint
- URL HTTPS em produção.
- SSRF check (não pode apontar para ranges privados).
- Secret gerado pelo sistema (não escolhido pelo admin).
- Eventos subscritos existem no catálogo.

### API Key
- Key armazenada como hash (não plaintext).
- Scope ≤ permissão do admin que criou.

## Invariantes Cross-Entity

### Todas as entidades filhas
- `organization_id` igual ao de entidades referenciadas.
- Exemplo: Lead.responsible_id → member da mesma org.

### Transação
- Operações cross-entity em transação:
  - Criar lead + entry + tags: tudo ou nada.
  - Mover stage + emitir evento: atômico.
  - Vendido + comissão + pedido ERP: idealmente transação distribuída; saga pattern se necessário.

### Foreign Keys
- Referências consistentes: entity só aponta para entities existentes.
- DELETE em cascata configurado:
  - Lead deletado (hard) → conversas, mensagens, entries, follow-ups, lead_history, lead_tags.
  - Organização deletada (hard) → tudo em cascata.
  - Workflow deletado → executions em andamento finalizam ou cancel.

## Validação de Input (por endpoint)

Para cada endpoint público:
- Schema JSON validado.
- Tipos estritos (número não aceita string).
- Limites (string ≤ N chars, array ≤ M itens).
- Enums checados.
- Formatos (UUID, email, phone, URL).
- Authorization antes da validação de conteúdo.

## Erros Estruturados

Retorno de erro sempre em formato:
```
{
  "error": {
    "code": "validation_failed",
    "message": "Human readable message",
    "details": [
      {"field": "name", "reason": "required"},
      {"field": "phone", "reason": "invalid_format"}
    ]
  }
}
```

Códigos padronizados. Lista mantida em referência compartilhada.

## Invariantes Temporais

- `created_at <= updated_at` sempre.
- `started_at <= completed_at` (ou `completed_at` null).
- `scheduled_for > created_at` (mensagens agendadas).
- `meeting_date > now` ao criar.
- Timestamps em UTC no banco.

## Invariantes Financeiros

- `commission_value >= 0` (ou negativo em caso de estorno explícito).
- `price >= 0`.
- `total = sum(items) - discount_total`.
- Moeda consistente dentro de proposta.
- Round a 2 casas decimais para exibição; armazenamento com precisão completa.

## Observabilidade de Violações

- Violação de invariante em produção deveria ser impossível.
- Se detectada (ex.: via check em batch): alerta imediato, correção emergencial.
- Auditoria forense: quando/como ocorreu.

## Testes de Invariante

Suite automatizada que verifica:
- Criar entidade com input inválido → rejeita.
- Modificar para estado inválido → rejeita.
- Operações concorrentes mantêm consistência.
- Estado após job async está consistente.
