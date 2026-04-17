---
tags:
  - features
  - template
  - feature-spec
type: feature-spec
created: 2026-04-15
last_updated: 2026-04-15
status: draft
---

# Template — Feature Spec

> Copie este arquivo ao criar uma nova feature. Preencha todas as seções. Seções vazias significam decisões não tomadas — e decisões não tomadas reprovam a feature.

## Visão

1 parágrafo: **o que é**, **para quem**, **por que agora**. Se não couber em um parágrafo, a feature não está clara o suficiente para ser construída.

## Jobs-to-be-done

Lista de jobs concretos que o usuário contrata essa feature para executar.

- Quando [situação], eu quero [ação], para [resultado].

## Telas e rotas

- `/rota` — descrição curta
- `/rota/:id` — descrição curta

## Componentes novos

Componentes de domínio que **nascem com essa feature**. Primitivos genéricos vão para `ui/` no Sistema Base.

- `NomeDoComponente` — responsabilidade

## Contratos de API

Endpoints consumidos + shapes.

- `METHOD /caminho` — descrição
	- Request: `{ ... }`
	- Response: `{ ... }`

## Eventos de realtime consumidos

Tópicos WS que a feature escuta.

- `entidade.evento` — efeito no client (invalidation, patch, toast, etc.)

## Permissões relevantes

Permissões necessárias para operar essa feature.

- `permission.name` — onde é checada

## Estados de UI

Enumeração explícita:

- **Loading:** descrição do skeleton
- **Empty:** copy + CTA
- **Error:** copy + ação de retry
- **Realtime updating:** sinal visual
- **Saving:** feedback ao usuário

## Edge cases

Casos que quebram soluções ingênuas.

- Caso — comportamento esperado

## Métricas de sucesso

Como saberemos que está bom.

- Métrica — meta

## Dependências

- Outras features: `[[F0X ...]]`
- Contratos: entidades envolvidas
- Componentes: primitivos reutilizados

## Decisões tomadas

- Decisão — razão — link para ADR se houver

## Fora de escopo

O que **não** entra nessa feature (vai para outra, ou fica para depois).

## Checklist de conclusão

Arquivo separado `Checklist de Conclusao.md` dentro da pasta da feature. Só avança para a próxima feature com 100%.

## Referências

- [[00 - Mapa de Features]]
- [[Escopo do Sistema Base]]
