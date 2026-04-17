---
tipo: feature
dominio: comunicacao
---

# Notas Internas

## Propósito

Comentários do time dentro de uma conversa, **invisíveis ao lead**. Permitem anotar contexto, passar observações entre colegas, registrar decisões sobre a conversa sem poluir o histórico público que o lead vê (ou que fica entre nós e o lead).

## Atores e Permissões

- **Qualquer membro com acesso ao chat** da conversa: cria e lê.
- **Admin**: tudo.
- **Lead**: NUNCA vê.
- **Agente IA**: pode ler notas como contexto (configurável); pode escrever nota via AI Action `add_note_internal`.

Ações: `conversation.add_note_internal`, `conversation.view` (notas incluídas).

## Dados Envolvidos

- `id`, `organization_id`, `conversation_id`, `lead_id`.
- `author_id`, `author_type` (`member` | `agent` | `system`).
- `content` (texto, 1-4000 chars).
- `mentions`: array de member_ids mencionados (`@nome`).
- `is_pinned`: bool (fixada no topo).
- `created_at`, `updated_at`, `edited`: bool.

## Regras de Negócio

1. Notas **NUNCA** são enviadas ao canal externo — invariante absoluto. Path de envio não vê notas.
2. Notas aparecem **apenas** em UI interna.
3. Mensagem e nota são entidades separadas, ou mensagem com flag `is_internal=true`. Implementação flexível; comportamento é o que importa.
4. Notas editáveis pelo autor; admin pode editar qualquer (com audit).
5. Menção (`@membro`) dispara notificação ao mencionado.
6. Notas podem ser fixadas (pinned) — aparecem no topo da thread para contexto rápido.

## Fluxos do Usuário

### Criar Nota
1. No chat, toggle "Nota interna" ao compor.
2. Campo muda de cor (fundo amarelo p.ex.).
3. Envia com Enter — cria nota, NÃO envia ao lead.
4. Nota aparece inline na thread com fundo distintivo e badge "Interno".

### Mencionar Membro
1. `@` + nome no conteúdo → autocomplete de membros ativos.
2. Salvar → member recebe notificação (push/email).

### Editar
1. Próprio autor clica "Editar" em nota sua.
2. Campo vira editável inline.
3. Salvar → marca `edited=true`, `updated_at` atualizado.

### Fixar
1. Click em "Fixar" em nota.
2. Nota vai para o topo da thread (seção fixada).
3. Útil para contexto crítico ("Lead é diretor — tratar com cuidado").

### Deletar
- Autor ou admin deleta.
- Soft-delete por default (audit).

## Automações e Eventos

### Emite
- `NoteAdded`, `NoteEdited`, `NoteDeleted`, `NotePinned`, `NoteUnpinned`, `NoteMention(member_id)`.

### Reage
- Notificação push/email em mention.

## Integrações

- **Chat**: renderização inline.
- **Agente IA**: pode ler notas como contexto para responder (configurável — pode aumentar quality mas também custo LLM).
- **Audit**: nota editada/removida fica registrada.

## Edge Cases

- **Autor removido da org**: nota mantém visibilidade; autor aparece como "Membro removido".
- **Nota criada em conversa que depois foi arquivada**: nota persiste.
- **Nota com muito conteúdo** (> 4000): quebra em múltiplas notas ou rejeita (regra da org).
- **Mention a membro inativo**: ainda cria, mas notificação é suprimida; membro vê ao reativar.

## Validações

- Content: 1-4000 chars.
- Menções: member_ids existentes na org.
- Permissão: membro tem acesso à conversa.

## Métricas

- Notas criadas por período.
- Uso por membro (engajamento em context internal).
- Mentions enviadas/respondidas.
- Notas fixadas por conversa (média).

## Segurança

- Content de nota pode ser sensível — mesmas regras de mensagem em log (nunca em plaintext em info).
- Anonimização em export quando LGPD solicitado.
- Notas não vazam em preview externo, em relatório enviado a terceiros, em export para lead.

## UX

- Diferenciação visual forte (cor, ícone, badge).
- Colapso opcional de notas antigas para não poluir.
- Pin elevado com ícone distintivo.
- Mention rende com link clicável para perfil do membro.
