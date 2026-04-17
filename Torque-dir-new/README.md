---
tipo: root
versao: 1.0
data: 2026-04-15
---

# Torque CRM — Documentação Funcional Completa (v1)

> Documentação **agnóstica de tecnologia** do Torque CRM. Escrita para permitir **reconstrução completa do sistema em qualquer linguagem ou stack** — Python, Go, Java, Rust, C#, Node, Elixir, Ruby. Zero dependência de detalhes da implementação atual.

## Sobre este Vault

Este vault é **auto-contido**. Descreve todo o produto Torque CRM em termos conceituais:

- **O que o produto é** e para quem.
- **Como ele funciona** funcionalmente (features, fluxos, regras).
- **Quais são as entidades** e suas invariantes.
- **Como se integra** com serviços externos.
- **Como se comporta** em cenários assíncronos, sob carga, em falha.
- **Quais decisões** foram tomadas e por quê.

Nenhum doc cita React, Supabase, TypeScript, nomes de tabela SQL, hooks, edge functions, ou qualquer detalhe específico da stack atual. O que é imutável — regras de negócio, comportamento, contratos externos — está documentado. O que é mutável — escolhas de tecnologia — não está.

## Como Navegar

Comece por [[00 - Visão Geral/Índice]] que é o índice completo.

## Estrutura

```
Torque-dir-new/
├── 00 - Visão Geral/              Introdução ao produto e princípios
├── 01 - Arquitetura Conceitual/   Camadas, multi-tenancy, padrões, NFRs, segurança
├── 02 - Modelo de Domínio/        Entidades e invariantes
├── 03 - Identidade e Permissões/  Auth, roles, escopo, isolamento
├── 04 - Funcionalidades/          Features organizadas por domínio
│   ├── Vendas/                    (9 docs)
│   ├── Comunicação/               (4 docs)
│   ├── IA/                        (3 docs)
│   ├── Automação/                 (3 docs)
│   ├── Equipe/                    (4 docs)
│   ├── Analytics/                 (7 docs)
│   └── Admin/                     (7 docs)
├── 05 - Integrações Externas/     Contratos com serviços de terceiros (12 docs)
├── 06 - Fluxos End-to-End/        Jornadas completas que cruzam features (7 docs)
├── 07 - Processos Assíncronos/    Jobs, filas, cron (5 docs)
├── 08 - Requisitos e Regras/      Regras globais, invariantes, gotchas, decisões (4 docs)
└── 09 - Referências/              Eventos, APIs externas, estados, quotas (4 docs)
```

Total: ~80 documentos cobrindo o sistema completo.

## Princípios da Documentação

1. **Tech-agnostic**: abstrações, não tecnologias.
2. **Conceitual + Comportamental**: o quê e como, não como-implementar.
3. **Dados como atributos lógicos**: não colunas SQL.
4. **Fluxos em prosa + pseudo-diagramas**: legíveis sem tooling específico.
5. **Estados explícitos**: máquinas de estado sempre documentadas.
6. **Edge cases sempre**: cada feature declara cenários atípicos.
7. **Invariantes claros**: o que nunca pode mudar.
8. **Integrações em contrato**: entrada/saída por serviço externo.

## Quem Deve Ler

- **Engenheiros** reimplementando em nova stack.
- **Arquitetos** validando decisões.
- **Product Managers** alinhando com desenvolvimento.
- **Novos membros do time** entendendo o todo.
- **Auditores** (segurança, compliance) verificando coerência.

## Ordem de Leitura Recomendada

### Para reimplementação completa
1. [[00 - Visão Geral/O que é o Torque]]
2. [[00 - Visão Geral/Princípios do Sistema]]
3. [[01 - Arquitetura Conceitual/Visão Geral]] + [[Multi-tenancy]] + [[Camadas do Sistema]]
4. [[02 - Modelo de Domínio/Visão Geral]] + [[Entidades Principais]]
5. [[03 - Identidade e Permissões/Modelo de Permissões]]
6. [[06 - Fluxos End-to-End/Lifecycle de um Lead]]
7. Ler cada feature em `04 - Funcionalidades/` conforme prioridade da implementação.
8. Consultar [[09 - Referências/]] conforme necessidade técnica.

### Para entender negócio
1. [[00 - Visão Geral/O que é o Torque]]
2. [[00 - Visão Geral/Personas e ICP]]
3. [[06 - Fluxos End-to-End/Lifecycle de um Lead]]
4. Navegação livre por features de interesse.

### Para validar decisões
1. [[08 - Requisitos e Regras/Decisões de Design]]
2. [[08 - Requisitos e Regras/Regras de Negócio Globais]]
3. [[08 - Requisitos e Regras/Validações e Invariantes]]

## Versão

Esta documentação reflete o Torque CRM em **abril de 2026**. Atualize conforme o produto evolui.

## Meta

Criado por compilação de:
- Código atual do Torque.
- Wiki existente (`Torque-wiki`, `Claude Code — Torque CRM`).
- CLAUDE.md do projeto.
- Experiência operacional documentada.

Todas as afirmações sobre o sistema foram extraídas dos docs existentes e do código. Reinterpretadas e reescritas em forma agnóstica de stack.

## Contribuição

Ao atualizar:
- Manter estilo agnóstico (nunca citar React/Supabase/stack atual).
- Novas features: criar doc no domínio correto.
- Novas integrações: em `05 - Integrações Externas/`.
- Mudanças em regras globais: atualizar [[08 - Requisitos e Regras/Regras de Negócio Globais]].
- Eventos novos: adicionar em [[09 - Referências/Catálogo de Eventos]].

## Valor Final

**Este vault é reconstrução-ready.**

Um time de 3-5 engenheiros experientes consegue reimplementar o Torque em qualquer stack moderna lendo este vault + adquirindo credenciais dos provedores externos. Sem precisar do código atual.

Isso é o objetivo da documentação world-class: **independência de implementação**, **preservação de intenção**, **continuidade de conhecimento**.
