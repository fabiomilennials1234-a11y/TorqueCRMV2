---
tipo: integracao
---

# Integrações Externas — Visão Geral

Mapa de todas as integrações com serviços externos. Cada integração é abstraída atrás de um adaptador — trocar o provedor de um canal, por exemplo, é trocar o adaptador, não reescrever a feature.

## Matriz de Integrações

| Integração | Direção | Criticidade | Fallback | Doc |
|---|---|---|---|---|
| WhatsApp (Evolution) | in + out | Alta | Canal alternativo | [[WhatsApp (Evolution API)]] |
| SZ.Chat (WhatsApp alternativo) | in + out | Média | WhatsApp primário | [[SZ.Chat]] |
| Meta Business (Facebook, Messenger, Ads) | in + out | Alta | — | [[Meta (Facebook Ads e Messenger)]] |
| Google Calendar | bidirectional | Média | Calendário manual | [[Google Calendar]] |
| TinyERP (ERP produtos) | in + out | Média | Cadastro manual | [[TinyERP]] |
| Asaas (provedor de pagamento) | in + out | Alta | Alternativa manual | [[Asaas (Provedor de Pagamento)]] |
| n8n (orquestrador) | in | Alta | API direta | [[n8n (Orquestrador Externo)]] |
| Modelo LLM generativo | out | Alta | Fallback a outro LLM; pausa agente | [[Modelo LLM Generativo]] |
| Embeddings vetoriais | out | Média | — | [[Embeddings Vetoriais]] |
| TTS (áudio) | out | Baixa | Texto | [[Text-to-Speech]] |
| Observabilidade (Sentry) | out | Baixa | Log local | [[Observabilidade (Sentry)]] |

## Princípios Comuns

### Adaptador
- Cada integração tem interface interna estável.
- Adaptador traduz para protocolo externo.
- Trocar provedor = trocar adaptador.

### Credenciais por Tenant
- Tokens OAuth, API keys: por organização.
- Vault isolado com acesso controlado.
- Rotação documentada.

### Idempotência
- Webhooks recebidos: dedupe por `external_id` ou assinatura do payload.
- Chamadas outbound: `Idempotency-Key` quando suportado pelo provedor.

### Retry
- Backoff exponencial.
- Circuit breaker por provedor.
- Dead letter para falhas persistentes.
- Alertas quando degradação.

### Observabilidade
- Log estruturado por chamada.
- Métricas: latência, taxa de erro, volume.
- Dashboard de saúde por integração.

### Segurança
- Allowlist de URLs para SSRF.
- Validação de webhook (HMAC ou secret).
- Secrets nunca em log.

### Failure Mode
- Cada integração declara seu fallback.
- Alertas ao admin quando provider está down.
- UI mostra banner quando feature depende de integração indisponível.

## Configuração

- Admin conecta integração em `Configurações → Integrações`.
- Wizard por integração (OAuth fluxo, ou form com credenciais).
- Test de conexão.
- Logs acessíveis.
- Desconectar a qualquer momento.

## Evolução

- Adicionar nova integração: criar adaptador + wizard + doc.
- Trocar provider existente: novo adaptador, migração de credenciais, feature flag para transição.
- Descontinuar: desconexão forçada + alerta aos afetados.

## Quotas por Integração

- Cada integração pode ter quota própria (baseada no contrato com o provider).
- Sistema controla para não exceder e ser banido.
- Alerta a admin quando > 80% do mês.
