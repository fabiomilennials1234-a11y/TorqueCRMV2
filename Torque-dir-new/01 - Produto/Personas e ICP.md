---
tags:
  - produto
  - personas
  - icp
  - jtbd
created: 2026-04-15
last_updated: 2026-04-16
status: vivo
---

# Personas e ICP

## ICP (Ideal Customer Profile) — a organizacao que compra o Torque

- **Segmento**: empresas B2B com venda consultiva — fabricas, distribuidoras, fornecedores industriais, integradoras, agencias, escolas profissionalizantes, clinicas com venda de plano, consultorias.
- **Regiao**: Brasil predominante (produto em portugues-BR, integracoes brasileiras como TinyERP e Asaas).
- **Operacao comercial**: opera WhatsApp intensamente, com fluxo de qualificacao → reuniao → proposta → fechamento.
- **Time**: 3 a 30 pessoas no comercial, divididas por papel (SDR / closer / prospectador / admin).
- **Volume**: dezenas a centenas de leads novos por dia, ticket medio de R$ 5k a R$ 500k.
- **Dor principal**: perde leads por falta de follow-up, nao sabe onde estao os leads no funil, cada vendedor tem sua planilha, gestor nao ve operacao em tempo real.

---

> **Regra critica — nao confunda conceito de negocio com role de sistema.**
> `SDR` e `Closer` sao **conceitos de negocio**, descrevem papeis comerciais dentro da operacao de uma org. **NUNCA** aparecem como roles no codigo.
> Roles no codigo sao apenas tres: `admin`, `membro`, `master`.
> Quem e SDR ou Closer e derivado de atributos do usuario (escopo em [[Pipe]]s, permissoes de feature, tags internas da org), nunca de um `role` enum. Violacao disso e bug de modelagem.

---

## 1. SDR / Prospector

**Contexto.** Esta o dia inteiro dentro da inbox de WhatsApp. Volume alto: dezenas a centenas de conversas abertas simultaneas. Meta e qualificar rapido e agendar reuniao com o closer. Trabalha em kanban visual, nao em planilha.

**Responsabilidades.**
- Receber `Lead`s de campanhas Meta Ads, formularios, indicacoes.
- Qualificar via conversa no WhatsApp (ICP, dor, orcamento, urgencia).
- Atualizar `Calor` do `Lead` (1-5) e mover pelo `Pipe` de prospeccao.
- Agendar reuniao com closer — transicao para `Stage` `Agendado`.
- Confirmar presenca no dia — transicao para `Compareceu`.

**Job-to-be-done principal.**
*"Quando chega um lead novo no WhatsApp, eu quero qualificar e agendar com o closer certo em minutos, sem sair da mesma tela, com o [[Copilot]] adiantando resposta quando eu estiver em outra conversa — para que eu consiga atender volume 3x maior sem perder calor do lead."*

**Telas criticas.**
- Inbox unificada (WhatsApp / Messenger / Instagram) com `Lead` no contexto lateral.
- Kanban do Pipe de prospeccao — arrastar entre `Stage`s.
- Ficha do Lead — historico, `Calor`, tags, origem, campos customizados.
- Painel do [[Copilot]] — ver e editar sugestoes, aprovar envios.

**Frustracoes tipicas.**
- Ficar alternando entre WhatsApp Web, planilha e CRM.
- Perder lead porque a janela de 24h do WhatsApp fechou sem ninguem ver.
- Nao saber qual lead responder primeiro (sem priorizacao por `Calor`).
- Ter que digitar a mesma mensagem de follow-up 50 vezes por dia.

---

## 2. Closer / Executivo Comercial

**Contexto.** Recebe leads ja qualificados pelo SDR. Foco em conduzir reuniao, apresentar proposta, fechar venda. Volume menor por dia, ticket maior, ciclo mais longo.

**Responsabilidades.**
- Conduzir reuniao `Agendada` / `Compareceu`.
- Gerar e enviar proposta comercial.
- Atualizar `Calor` e mover `Lead` pelo `Pipe` de fechamento.
- Registrar objecoes, proximos passos, valores.
- Fechar como `Vendido` ou `Perdido` (com motivo obrigatorio).
- Acionar emissao de NF-e no TinyERP quando `Vendido`.

**Job-to-be-done principal.**
*"Quando eu abro um lead agendado, eu quero o contexto completo da conversa, a qualificacao do SDR e os sinais de calor visiveis em 5 segundos — para chegar na reuniao sabendo exatamente onde o lead esta na jornada e aumentar minha taxa de fechamento."*

**Telas criticas.**
- Kanban do Pipe de fechamento — `Agendado` → `Compareceu` → Proposta → `Vendido` / `Perdido`.
- Ficha do Lead com historico completo da conversa (SDR + Copilot).
- Editor de proposta / registro de venda.
- Dashboard pessoal — meu pipeline, minhas metas, minha comissao prevista.

**Frustracoes tipicas.**
- Entrar em reuniao sem saber o que o SDR conversou.
- Nao ver claramente qual deal esta esfriando (sem `Calor` vivo).
- Ter que atualizar status em dois lugares (CRM + ERP).

---

## 3. Gestor Comercial (Admin)

**Contexto.** Responde pelos numeros da operacao. Olha o dashboard varias vezes ao dia. Configura a maquina: quem atende quem, quais automacoes rodam, quais campanhas alimentam o funil. Tem role `admin` no sistema.

