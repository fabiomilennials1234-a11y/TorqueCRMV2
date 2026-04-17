---
tipo: requisitos
---

# Decisões de Design

Decisões arquiteturais e de produto deliberadas, com justificativa. Documenta "por que assim". Serve de referência a quem reimplementa.

## D-01: Roles fixos (admin/master/membro)

**Decisão**: apenas três roles no sistema de permissão; SDR/Closer são `specialty`.

**Por quê**: adicionar roles toda vez que a UX pede um novo papel funcional leva a proliferação caótica. Separar autoridade (role) de função (specialty) permite evolução da UX sem mexer no core de segurança.

**Trade-off**: admin precisa configurar overrides para casos atípicos. Aceitável pela clareza ganha.

## D-02: Multi-tenant em Row-Level Security

**Decisão**: isolamento no banco de dados via políticas (RLS ou equivalente), não apenas na aplicação.

**Por quê**: bug em aplicação pode vazar dados entre tenants. RLS como último bastião é irreplaceable.

**Trade-off**: complexidade extra no banco. Vale a segurança.

## D-03: Pipeline Entries separadas, Lead único

**Decisão**: Lead é entidade única; presença em pipelines é via tabela separada (Pipeline Entry).

**Por quê**: Lead pode estar em múltiplos pipes simultaneamente (qualificação + campanha + upsell). Status do lead é derivado, não fixo.

**Trade-off**: queries mais complexas. Ganho em flexibilidade justifica.

## D-04: Workflows como DAG visual

**Decisão**: automação em grafo acíclico dirigido, editor visual, sem código.

**Por quê**: admin deve configurar sem devs. DAG é expressivo o suficiente para 90% dos casos e inteligível visualmente.

**Trade-off**: alguns casos complexos ficam sem suporte; são resolvidos com `call_webhook` para código externo.

## D-05: Agentes IA com Wizard e System Prompt Gerado

**Decisão**: admin preenche wizard; system prompt gerado automaticamente.

**Por quê**: escrever system prompt à mão é arte + risco de erro. Wizard gera prompt estruturado e coerente.

**Trade-off**: menos flexibilidade. Custom agents têm mais liberdade, mas com orientação forte do wizard.

## D-06: Batch Window em Agente IA (8s default)

**Decisão**: agente não responde a cada mensagem; agrupa em janela curta.

**Por quê**: humanização. WhatsApp tem cultura de múltiplas mensagens rápidas. Responder a cada é robotic.

**Trade-off**: latência percebida um pouco maior. Aceitável.

## D-07: Copilot pausa em Takeover Humano

**Decisão**: humano responder pausa agente por 10 min.

**Por quê**: previne colisão. Humano e agente respondendo simultaneamente é desastroso.

**Trade-off**: janela fixa pode ser curta/longa demais. Configurável por agente.

## D-08: Tags case-insensitive

**Decisão**: tags normalizadas para lowercase trim ao comparar.

**Por quê**: "Ouro" e "ouro" e " ouro " são a mesma coisa para o usuário. Evitar duplicação.

**Trade-off**: apresentação preserva o original. Query normaliza.

## D-09: Webhook público tem contrato estável

**Decisão**: `/webhooks/leads` versionado, backward-compatible dentro da versão.

**Por quê**: clientes externos dependem (n8n, Zapier). Quebrar contrato quebra a operação deles.

**Trade-off**: dificulta evolução. Compensado por `v2` paralela quando breaking.

## D-10: Idempotência via external_id + dedupe por contato

**Decisão**: ingestão usa `external_id` preferencialmente; fallback dedupe por phone/email.

**Por quê**: retries e duplicação são comuns em integrações. Sem idempotência, lead duplicado é ruído garantido.

**Trade-off**: dedupe por contato pode falhar se lead trocou telefone. Admin gerencia merge manual em casos raros.

## D-11: Soft-delete preferencial

**Decisão**: entidades importantes são soft-deleted.

**Por quê**: preserva histórico; erros de deleção reversíveis; análises retroativas possíveis.

**Trade-off**: banco cresce. Arquivamento + retenção mitiga.

## D-12: Append-only Audit

**Decisão**: audit log nunca atualizado ou deletado.

**Por quê**: integridade forense. Valor probatório. Compliance LGPD/legal.

**Trade-off**: volume. Partição + retenção controlam.

