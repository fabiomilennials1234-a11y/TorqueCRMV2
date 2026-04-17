---
tipo: referencia
---

# Catálogo de APIs Externas

Resumo das APIs externas consumidas pelo Torque. Cada linha aponta para doc detalhada.

## Canais de Mensagem

| Provedor | Uso | Doc detalhada |
|---|---|---|
| Evolution API | WhatsApp primário | [[05 - Integrações Externas/WhatsApp (Evolution API)]] |
| SZ.Chat | WhatsApp alternativo | [[05 - Integrações Externas/SZ.Chat]] |
| Meta Graph API | Messenger, Lead Ads | [[05 - Integrações Externas/Meta (Facebook Ads e Messenger)]] |

## Calendário

| Provedor | Uso |
|---|---|
| Google Calendar API v3 | Agendamentos e sync bidirecional |

## ERP

| Provedor | Uso |
|---|---|
| TinyERP REST API | Sync de produtos, criação de pedidos |

## Pagamento

| Provedor | Uso |
|---|---|
| Asaas | Cobrança recorrente (assinaturas + clientes finais opt-in) |

## IA e LLM

| Provedor | Uso |
|---|---|
| OpenRouter | Roteamento de LLMs (Claude/GPT/Gemini/etc.) |
| Google Gemini | Embeddings 1536d |
| ElevenLabs | Text-to-Speech |

## Observabilidade

| Provedor | Uso |
|---|---|
| Sentry | Exception capture, release health |

## Orquestração externa (consumidor, não consumido)

| Ferramenta | Uso |
|---|---|
| n8n | Cliente usa para ingestão; consome webhook público do Torque |

## Storage de Objetos

| Provedor | Uso |
|---|---|
| Storage S3-compatible | Áudios, imagens, documentos, anexos |

## Autenticação

| Provedor | Uso |
|---|---|
| OAuth 2.0 (Google, Meta) | Integrações per-user / per-org |

## Tipos de Autenticação Usadas

| Tipo | Exemplos |
|---|---|
| API Key em header | Evolution, SZ.Chat, Asaas, TinyERP, OpenRouter, ElevenLabs, Gemini |
| OAuth 2.0 | Google Calendar, Meta |
| HMAC (webhooks inbound) | Meta, Asaas, outros |
| Token na URL (webhooks) | Evolution, SZ.Chat |
| mTLS (interno) | Entre serviços internos |

## Rate Limits Agregados (referência)

| Provedor | Limite típico |
|---|---|
| Evolution | ~100 msg/h por número (WhatsApp não-oficial) |
| Meta Graph | ~200 req/hora/token |
| Google Calendar | ~1M req/dia/project |
| TinyERP | Varia por plano cliente |
| Asaas | Varia por plano |
| OpenRouter (LLM) | Varia por modelo |
| Gemini embeddings | ~1500 req/min |
| ElevenLabs | Varia por plano |

## Latências Aceitáveis (p95)

| Provedor | Esperado |
|---|---|
| Canal (envio msg) | < 3s |
| LLM chat completion | < 5s |
| Embeddings | < 500ms |
| TTS | < 5s para áudio de 30s |
| Calendar API | < 2s |
| Asaas payment | < 5s |
| TinyERP | < 3s |

## Timeout Configurado

| Provedor | Timeout |
|---|---|
| Canais (envio) | 15s |
| LLM | 30s |
| Embeddings | 10s |
| TTS | 30s |
| Calendar | 20s |
| Asaas | 30s |
| Outros | 10s default |

## Custos Estimados (referência)

Não documentar valores exatos aqui (mudam). Rastrear em doc interna.

Categorias:
- Canal de mensagem: fixo mensal + variável por msg.
- LLM: variável por token.
- Embeddings: variável por token (barato).
- TTS: variável por char.
- Outros: conforme plano.

## Fallbacks Documentados

- **LLM down**: fallback para outro modelo via OpenRouter.
- **Canal primário down**: admin migra para alternativo manualmente (ou feature flag).
- **Calendar down**: lembretes internos do Torque funcionam.
- **ERP down**: admin cadastra manualmente.
- **Asaas down**: admin faz cobrança fora.

## Segurança

- Todas em HTTPS.
- Secrets em vault cifrado.
- Rotação documentada.
- IP allowlist onde suportado.
- Webhook signature validation obrigatória.

## Compliance

- LGPD: titular sabe que dados são compartilhados (consentimento).
- Contratos com providers incluem DPA.
- Retenção de dados respeitada.

## Substituibilidade

Cada integração tem **adaptador** interno estável. Trocar provider = trocar adaptador. Features downstream não precisam mudar.
