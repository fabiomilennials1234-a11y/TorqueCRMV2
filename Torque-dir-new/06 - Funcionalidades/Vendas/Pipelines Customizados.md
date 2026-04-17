---
tipo: feature
dominio: vendas
pipeline: custom
---

# Pipelines Customizados

## Propósito

Permite que a organização crie **funis próprios** além dos três estruturais (WhatsApp, Confirmação, Propostas). Útil para processos complementares: pós-venda, onboarding de cliente novo, recuperação de inadimplente, upsell, processo de contratação (se org usar para RH interno), etc.

## Atores e Permissões

- **Admin**: cria, edita, deleta.
- **Membros**: acesso de visualização/movimento configurado pelo admin caso a caso.
- **Agente IA**: opera conforme Kanban Rules do custom pipe (configurável).
- **Workflow**: disparos e actions sobre custom pipes com mesma rigidez dos estruturais.

Ações: `pipeline.view:{custom_id}`, `pipeline.move_entry:{custom_id}`, `pipeline.create_custom`, `pipeline.edit_config`, `pipeline.delete_custom`.

## Dados Envolvidos

Mesmo modelo de Pipeline + Stage + Entry descrito em [[02 - Modelo de Domínio/Pipeline]]. Atributos adicionais:

- `icon`: ícone customizável.
- `color`: cor do pipe.
- `badge_mode`: como mostrar badge no menu (contagem de entries ativos).
- `access_mode`: `all_members` | `specific_members` | `admin_only`.

## Regras de Negócio

1. Admin cria pipe com nome único dentro da organização.
2. Admin adiciona stages, define ordem, cor, `is_final` (nenhuma, positiva, negativa).
3. Admin define regra de entrada: manual, via workflow, via regra de pipe de outro funil.
4. Admin define quem pode ver/mover.
5. Deletar pipe com entries ativos: UI exige migração dos entries (para outro pipe ou cancelamento) antes.
6. Nome de stage único dentro do pipe.
7. Lead pode estar em custom pipe + pipes estruturais simultaneamente sem conflito.

## Fluxos do Usuário

### Criar Pipeline Customizado
1. Admin abre `Funis Hub → Novo Pipeline`.
2. Form:
   - Nome.
   - Descrição.
   - Ícone e cor.
   - Stages (adiciona dinamicamente; define nome, ordem, cor, is_final).
   - Permissões de acesso.
3. Preview do kanban.
4. Salvar → pipe ativo (ou draft se admin escolher).

### Editar
1. Admin clica em "Editar Pipeline".
2. Pode:
   - Renomear.
   - Adicionar / remover / reordenar stages.
   - Mudar permissões.
3. Remoção de stage com entries: UI exige destino.

### Adicionar Entry
1. Via drawer do lead → "Adicionar ao Pipeline Customizado X".
2. Escolhe stage inicial.
3. Entry criada.

### Operar Kanban
- Idêntico aos pipes estruturais: drag-drop, filtros, detalhe.
- Realtime atualizações.

### Deletar
1. "Deletar pipeline".
2. Confirmação dupla.
3. Se há entries: exige migração ou cancelar todos.
4. Soft-delete; master pode hard-delete.

## Automações e Eventos

### Emite
- `CustomPipelineCreated`, `CustomPipelineUpdated`, `CustomPipelineDeleted`.
- `LeadEnteredPipe`, `LeadStageChanged`, `LeadLeftPipe` — idêntico aos estruturais, discriminado por `pipeline_id`.

### Disponível para workflows
- Todos os triggers/actions que funcionam em pipes estruturais também funcionam em customizados (`pipeline_id` é a variável).

## Casos de Uso Comuns

### 1. Pós-venda / Onboarding de Cliente
Stages: `contrato_assinado` → `setup_em_andamento` → `treinamento` → `ativo` → `churn_risk` → `churned`.
- Cliente vendido entra aqui via regra `on_stage_change(propostas → vendido)`.

### 2. Recuperação de Inadimplente
Stages: `fatura_em_atraso` → `primeira_cobranca` → `segunda_cobranca` → `negociacao` → `acordo_fechado` / `protestado`.
- Webhook do Asaas (payment_failed) cria entry.

### 3. Upsell
Stages: `oportunidade_identificada` → `proposta_upsell` → `ganho_upsell` / `rejeitado`.
- Ver [[Upsell]].

### 4. Qualificação Complexa (ciclo longo B2B)
Stages: `contato_inicial` → `pesquisa` → `discovery_call` → `technical_call` → `POC` → `proposta` → `negociacao` → `fechado`.
- Substitui ou complementa os estruturais para orgs com venda complexa.

## Edge Cases

- **Limite de pipes customizados por plano**: enforced no `create`. UI mostra upsell.
- **Stage sem is_final em pipe longo**: entry fica ativo indefinidamente; análise de SLA detecta.
- **Pipe com zero stages**: rejeitado; exige ao menos 1.
- **Membro sem permissão tenta abrir**: UI retorna 403 amigável.
- **Pipe desativado** (is_active=false): não aparece no menu; entries preservados.
- **Renomear pipe**: mudanças se propagam; workflows com referência por ID continuam funcionando (ID não muda).

## Validações

- Nome: não-vazio, único (case-insensitive) dentro da org.
- Ao menos 1 stage.
- Ordem de stages sem duplicata.
- `access_mode` válido.
- Permissões dos membros específicos válidas (devem ser membros da org).

## Métricas

- Entries ativos por stage.
- Throughput (entries finalizados por período).
- Tempo médio por stage.
- Conversão entre stages.
- Taxa de sucesso (`finished_reason=positive / total_finished`).
- Workflows associados ao pipe: taxa de execução/sucesso.

## Boas Práticas (UX)

- Admin deve nomear stages claramente (`pago` é melhor que `estado_3`).
- Cores intuitivas: verde para positivo, vermelho para negativo, azul/neutro para progresso.
- Não criar pipes com mais de 10 stages (kanban fica pesado).
- Pipes customizados ajudam a **operação** — não substituir pelos estruturais sem motivo forte.