**Responsabilidades.**
- Ranking da equipe — volume, conversao, receita, tempo de resposta.
- Metas por SDR, por closer, por `Pipe`, por periodo.
- Comissoes — regras, calculos, relatorio para pagamento.
- Analytics — funil, CAC por fonte, ROI de campanha, coorte.
- Configurar Dispatch Rules (sequencias automaticas de mensagem).
- Configurar Campanhas e integracao com Meta Ads.
- Criar e editar `Pipe`s e `Stage`s.
- Gerenciar seats e permissoes de membros da org.
- Ouvir o [[Oraculo Comercial]] apontando gargalos.

**Job-to-be-done principal.**
*"Quando eu abro o dashboard de segunda de manha, eu quero ver em uma tela onde o funil esta furando, qual SDR esta abaixo da meta, qual campanha esta dando ROI negativo e o que o [[Oraculo Comercial]] recomenda ajustar essa semana — para agir no mesmo dia, sem precisar montar relatorio em planilha."*

**Telas criticas.**
- Dashboard executivo com [[Oraculo Comercial]].
- Analytics — funil, campanhas, coortes, atribuicao Meta Ads → receita.
- Ranking e metas.
- Workflow Builder — criar e editar Dispatch Rules.
- Configuracoes da org — `Pipe`s, `Stage`s, seats, integracoes, billing.

**Frustracoes tipicas.**
- Exportar tudo para Excel para conseguir ver qualquer coisa cruzada.
- Nao saber atribuir receita real a campanha que gerou o lead.
- Automacao "quebrada" que ninguem percebeu por dias.

---

## 4. Master Admin (Milennials)

**Contexto.** Equipe interna da Milennials. Opera **cross-tenant**: enxerga e gerencia todas as orgs do Torque. Role `master` no sistema. Raro, alto privilegio, tudo auditado.

**Responsabilidades.**
- Gerenciar todas as orgs — criacao, suspensao, mudanca de plano, ajustes manuais.
- Audit logs cross-tenant — quem fez o que, onde, quando.
- Operations Center — saude do sistema, filas, jobs, integracoes WhatsApp/Meta.
- Planos — criar, editar, precificar, ativar/desativar.
- Feature flags — ligar/desligar feature por org, por plano, globalmente.
- Billing — disputas, reembolsos, inadimplencia Asaas.
- Suporte de alto nivel — impersonate com consentimento e rastro.

**Job-to-be-done principal.**
*"Quando uma org critica reporta problema ou quando eu preciso ajustar um plano, eu quero agir cross-tenant com escopo explicito, auditado e reversivel — sem jamais contaminar o tenant dela com meu acesso e sem depender de engenharia para operacoes rotineiras."*

**Telas criticas.**
- Master Admin — listagem de orgs com health score, plano, billing status.
- Master Admin — detalhe da org (com toggle de feature flag, impersonate auditado).
- Audit log cross-tenant — filtragem rica, export.
- Operations Center — filas, webhooks Meta, integracoes Asaas/TinyERP/SZ.Chat.

---

## 5. Personas externas

### Lead (cliente final da organizacao)

- **Nunca acessa o Torque diretamente**. Interage via canais de mensagem (WhatsApp, Messenger) com agentes humanos ou com agentes IA.
- **Expectativas**: conversa fluida, humanizada, que resolve seu problema ou agenda proxima etapa.
- **Sinal importante**: qualquer friccao percebida (resposta robotica, atraso excessivo, mensagens duplicadas) degrada a reputacao da organizacao.

### Integrador / Parceiro n8n

- **Responsabilidades**: construir fluxos no orquestrador externo que capturam leads (Trello, Meta Ads, forms) e os enviam para o webhook de ingestao.
- **Expectativas**: contrato de webhook estavel, payload documentado, idempotencia, resposta rapida.

---

## Resumo operacional

| Persona | Role de sistema | Escopo | Foco diario |
|---|---|---|---|
| SDR / Prospector | `membro` | Propria org, [[Pipe]]s atribuidos | Inbox + kanban de prospeccao |
| Closer / Executivo | `membro` | Propria org, [[Pipe]]s atribuidos | Kanban de fechamento + proposta |
| Gestor Comercial | `admin` | Propria org, todos os recursos | Dashboard + configuracao |
| Master Admin | `master` | Cross-tenant (todas as orgs) | Operacoes da plataforma |
| Lead | N/A | Externo — consome produto indiretamente | Canais de mensagem |
| Integrador n8n | N/A | Externo — API/webhook | Webhook de ingestao |

## Mapeamento Persona → Feature

| Persona | Features primarias |
|---------|-------------------|
| Admin | Tudo, com foco em Workflow Builder, Configuracoes, Analytics, Copilot wizard |
| SDR | Pipeline WhatsApp, Chat, Follow-ups, Templates |
| Closer | Pipeline Confirmacao/Propostas, Produtos, Calendario |
| Prospectador | Campanhas, Chat, Dashboard Outbound |
| Membro restrito | Subset configurado pelo admin |
| Master | Painel master, provisionamento, impersonacao, metricas cross-org |
| Lead | Nenhuma — consome produto indiretamente via canais |
| Integrador n8n | Webhook de ingestao, API publica |

## Implicacoes de design

- **Admin precisa configurar sem devs** → editores visuais para workflow, copilot, templates, pipelines customizados.
- **SDR/Closer usam o dia todo** → telas densas mas nao fatigantes, atalhos, realtime, dark-first.
- **Master e cauteloso** → toda acao impactante tem confirmacao dupla e fica em log de auditoria.
- **Lead precisa confiar** → agentes IA sao humanizados (SmartSplitMessage, batch de 8s, pausa em takeover), nunca se identificam como bots a menos que explicitamente configurado.
- **Integrador precisa estabilidade** → contrato do webhook de ingestao e versionado e backward-compatible.
