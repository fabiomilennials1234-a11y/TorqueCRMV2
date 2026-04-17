---
tipo: dominio
entidade: Tag
---

# Tag

Rótulo de segmentação livre aplicável a leads (e conversas). Não define workflow nem permissão — é meramente classificação para filtro, relatório e automação.

## Atributos

- `id`: UUID.
- `organization_id`.
- `name`: texto (ex.: "Ouro", "Diamante", "Perdido-por-preço", "Interessado-em-produto-X").
- `normalized_name`: `name.toLowerCase().trim()` para dedupe case-insensitive.
- `color`: cor visual (HSL ou hex).
- `description`: texto opcional.
- `category`: agrupamento opcional (ex.: "Segmento de Lead", "Motivo de perda", "Origem de evento").
- `is_auto_managed`: se for criada/removida por automação (não deve ser editada manualmente).
- `created_by`: membro que criou.
- `usage_count`: derivada — número de leads com esta tag (cache atualizado).

## Invariantes

1. `normalized_name` único dentro da organização.
2. Tag não pode ser deletada se em uso (pedir desassociar primeiro, ou soft-delete com flag `is_deleted`).
3. Tag nunca cruza organização (sem tags "globais").

## Associação Lead-Tag

- Entidade N:N: `lead_id`, `tag_id`, `added_by`, `added_at`, `origin` (`manual` | `automation` | `workflow` | `agent` | `webhook`).
- Chave única: `(lead_id, tag_id)`.
- Associar mesma tag duas vezes: no-op idempotente.

## Operações

### CreateTag
- Validar nome não vazio, não duplicado.
- Pode definir cor e descrição.

### UpdateTag
- Editar label/cor/descrição.
- Renomear: atualiza `normalized_name`. Se conflita com outra tag, rejeita.

### DeleteTag
- Se `usage_count > 0`: avisar e pedir confirmação.
- Soft-delete preferido.

### AddTagToLead / RemoveTagFromLead
- Validar tag e lead pertencem à mesma org.
- Atualizar associação.
- Emitir evento correspondente.

## Eventos

- `TagCreated`, `TagUpdated`, `TagDeleted`.
- `LeadTagAdded(lead_id, tag_id, origin, by)`.
- `LeadTagRemoved(lead_id, tag_id, by)`.

## Usos

- **Filtros**: em qualquer lista de leads.
- **Workflows**: triggers `tag_added`, `tag_removed`; actions `add_tag`, `remove_tag`.
- **Regras de pipe**: segmentação por tag.
- **Campanhas**: públicos-alvo (leads com tag X).
- **Analytics**: segmentação (CPL por tag, conversão por tag).
- **Agentes IA**: agentes podem aplicar tag como ação (ex.: "lead qualificado" → tag `Qualificado`).

## Busca

- Autocomplete case-insensitive por prefixo.
- Busca por categoria.
- Sugestão: ao começar a digitar, mostrar tags existentes antes de permitir criação nova.

## Regras de Negócio

- **Máximo de tags por lead**: limite alto (ex.: 50), UI avisa em >10.
- **Tag auto-gerenciada** (ex.: criada por workflow) aparece com ícone distintivo. Usuário ainda pode remover manualmente, mas o workflow pode re-adicionar.
- **Tag pode ser "protegida"**: só admin remove (útil para tags de segmento financeiro aplicadas por regra).

## Categorias sugeridas (convenção, não enforçada)

- **Segmento**: Ouro, Prata, Bronze, Diamante (qualidade financeira).
- **Origem de evento**: Evento-X, Webinar-Y.
- **Motivo de perda**: Sem-orçamento, Sem-timing, Concorrente.
- **Interesse**: Produto-X, Serviço-Y.
- **Status**: Qualificado, Requalificar, Descartado.

A organização decide estrutura.

## Limites

- Total de tags por org: limite de milhares (sem fricção prática).
- `usage_count` atualizado assincronamente (cache com refresh em evento).

## Integrações

- Webhook de ingestão aceita `tags: ["..."]` e resolve/cria case-insensitive.
- Campanhas podem adicionar/remover tags como action.
- Copilot agents com `allowed_actions` incluindo `add_tag`/`remove_tag` podem manipular.
