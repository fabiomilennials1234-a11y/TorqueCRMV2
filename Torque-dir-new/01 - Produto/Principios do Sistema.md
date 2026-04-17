---
tipo: visao-geral
---

# Princípios do Sistema

Os princípios abaixo são **invariantes de design**. Toda decisão arquitetural, toda feature nova, toda correção deve respeitar esses princípios. Quando há conflito entre princípios, o mais acima vence.

## 1. Isolamento absoluto entre tenants

Nenhum dado de uma organização pode ser visto, listado, enumerado ou inferido por outra organização, sob nenhuma circunstância. O sistema deve falhar fechado: na ausência de um escopo de organização válido, a operação é negada. Master admin é a única exceção, e toda ação master deixa rastro de auditoria.

## 2. Toda entrada externa é hostil

Webhooks, inputs de formulário, parâmetros de URL, payloads de API — tudo passa por validação estrita. Tamanhos, tipos, formatos, limites. Nenhuma ação de efeito colateral é executada sobre input não-validado. Segredos (API keys, tokens) nunca entram em log.

## 3. Eventual é aceitável, inconsistente não é

Operações podem ser assíncronas (processamento em worker, batch, fila), mas o estado final deve ser sempre consistente e auditável. Toda operação de efeito colateral é idempotente por padrão (chave de dedupe, retry seguro, dead letter rastreável).

## 4. Tempo real onde usuário olha, batch onde usuário não olha

Kanbans, chat e contadores visíveis são realtime (subscriptions com debounce pequeno). Jobs recorrentes (follow-ups, campanhas, distribuição, scoring) rodam em batch via agendador. O sistema não acorda um worker por cada evento: agrega, processa em janela, controla custo.

## 5. Humanização da IA não é opcional

Agentes IA conversacionais agrupam mensagens em janela curta antes de responder (não respondem a cada tecla do usuário), chunkam respostas longas em mensagens consecutivas naturais, pausam quando um humano assume a conversa, e nunca são robóticos em tom. Agente IA é um vendedor — não um chatbot.

## 6. O admin não precisa de dev para configurar

Workflows, agentes IA, regras de pipeline, templates, campanhas — todos têm editores visuais. Nenhuma operação comum exige abertura de ticket para engenharia. Quando há limite (plano, quota), é comunicado com clareza na UI.

## 7. Dark-first, world-class

A interface é dark-first por padrão. Design referência: Apple, Airbnb, Linear, Stripe, Vercel. Tipografia editorial, densidade inteligente, microinterações suaves. Nunca template genérico — se parece SaaS de prateleira, falhou.

## 8. Multi-canal desde o início

Nenhuma feature pode assumir "é WhatsApp". Todo envio/recepção de mensagem passa por uma abstração de canal. Adicionar um novo canal é trocar um adaptador, não reescrever feature.

## 9. Observabilidade embutida

Todo job assíncrono, todo webhook, toda execução de workflow e toda interação com IA emite telemetria: início, fim, duração, resultado, erros, contexto (organização, usuário, entidade). Logs estruturados, não texto corrido. Sentry ou equivalente captura exceções com contexto de tenant.

## 10. Quotas são contratuais

Limites de plano (leads/mês, mensagens/mês, agentes, FAQs) são enforçados no backend, não só na UI. Quando o limite é atingido, a operação é bloqueada com mensagem explícita e o admin é notificado. Quotas nunca "se resolvem sozinhas" — o sistema não cobra extra silenciosamente.

## 11. Idempotência é padrão, não exceção

Webhook recebido duas vezes não gera dois leads. Mensagem enviada duas vezes não envia duas vezes para o cliente. Workflow retriado não duplica ações. Chave de idempotência explícita em toda operação externa.

## 12. Backward-compatibility do webhook público

O webhook de ingestão de leads é contrato público consumido por integradores externos (n8n, Zapier, Make). Nenhuma mudança breaking sem versão nova e período de transição.

## 13. Role no código ≠ role na UI

No código, roles são apenas `admin`, `master`, `membro`. SDR, Closer, Prospectador são **conceitos de UI e negócio**, implementados via especialização de membro, não via role. Novas especializações não exigem nova role.

## 14. Fallback gracioso de IA

Se o modelo LLM falhar, timeout, ou retornar resposta inválida, o sistema degrada graciosamente: pausa o agente, notifica o admin, não envia mensagem ruim para o lead. Nunca alucina action (ex.: mover stage para stage inexistente).

## 15. Auditoria é imutável

Histórico de ações em leads, mudanças de stage, entregas de mensagem, execuções de workflow — tudo é append-only. Correção retrospectiva cria novo evento, não edita evento antigo.

## 16. Performance é restrição de design

Kanbans abrem em <1s com dezenas de colunas e centenas de cards. Analytics com filtros de 90 dias respondem em <2s. Agregações pesadas são materializadas/cacheadas. O padrão é ver-rápido, não carregar-então-ver.

## 17. Segurança desde o primeiro commit

Nunca expor chaves de serviço no cliente. Toda função sensível valida authority antes de agir. Senhas em hash. Tokens em storage seguro. Secrets em vault separado de código. Headers de segurança (CSP, HSTS) ligados por padrão.

## 18. Se parece template, reprovou

Cada tela é uma decisão. Nada é cópia de kit pronto sem intenção. Cores, espaçamentos, microinterações, tons de voz — tudo intencional.

---

Esses princípios são critério de aprovação de qualquer mudança. Revisão de código, revisão de design e revisão de spec usam essa lista.
