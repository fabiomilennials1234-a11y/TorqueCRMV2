---
tags: [adr, architecture]
created: 2026-04-15
last_updated: 2026-04-15
status: accepted
id: ADR-001
---

# ADR-001 — Contratos OpenAPI com fronteira snake/camel

## Contexto

O novo backend em Go substitui a camada Supabase que antes servia de fonte de verdade implícita para os tipos do front. Essa troca remove tanto o schema PostgREST quanto os tipos gerados automaticamente pelo CLI do Supabase, o que deixa o front sem nenhuma garantia de sincronia com os contratos do servidor. Tipos escritos à mão em `src/types/` se descolam do backend em questão de semanas, e esse descolamento é a raiz de uma categoria inteira de bugs — campos renomeados que ninguém percebe até produção, payloads com shape divergente, enums com valores fantasma.

Há uma tensão idiomática entre as duas stacks. Go idiomático usa `snake_case` em tags JSON e em nomes de coluna no Postgres, tanto pela convenção da linguagem quanto pelo alinhamento com o banco. JavaScript e TypeScript idiomáticos usam `camelCase` em propriedades e variáveis, e qualquer código de UI que acesse `lead.first_name` em vez de `lead.firstName` fere a coesão do codebase e aparece imediatamente em code review. Tentar resolver isso no servidor (forçando camelCase no JSON do Go) polui o backend; tentar resolver no front ad-hoc gera código inconsistente.

A decisão precisa resolver três problemas simultaneamente: sincronia estrutural entre servidor e cliente, naming idiomático de cada lado, e ergonomia do dia-a-dia para quem escreve código de UI.

## Decisão

OpenAPI é a fonte única de verdade dos contratos, gerado pelo Go a partir das rotas e structs tipadas, e consumido no front por um pipeline determinístico.

- O servidor Go gera `openapi.yaml` a partir dos handlers (via `kin-openapi` ou equivalente idiomático) no build.
- O front roda `openapi-typescript` contra esse `openapi.yaml` e emite `src/contracts/api.gen.ts`. Esse arquivo nunca é editado à mão.
- A serialização on-the-wire é `snake_case` em ambas as direções — entrada e saída — preservando idiomaticidade do Go e alinhamento com colunas do Postgres.
- A fronteira entre wire e domínio do front vive em `src/api/`. Cada recurso tem um transformer declarativo que converte `snake_case` em `camelCase` na entrada e o inverso na saída. A partir dessa camada, nenhum código de UI vê `snake_case`.
- O transformer é declarativo (mapa de campos) e não usa conversão genérica por regex, para preservar renames intencionais e campos com casing especial.

## Alternativas consideradas

- **tRPC** — colar tipos diretamente de servidor a cliente via inferência TypeScript — descartado porque o backend é Go, não Node, e tRPC é fundamentalmente acoplado ao runtime JS.
- **gRPC-Web + protobuf** — contratos em `.proto`, geração para Go e TS, transport binário — descartado pelo overhead de tooling (envoy/proxy, plugins buf, debug binário no devtools) sem ganho de performance que justifique para cargas JSON medianas.
- **Zod-first** — schemas Zod no front como fonte de verdade, servidor valida a partir de schemas espelhados — descartado porque a fonte de verdade deve ser o servidor; o cliente não dita shape para o sistema.
- **Hand-written types** — manter `src/types/` escrito à mão e disciplina de code review — descartado porque o descolamento é matematicamente garantido conforme o time cresce e o ritmo de mudanças aumenta.

## Consequências

**Positivas**

- Tipos do front e contratos do backend permanecem em sincronia por construção, não por disciplina.
- Refactors cross-stack (rename de campo, mudança de enum) quebram o build do front imediatamente, onde o custo de corrigir é menor.
- A documentação de API é subproduto do processo, não artefato separado que envelhece.
- A fronteira snake/camel é explícita e auditável, não espalhada pelo codebase.

**Negativas**

- O front depende da qualidade do OpenAPI emitido pelo Go; structs Go mal anotadas geram tipos ruins no front.
- O build do front precisa do `openapi.yaml` disponível, o que adiciona um passo no pipeline e uma dependência cross-repo em dev local.
- O transformer snake↔camel é código que precisa ser mantido e testado, ainda que declarativo.
- Campos adicionados no servidor só aparecem no front após re-geração, o que pode criar uma leve janela de defasagem em dev.

## Impacto

- `src/contracts/api.gen.ts` — artefato gerado, nunca editado à mão.
- `src/api/` — transformers, clients tipados por recurso, ponto único de serialização.
- `src/lib/fetch.ts` — respeita content-type e delega transformação à camada `api/`.
- `package.json` — scripts `generate:types` e hook pré-build.
- Pipeline de CI — passo de regeneração e verificação de drift do arquivo gerado.

## Próximos passos

1. Criar `src/contracts/manual.ts` como rascunho enquanto o Go não emite OpenAPI completo.
2. Configurar script `generate:types` invocando `openapi-typescript` contra URL ou arquivo local.
3. Implementar um transformer de referência em `src/api/leads.ts` para servir de template.
4. Documentar o workflow de atualização (quando regerar, como detectar drift em CI).

## Links

- [[Contratos e Tipos]]
- [[Arquitetura do Front]]
- [[ADR-003-auth-httponly-cookies-samesite]]
- [[ADR-004-paginacao-cursor-based]]
