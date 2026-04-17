---
tags: [design, ui, vocabulario, copy, microcopy]
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Vocabulário de UI

Este documento fixa os termos literais que aparecem na interface do Torque. Não é guideline de marketing — é contrato de microcopy. Mudança aqui é decisão de produto, não discussão de tradução. Ver [[Principios de Identidade Visual]] e [[Criterios de Reprovacao]].

## Termos canônicos

### Funis

Usamos **"Funis"**. Nunca "Pipelines", "Pipes" como label de navegação (é só internamente/técnico), nunca "Sales Funnels".

Justificativa: "Pipeline" é jargão SaaS importado. "Funis" é português, imediatamente legível pelo usuário comercial brasileiro, e carrega a imagem mental certa — afunilamento, não tubulação.

Exceção: "Pipe Propostas" é nome próprio de módulo específico. Mantém-se.

### Conversas

Usamos **"Conversas"**. Nunca "Chat" como label de módulo, nunca "Messages", nunca "Inbox".

Justificativa: "Chat" reduz o escopo (WhatsApp, Messenger, Instagram, SZ são canais, não chats isolados). "Conversas" engloba todas as trocas independente de canal e tem peso humano que "Messages" não tem.

### Agentes IA / Copilot

**"Agentes IA"** é o label do módulo. **"Copilot"** é o nome próprio do agente principal.

```
Nav: Agentes IA
Tela: Copilot - Configurar atendimento
```

Nunca "AI Agents", "Bot", "Assistant", "Artificial Intelligence". Copilot é nome de produto e vai com C maiúsculo sempre.

### Novo lead

Sempre **"Novo lead"** — o L em "lead" é minúsculo mesmo no início de frase ou label, salvo quando é primeira palavra de sentença.

```
Botão: Novo lead
Placeholder: Nome do lead
Título: Importar leads
Frase: Lead criado  (aqui L maiúsculo porque é começo)
```

Nunca "Novo Lead", "Create Lead", "Adicionar Contato", "Cliente".

### Compareceu

No contexto de reuniões/visitas. Usamos **"Compareceu"** no passado e **"Comparecer"** no infinitivo.

Nunca "Confirmado" (confirmar é etapa antes, compareceu é evento já ocorrido), nunca "Atendido".

```
Status options: Agendado / Confirmou presença / Compareceu / Faltou / Remarcou
```

### Calor (1–5)

No `HeatSlider` e em toda referência ao nível de temperatura do lead: **"Calor"** + número de 1 a 5.

```
Label: Calor
UI: Calor 3
Slider: Calor 1 → Calor 5
```

Nunca "Temperatura", "Hotness", "Score", "Rating" nesse contexto específico. O sistema tem score (ScoreMeter) que é outra coisa.

### Oráculo Comercial

Título canônico do chat IA no dashboard principal. **"Oráculo Comercial"**.

```
Card title: Oráculo Comercial
Placeholder: Pergunte ao Oráculo...
```

Nunca "AI Assistant", "Ask AI", "Chat IA". É branding interno e fica.

### Master Panel

Painel administrativo do master (dono do workspace). **"Master Panel"**.

Nunca "Admin Panel", "Painel Administrativo", "Settings avançadas".

### Roles: Membro / Admin / Master

Três roles. Labels exatos:

- **Membro** — usuário padrão, acesso a operação.
- **Admin** — administra workspace, gerencia membros, configura funis.
- **Master** — dono do workspace, billing, integrações globais.

Nunca "Usuário", "User", "Employee" no lugar de Membro. Nunca "Owner" no lugar de Master. Nunca "Manager" em qualquer lugar.

### Nomes de canais

Uso literal:

- **WhatsApp** (não "Whatsapp" nem "WA" em UI visível — apenas em ChannelBadge `icon-only`).
- **Messenger** (não "Facebook Messenger" nem "FB").
- **Instagram** (não "Insta" nem "IG").
- **SZ.Chat** (ponto entre SZ e Chat, canal proprietário).

---

## Tom de escrita

