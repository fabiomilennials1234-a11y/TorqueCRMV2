---
tipo: feature
dominio: comunicacao
---

# Templates de Mensagem

## Propósito

Biblioteca de mensagens reutilizáveis com **placeholders dinâmicos** (nome do lead, empresa, campo custom). Reduz esforço de digitar sempre o mesmo, padroniza voz da org, e alimenta automações (workflows, campanhas).

## Atores e Permissões

- **Admin**: CRUD de templates da organização.
- **Membros**: usam templates; podem criar "privados" (visíveis só a si).
- **Workflows e Campanhas**: referenciam templates por id.

Ações: `template.view`, `template.use`, `template.create`, `template.edit`, `template.delete`.

## Dados Envolvidos

- `id`, `organization_id`, `name`, `content`, `placeholders` (derivados do content), `category`, `is_active`, `is_private`, `created_by`, `created_at`, `updated_at`.
- `usage_count`: cache.
- `preview_variables`: valores fake para preview (ex.: `{name: "Maria"}`).

## Placeholders Disponíveis

Sintaxe: `{{ path }}`.

- `{{ lead.name }}` — primeiro nome do lead.
- `{{ lead.company }}` — empresa.
- `{{ lead.email }}` — email.
- `{{ lead.phone }}` — telefone.
- `{{ lead.custom_fields.<key> }}` — campo custom.
- `{{ member.name }}` — nome do membro que envia (quando aplicável).
- `{{ organization.name }}` — nome da empresa.
- `{{ today }}` — data atual.
- `{{ meeting.date }}` — quando contexto é uma reunião.
- `{{ ... }}` — conforme contexto de uso.

Template engine resolve antes do envio. Placeholder não-resolvível: fallback para string vazia ou valor default configurado.

## Regras de Negócio

1. Nome único dentro da org (case-insensitive).
2. Content suporta texto e placeholders. Mídia (imagem/doc/audio anexa) é referenciada por ID separado.
3. Template com placeholder não-resolvível em uso real: warning mas envia (com string vazia) — configurável para **bloquear** se campo critical.
4. Categorias: "Abordagem", "Follow-up", "Confirmação", "Proposta", "Despedida", "Customizada".
5. Templates privados do membro: só ele usa; não aparecem para outros.

## Fluxos do Usuário

### Listar
1. Menu → `Templates`.
2. Tabela com: nome, categoria, preview (primeiros 100 chars), uso no mês, atalho se definido.
3. Filtros por categoria, admin vê tudo.
4. Admin destaca templates mais usados.

### Criar
1. Botão "Novo Template".
2. Form: nome, categoria, conteúdo (editor com autocomplete de placeholders), mídia anexa opcional.
3. Preview ao vivo com valores mock.
4. Salvar.

### Editar
- Clique em template → modal/tela de edição.
- Versionamento interno (últimas versões preservadas).

### Usar em Chat
1. No campo de envio, digita `/template` ou clica ícone.
2. Autocomplete/buscador abre.
3. Seleciona → content aparece no campo com placeholders resolvidos para o lead atual.
4. Editável antes de enviar.

### Usar em Workflow / Campanha
- Action `send_message` referencia `template_id`.
- Resolução de placeholders no momento do envio.

### Duplicar
- Botão "Duplicar" cria cópia editável.

### Deletar
- Soft-delete; se em uso por workflow/campanha, UI avisa referências.

## Automações e Eventos

### Emite
- `TemplateCreated`, `TemplateUpdated`, `TemplateDeleted`, `TemplateUsed(template_id, by, context)`.

### Reage
- Workflow/Campanha chama template → resolve + envia.

## Integrações

- **Chat**: uso direto.
- **Workflow**: action.
- **Campanha**: sequência de mensagens.
- **Agente IA**: pode referenciar templates como fallback estruturado.
- **Mídia**: anexos linkados ao template.

## Edge Cases

- **Placeholder inexistente** (`{{ lead.xyz }}` sem campo): no envio, fallback para vazio + warning em audit.
- **Template muito longo** (> limite de canal): UI avisa; envio pode chunkar.
- **Mídia no template + conteúdo textual**: ambos enviados (texto como legenda).
- **Emoji no template**: suportado.
- **Template com lista numerada/formatação**: respeita limites do canal (ex.: WhatsApp suporta `*bold*`, `_italic_`).
- **Idioma**: templates são texto — org pode criar versões por idioma (nome sufixo: "Abordagem PT-BR", "Abordagem EN").
- **Variável condicional** (`{{# if lead.company }} ... {{/if}}`): suportado como extensão se necessário.

## Validações

- Nome: 2-100 chars.
- Content: 1-4000 chars.
- Placeholders detectados: valida sintaxe antes de salvar.
- Mídia: tamanhos conforme canal.

## Métricas

- Templates mais usados.
- Conversão associada a cada template (ex.: template X leva a mais respostas).
- Tempo economizado (estimativa — uso × tempo digitação evitado).

## Organização e Boas Práticas

- Categorizar desde o início evita caos.
- Templates devem ser específicos — "Abordagem Produto X" é melhor que "Abordagem".
- Revisar periodicamente — remover inusados, atualizar desatualizados.
- Usar para padronizar voz da org, mas editáveis antes do envio para personalização.
