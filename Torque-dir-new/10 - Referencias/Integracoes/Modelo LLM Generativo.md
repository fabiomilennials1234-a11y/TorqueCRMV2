---
tipo: integracao
direcao: out
criticidade: alta
---

# Modelo LLM Generativo

Serviço externo de modelo de linguagem usado por agentes IA (copilot) e pelo Oráculo Comercial. Gera respostas conversacionais estruturadas em linguagem natural.

## Propósito

- Gerar resposta natural do agente IA a leads.
- Interpretar pergunta do usuário para text-to-query do Oráculo.
- Avaliar qualidade de conversas.
- Gerar conteúdo (FAQs sugeridas, exemplos de conversa).

## Contrato

- Endpoint REST de "chat completion".
- Input: lista de mensagens (system + messages histórico + prompt final).
- Output: mensagem assistente + possivelmente function call / tool use.
- Suporta temperature, max_tokens, response_format (JSON mode), streaming.

## Autenticação

- API key (do provedor; o Torque centraliza em vault).
- Por **organização** (se Torque oferece uso de modelo próprio do cliente) OU **globalmente** (Torque paga; repassa custo no plano).

## Provedor

Atualmente via OpenRouter — que permite uso de múltiplos modelos (Claude, GPT, Gemini, LLaMA, etc.) por uma única API. Escolha de modelo por:
- Qualidade necessária (agente premium usa top-tier).
- Custo (modelos mais baratos para FAQs simples).
- Latência (modelos menores para batch).

## Endpoints Consumidos

### Chat Completion
```
POST /chat/completions
{
  "model": "provedor/modelo-x",
  "messages": [
    {"role": "system", "content": "..."},
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."},
    ...
  ],
  "temperature": 0.7,
  "max_tokens": 1024,
  "response_format": {"type": "json_object"},
  "stream": false
}
```

Resposta:
```
{
  "id": "...",
  "model": "...",
  "choices": [
    {
      "message": {"role": "assistant", "content": "..."},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 512,
    "completion_tokens": 128,
    "total_tokens": 640
  }
}
```

### Streaming (para UI realtime, se aplicável)
- Server-sent events com chunks.
- Torque tipicamente não streamia ao lead (envia mensagem completa), mas pode usar em playground.

## Fluxo Típico (Agente IA respondendo)

1. Sistema monta mensagens (ver [[04 - Funcionalidades/IA/Copilot (Agentes IA)]]#fluxo-conversacional).
2. POST para LLM com timeout 30s.
3. Resposta chega.
4. Parse JSON (se response_format JSON).
5. Valida schema: `{message: str, actions: [...]}`.
6. Executa actions permitidas.
7. Envia message ao lead.

## Regras de Negócio

1. Timeout 30s; além disso assume failure.
2. Retry 1x em erro transiente (500, 429, timeout).
3. Após falha: pausa agente + notifica admin.
4. Parsing JSON inválido: retry com reforço de instrução; se falha, pausa.
5. Action fora de `allowed_actions`: descarta (não executa).
6. Custo rastreado: tokens × preço do modelo.
7. Budget por organização/mês: hardcap.
8. Fallback para modelo mais barato se custo crítico.

## Prompt Caching (se provedor suporta)

- System prompt grande (business_context + FAQs) pode ser cacheado.
- Reduz custo significativamente em conversas longas.
- Aplicação do cache transparente; métricas mostram hit rate.

## Edge Cases

- **Modelo indisponível**: fallback para outro modelo equivalente (OpenRouter permite).
- **Rate limit provider**: backoff + enfileirar.
- **Resposta em idioma errado**: reprocess com instrução explícita.
- **Resposta tóxica/inapropriada**: filtro de saída (heurística ou moderation API); se positivo, pausa e alerta.
- **Prompt injection**: input do lead não injeta instrução — sistema delimita claramente.
- **Alucinação de fato**: agente é instruído a não inventar; FAQ RAG limita; ainda assim, falhas acontecem → avaliação qualidade.
- **Saída muito longa**: max_tokens limita; Smart Split chunka.

## Custo

- Tokens input + output × preço por 1M tokens (varia por modelo).
- Torque monitora consumo por org/dia.
- Alerta em consumo anômalo.

## Observabilidade

- Log por chamada: modelo, tokens, latência, resultado.
- Métricas: tokens/dia, custo/dia, latência p95.
- Dashboard por agente: quanto consome, qualidade.

## Segurança / Privacidade

- Contrato com provedor: **não usar conteúdo para treinar modelos** (planos business).
- Conteúdo de conversa (PII) enviado ao modelo: aceitável conforme ToS do cliente.
- Logs de conteúdo de mensagem não em `info` em produção.
- Rotação de API key.

## Quotas

- Tokens/mês por plano (Torque).
- Quando excede: agente pausa, admin é notificado para fazer upgrade ou ajustar.

## Evolução

- **Modelos self-hosted**: uma org enterprise pode usar modelo próprio (Ollama, vLLM internal) → Torque aponta endpoint diferente.
- **Model routing inteligente**: roteia dinamicamente entre modelos por custo/qualidade.
- **Fine-tuning**: treinar modelo dedicado a partir de conversas (respeitando privacidade).