Direto, seco, confiante. Como conversa de engenheiro sênior com operador experiente. Nada cordial demais. Nada jocoso.

### Toasts de sucesso

Frases curtas no presente, sem exclamação, sem "com sucesso".

```
Lead criado
Mensagem enviada
Funil atualizado
Integração ativada
```

**Não:**

```
Foi criado com sucesso!
Mensagem enviada com sucesso ✅
Pronto! Seu lead foi adicionado
```

### Toasts de erro

Frase curta, direta, sem acusar o usuário.

```
Não foi possível enviar a mensagem
Falha ao conectar WhatsApp
Limite de leads atingido
```

Evitar "Erro:", "Ops!", "Algo deu errado".

### Empty states

Frase editorial curta (1 linha) + CTA clara. Sem spam motivacional.

```
Nenhum lead por aqui. Importe uma planilha ou crie manualmente.
→ [Importar]  [Novo lead]
```

```
Nenhuma conversa nova.
→ [Ver arquivadas]
```

```
Sem propostas ativas. Mova um lead para o Pipe Propostas.
```

**Não:**

```
🎉 Você ainda não tem nenhum lead! Vamos começar?
Nada para mostrar. Que tal adicionar algo?
Está vazio aqui :(
```

### Labels de campo

Imperativo curto ou substantivo direto. Sem "Por favor".

```
Nome
E-mail
Telefone
Mensagem inicial
```

Hints opcionais só quando necessário:

```
Telefone
Com DDD, sem formatação
```

### Botões

Verbo + substantivo (ou verbo puro se o contexto for óbvio).

```
Novo lead
Enviar
Salvar
Descartar
Importar planilha
Conectar WhatsApp
```

Nunca "Clique aqui", "Go", "Ok!".

### Confirmações destructive

Claras, não hesitantes. Sem "Tem certeza?".

```
Arquivar lead?
→ [Arquivar]  [Cancelar]

Excluir funil permanentemente?
Esta ação não pode ser desfeita.
→ [Excluir]  [Cancelar]
```

---

## Regras gerais de escrita

1. **Sem exclamações.** Único lugar aceitável: celebração de fechamento de venda.
2. **Sem emojis em UI própria.** Emojis aparecem só quando o usuário os inseriu em mensagens.
3. **Sem "com sucesso".** O verbo no passado já comunica sucesso.
4. **Sem "por favor".** Assumimos que o produto está pedindo, não implorando.
5. **Sem "algo".** Frases tipo "Algo deu errado" são ruído. Ser específico ou dizer "Falha ao X".
6. **Sem "você".** Mensagens de sistema são impessoais. "Limite atingido", não "Você atingiu o limite".
7. **Português do Brasil.** Nunca "e-mail" escrito "email", nunca "clique" como "click".
8. **Mono para números e IDs.** Qualquer valor numérico ou identificador em UI usa `font-metric`. Ver [[Tipografia]].

## Capitalização

Labels curtos em UI seguem sentence case: apenas a primeira letra maiúscula.

```
Novo lead
Importar planilha
Configurar WhatsApp
```

Exceções:

- Nomes próprios (Copilot, Oráculo Comercial, WhatsApp, Instagram).
- Módulos no topo da navegação podem ir em maiúscula se a diagramação exigir — mas a regra default é sentence case.

## Números, datas, valores

Ver [[Tipografia]]. Resumo:

- Valores monetários: `R$ 1.234,56` (vírgula decimal, ponto milhar, espaço depois do R$).
- Datas relativas até 7 dias: "há 2 min", "há 1 h", "há 3 d".
- Datas absolutas acima: "12 abr", "12 abr 2026" (quando ano ≠ corrente).
- Sempre em `font-metric tabular-nums`.

## Referência cruzada

- [[Tipografia]] — famílias e escala.
- [[Componentes Primitivos]] — EmptyState e Toast moram aqui.
- [[Criterios de Reprovacao]] — textos que reprovam.
- [[Principios de Identidade Visual]] — tom geral.
