---
tipo: requisitos
---

# Regras de Negócio Globais

Regras que atravessam múltiplas features e são **invariantes** do sistema. Cada implementação deve respeitar.

## Isolamento de Tenant (RG-01)

> Nenhum dado de uma organização é visível, modificável ou inferível por outra organização.

- Aplicável em todas as camadas.
- Master admin é a única exceção; ações são auditadas.
- Falha fechada: sem tenant, sem acesso.

## Nomenclatura de Roles (RG-02)

> Roles no código são EXCLUSIVAMENTE `admin`, `master`, `membro`.

- SDR, Closer, Prospectador são `specialty` — não role.
- Novos papéis funcionais não exigem nova role.

## Webhook de Ingestão Público (RG-03)

> O contrato do webhook de ingestão de leads é público e estável.

- Backward-compatible nas versões `v1`.
- Breaking changes exigem nova versão (`v2`) e janela de transição.
- Formatos aceitos para tags/arrays mantidos.

## Idempotência (RG-04)

> Toda operação externa é idempotente.

- Webhook repetido não duplica.
- Mensagem enviada 2x não envia 2x ao lead.
- Retry de workflow não refaz passos concluídos.

## Auditoria Imutável (RG-05)

> Auditoria é append-only.

- Nunca UPDATE.
- Nunca DELETE (exceto retenção legal programada).
- Correção cria novo evento.

## Human Takeover (RG-06)

> Humano enviando mensagem pausa agente IA automaticamente por 10 min.

- Previne conflito humano-máquina.
- Janela configurável.
- Agente retoma ao expirar sem interação humana nova.

## Opt-out Global (RG-07)

> Tag `no-contact` ou `opt-out` impede qualquer envio outbound automatizado ao lead.

- Campanhas não enrolam.
- Workflows pulam.
- Mensagens agendadas cancelam.
- Respeitado em todos os canais.

## Janela de Negócio (RG-08)

> Envios outbound automatizados respeitam janela de negócio da org (se flag `respect_business_hours`).

- Mensagens fora da janela reagendam para próximo horário válido.
- Agente IA respeita conforme config.
- Mensagens urgentes (resposta a inbound ativo) podem ignorar.

## Quotas Enforced (RG-09)

> Limites de plano são enforced no backend, não só UI.

- Quota atingida bloqueia operação com mensagem explícita.
- Contador incrementado na mesma transação do recurso criado.
- Reset em ciclo mensal.

## Dados Sensíveis Nunca em Log (RG-10)

> Senhas, tokens, API keys, conteúdo completo de mensagens em `info` em produção NUNCA.

- `debug` level controlado.
- PII mascarada em logs de produção.
- Export e compartilhamento documentados em audit.

## Soft-Delete Preferencial (RG-11)

> Entidades importantes não são hard-deletadas por default.

- Soft-delete preserva histórico.
- Hard-delete apenas por admin-role + janela de carência + master em casos extremos.
- Dados anonimizados em LGPD pedido de exclusão (preserva integridade auditoria).

## Permissão Rechecada no Backend (RG-12)

> Cliente nunca é fonte de autoridade.

- UI esconde/desabilita, backend revalida.
- Nenhum endpoint confia em input do cliente para `organization_id`, `role`, `permissions`.

## Realtime Escopado (RG-13)

> Subscriptions realtime filtradas no servidor por organization_id.

- Cliente não pode "ouvir" dados de outra org.
- Token valida escopo ao subscribe.

## Status Derivado do Lead (RG-14)

> Lead não tem "status" unificado; é derivado dos pipeline entries.

- Sem campo `status` singular na tabela Lead.
- UI calcula derivação.

## Multi-canal por Padrão (RG-15)

> Nenhuma feature assume canal específico.

- Adaptador abstrai provedor.
- Adicionar canal novo = adicionar adaptador.

## Backward Compatibility (RG-16)

> API pública tem janela de deprecation mínima.

- Endpoint deprecated: 6 meses antes de sunset.
- Versões antigas coexistem (v1, v2).

## Atribuição Sem Espalhamento (RG-17)

> Lead tem até 3 atribuições (responsible, sdr, closer). Não criar novas relações de atribuição sem justificativa forte.

- Casos especiais usam tags ou custom fields.

## Formato de Phone (RG-18)

> Telefone armazenado SEMPRE em E.164 (`+5511987654321`).

- Normalização na ingestão.
- Validação estrita.
- Formato display local (UI) é apresentação, não armazenamento.

## Timezone (RG-19)

> Timestamps armazenados em UTC. Apresentação em fuso da org.

- Sem ambiguidade de fuso em banco.
- Cálculo de "hoje", "esta semana" sempre em fuso da org.

## Validação em Camadas (RG-20)

> Input validado em UI, aplicação, domínio, persistência.

- Cada camada é independente.
- Bug em uma não escapa pela outra.

## Nenhum Secret no Cliente (RG-21)

> Cliente (UI, mobile, integração) nunca recebe chave de serviço externo.

- API keys de providers ficam apenas no backend.
- Tokens de sessão são curtos e scoped.

## Decisão com Justificativa (RG-22)

> Ações sensíveis (remover membro, deletar org, rotacionar secret) registram justificativa quando aplicável.

- Audit log tem campo `reason`.
- Master impersonation obrigatoriamente.

## Degradação Graciosa (RG-23)

> Sistema degrada graciosamente em falha de integração externa.

- LLM falha → agente pausa, não envia resposta ruim.
- Canal offline → fila, não drop.
- Analytics falha → widget com erro, não tela branca.

## Testes de Isolamento (RG-24)

> Toda mudança em endpoint sensível passa por suite de teste de isolamento multi-tenant.

- Autentica como tenant A, tenta acessar tenant B → deve falhar.
- Automatizado em CI.

## Recomendado vs Obrigatório

Acima são obrigatórios (invariantes). Recomendadas adicionais:

- **Dark-first** em toda feature nova (preferência de design).
- **Mobile-friendly** em features de uso diário (SDR).
- **Acessibilidade WCAG AA** como alvo.
- **Documentação** atualizada a cada mudança (este vault + CLAUDE.md).
- **Rollback plan** para cada feature complexa em release.