## D-13: Quota Enforcement no Backend

**Decisão**: quotas são hard-limit no backend, não só soft-limit na UI.

**Por quê**: UI pode ter bug. Cliente mal-intencionado pode bypass UI. Backend é última defesa.

**Trade-off**: nenhum real; é obrigatório.

## D-14: Realtime via Subscription Filtrada

**Decisão**: cliente subscribe a tópicos escopados por organization_id; servidor valida.

**Por quê**: evita servidor enviar tudo e cliente filtrar (vaza). Cliente só vê o que é dele.

**Trade-off**: complexidade em implementação. Aceitável.

## D-15: Single Language de Domínio (inglês no código, português na UI)

**Decisão**: nomes de entidades, colunas, APIs em inglês. UI/labels em português-BR.

**Por quê**: inglês é padrão técnico internacional. Português é audience do Torque.

**Trade-off**: traduções paralelas; overhead baixo.

## D-16: Agente IA tem Biblioteca de Áudios Pré-gerados

**Decisão**: áudios comuns pré-gerados; TTS só para dinâmico.

**Por quê**: economia de latência + custo. Primeira resposta rápida.

**Trade-off**: manutenção de biblioteca. Vale para comum/estático.

## D-17: Campanhas Paralelas aos Pipes

**Decisão**: campanha é processo próprio; lead pode estar em campanha E pipe.

**Por quê**: pipeline modela processo orgânico; campanha é esforço pontual. São coisas diferentes.

**Trade-off**: conceito extra para o admin aprender. Compensado pela flexibilidade.

## D-18: TypeScript generated types do banco

**Decisão** (implementação atual, não prescritivo à reimplementação): tipos TypeScript gerados do schema do banco via ferramenta.

**Por quê**: DRY; garante coerência entre schema e código; evita drift.

**Nota**: reimplementação em outra stack tem seu análogo (sqlc em Go, sqlalchemy em Python, Ecto em Elixir, etc.).

## D-19: Dark-first UI

**Decisão**: UI começa com tema dark por padrão.

**Por quê**: é o que os usuários do ICP preferem (desenvolvedores, comerciais experientes). Reduz fadiga visual em uso prolongado.

**Trade-off**: light mode funcional mas secundário.

## D-20: Integration Adapters

**Decisão**: cada integração tem adaptador estável; troca de provider = troca de adaptador.

**Por quê**: providers mudam ou saem do ar. Não acoplar feature a provider.

**Trade-off**: overhead inicial para escrever adaptador. Vale a longo prazo.

## D-21: Event Sourcing Parcial

**Decisão**: entidades têm estado canônico + log de eventos paralelo.

**Por quê**: queries rápidas via estado; audit via eventos; rebuild se necessário.

**Trade-off**: dualidade: garantir consistência entre estado e eventos.

## D-22: Multi-channel Desde o Start

**Decisão**: código não assume "WhatsApp"; abstração de canal desde início.

**Por quê**: cliente precisa de flexibilidade. Adicionar Messenger/SMS/Email depois é mais barato se já abstraído.

**Trade-off**: abstração inicial. Vale — já provou em 3 canais ativos.

## D-23: Oráculo Comercial Read-only

**Decisão**: Oráculo só responde, nunca executa ações.

**Por quê**: risco. LLM gerando query que atualiza dados = catástrofe potencial.

**Trade-off**: menos "agente autônomo". Ganho em segurança > perda em utilidade.

## D-24: Sem Multi-language no Core (ainda)

**Decisão**: português-BR como único idioma em v1.

**Por quê**: mercado foco é Brasil. Multi-language adiciona complexidade sem valor imediato.

**Trade-off**: internacionalização futura vai exigir refactor (chaves de tradução, formatos regionais).

## D-25: Worker escalável horizontalmente

**Decisão**: workers sem estado local; escala por adicionar instâncias.

**Por quê**: volume varia drasticamente (campanhas grandes). Escala elástica é necessária.

**Trade-off**: lock em fila obrigatório. Padrão conhecido.

## Meta-decisão: Documentar decisões

**Decisão**: toda decisão arquitetural relevante entra em ADR (Architecture Decision Record) em doc permanente.

**Por quê**: equipe muda; decisões se perdem. ADR preserva contexto para futuros engenheiros.

**Trade-off**: tempo de documentar. Vale 10x.
