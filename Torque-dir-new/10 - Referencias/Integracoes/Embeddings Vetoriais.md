---
tipo: integracao
direcao: out
criticidade: media
---

# Embeddings Vetoriais

Serviço externo que gera **embeddings** (vetores de alta dimensão) a partir de texto. Usado para RAG (Retrieval-Augmented Generation) de FAQs dos agentes IA — permite busca semântica de "qual FAQ é mais relevante para a pergunta do lead?".

## Propósito

- Gerar vetor para cada FAQ question na criação/atualização.
- Gerar vetor para a pergunta do lead em tempo real.
- Calcular similaridade coseno e retornar top-K FAQs.
- Suportar outros usos futuros: clustering de leads similares, sugestão de templates, busca semântica de leads.

## Contrato

- Endpoint REST `/embeddings`.
- Input: texto (string) + modelo.
- Output: vetor numérico (dimensão fixa: ex. 1536).

## Provedor

Atualmente Google Gemini (`text-embedding-004` ou similar com 1536 dim). Alternativas: OpenAI (`text-embedding-3-small`), Voyage AI.

## Autenticação

- API key por Torque (não por org, centralizado).

## Endpoint Consumido

```
POST /embeddings
{
  "model": "text-embedding-004",
  "input": "Qual o prazo de entrega?"
}
```

Resposta:
```
{
  "data": [
    {
      "embedding": [0.12, -0.34, ..., 0.07],  // 1536 dims
      "index": 0
    }
  ],
  "model": "text-embedding-004",
  "usage": {"total_tokens": 8}
}
```

## Banco Vetorial

Após gerar embedding, armazena em banco vetorial (extensão de Postgres, Pinecone, Qdrant, etc.) com metadados:
- `faq_id`.
- `agent_id`.
- `organization_id`.
- `vector` (float array).
- `text_snapshot` (texto original).

Indexação: HNSW ou IVF para busca aproximada (kNN).

## Operações

### Indexar FAQ
1. Admin cria/edita FAQ.
2. Job assíncrono gera embedding.
3. Armazena no banco vetorial.
4. Flag `last_indexed_at` atualizado.

### Busca (RAG)
1. Agente recebe mensagem do lead.
2. Gera embedding da mensagem.
3. Query no banco vetorial:
   ```
   SELECT faq_id, similarity(vector, query_vector) as score
   WHERE organization_id = X AND agent_id = Y
   ORDER BY vector <-> query_vector  -- cosine distance
   LIMIT 5
   ```
4. Filtra por threshold (ex.: similaridade ≥ 0.7).
5. Retorna top-K FAQs.
6. Inclui no contexto do LLM.

## Regras de Negócio

1. FAQ criada recentemente sem embedding: exclusa da busca até indexação.
2. Embedding reindexado se modelo muda (migração).
3. Banco vetorial filtra obrigatoriamente por `organization_id` **antes** de calcular similaridade (não por baixo do index).
4. Dedupe: mesma FAQ (hash do texto) não reindexada se não mudou.
5. Falha na geração: retry; após esgotar, FAQ fica sem embedding mas visível na UI (indicador "não indexada").

## Edge Cases

- **Modelo indisponível**: retry com backoff; fallback para modelo alternativo (se configurado).
- **Dimensão incompatível** (trocou modelo com dim diferente): migração planejada, reindex completo.
- **Texto muito longo**: limite de tokens do provider; truncar ou split.
- **Texto vazio/só espaços**: rejeita.
- **Múltiplas orgs com mesma FAQ texto**: embeddings separados (indexação por org).

## Performance

- Embedding single: ~100-300ms.
- Query kNN em 1000 vetores: < 10ms.
- Cache de embeddings de pergunta do lead: útil se mesma pergunta chega várias vezes.

## Custo

- Tokens × preço do modelo de embedding.
- Barato comparado a LLM.
- Monitora por org (FAQs × média tokens × price).

## Segurança

- API key em vault.
- HTTPS.
- Contrato com provider: não usar para treinar.

## Observabilidade

- Log por chamada: duração, tokens.
- Métricas: embeddings gerados/dia, busca rate, hit rate (quantas FAQs atingem threshold).

## Evolução

- Embeddings multilingues unificados (para clientes com lead em diferentes idiomas).
- Embeddings de outras entidades (produtos, leads, conversas).
- Suporte a modelos self-hosted (sentence-transformers) para quem quer controle total.
