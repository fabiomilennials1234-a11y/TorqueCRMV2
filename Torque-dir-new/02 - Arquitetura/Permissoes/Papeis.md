---
tipo: identidade
---

# Papéis (Roles)

O Torque usa apenas **três papéis** no modelo de dados:

- `master`
- `admin`
- `membro`

Toda outra nomenclatura (SDR, Closer, Prospectador, SDR-sênior, Atendente) é **especialização** — não afeta modelo de permissão nativamente, apenas orienta presets e UI.

> **Invariante de código**: NUNCA criar roles adicionais (SDR/Closer como role). Eles são **conceitos de negócio** aplicados via specialty + presets de permissão.

## Master

### Identidade
- Pertence à empresa-dona do Torque (Milennials).
- Vinculação separada da tabela de membros de org cliente.
- Um master pode ter múltiplos privilégios internos (suporte, billing, tech).

### Privilégios
- Bypass de isolamento multi-tenant (com auditoria obrigatória).
- Impersonar admin de organização cliente.
- Provisionar/suspender/deletar organizações.
- Ver métricas cross-org.
- Rotacionar secrets globais.
- Gerenciar planos e features do catálogo global.

### Restrições
- MFA obrigatório.
- Impersonação requer justificativa (texto).
- Toda ação gera entrada em audit crítico, alguns geram alerta imediato.
- Master nunca é "admin de org cliente" ao mesmo tempo — contextos separados.

## Admin de Organização

### Responsabilidades
- Configurar organização: dados, branding, integrações.
- Gerenciar time: convidar, editar, remover membros.
- Configurar permissões: overrides por membro.
- Criar e manter workflows, campanhas, agentes IA.
- Ver todos os analytics.
- Aprovar comissões.
- Gerenciar plano, billing, API keys.

### Invariantes
- Toda organização precisa de **ao menos um admin ativo** a todo momento.
- Admin não pode rebaixar a si mesmo se é o último admin ativo.
- Admin não pode exceder limites do plano.

## Membro

### Identidade
- Papel padrão de usuário da organização.
- Permissões padrão baseadas em `specialty`:

### Specialty: `sdr`
- Qualifica leads, primeiro contato, agendamento.
- Acesso: Pipeline WhatsApp, Chat, Follow-ups, Templates.

### Specialty: `closer`
- Conduz reunião, proposta, fecha venda.
- Acesso: Pipeline Confirmação/Propostas, Produtos, Calendário.

### Specialty: `prospectador`
- Campanhas outbound.
- Acesso: Campanhas, Chat Outbound, Dashboard Outbound.

### Specialty: `admin`
- Igual a role=admin — uma forma de marcar que o membro atua como admin-operador.
- Na prática, role=admin é o que conta; specialty=admin é só metadado.

### Specialty: `outro`
- Permissões mínimas, customizadas caso a caso.

### Regras
- Membro vê apenas leads atribuídos a si (default) ou sem atribuição.
- Admin pode conceder ver-todos via override.
- Membro não vê Configurações, Workflow Builder, Master area, por default.

## Matriz Resumida de Permissões Default

| Feature / Ação | Master | Admin | Membro (SDR) | Membro (Closer) | Membro (Prospectador) | Membro (Outro) |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| Ver leads próprios | ✔ | ✔ | ✔ | ✔ | ✔ | ✔ |
| Ver leads de outros | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Criar lead | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Deletar lead | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Mover em Pipe WhatsApp | ✔ | ✔ | ✔ | ✔ | ✘ | ✘ |
| Mover em Pipe Confirmação | ✔ | ✔ | ✔ | ✔ | ✘ | ✘ |
| Mover em Pipe Propostas | ✔ | ✔ | ✘ | ✔ | ✘ | ✘ |
| Editar Proposta | ✔ | ✔ | ✘ | ✔ | ✘ | ✘ |
| Ver Chat | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Enviar mensagem | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Takeover de conversa | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Criar follow-up | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Ver Time | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Convidar Membro | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Gerenciar Workflow | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Gerenciar Campanha | ✔ | ✔ | ✘ | ✘ | ✔ | ✘ |
| Gerenciar Copilot | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Config da Org | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Integrações | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Analytics Próprio | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Analytics Time | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Ver Comissão Própria | ✔ | ✔ | ✔ | ✔ | ✔ | ✘ |
| Editar Regras de Comissão | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Aprovar Comissão | ✔ | ✔ | ✘ | ✘ | ✘ | ✘ |
| Master Area | ✔ | ✘ | ✘ | ✘ | ✘ | ✘ |
| Impersonar Org | ✔ | ✘ | ✘ | ✘ | ✘ | ✘ |

Todos os defaults são **overrides-capable** pelo admin via member permissions.

## Mudança de Papel

### Promoção a admin
- Outro admin ativa.
- Requer: validação de quota do plano (alguns planos limitam admins).
- Audit log crítico.
- Membro promovido recebe email de notificação.

### Rebaixamento para membro
- Outro admin rebaixa.
- Proibido se é o único admin ativo.
- Audit log.
- Email ao membro.

### Mudança de specialty (não afeta role)
- Admin muda.
- Presets associados à nova specialty **não** se aplicam automaticamente — admin escolhe aplicar (para não apagar overrides custom por engano).
- Audit log.

## Herança de Permissão

- Não há herança hierárquica (admin não "inclui" tudo de membro mais específico — é matrix).
- Master NÃO herda automaticamente admin de uma org cliente — precisa impersonar explicitamente.

## Visibilidade de Role

- UI mostra role do usuário logado no perfil.
- Admin vê role e specialty de cada membro no painel de time.
- Lead nunca vê role/specialty de quem está atendendo (privacidade).

## Master Admin vs Admin

Ponto crítico: **master e admin são mundos separados**.

- Master nunca aparece na lista de membros de uma org cliente.
- Admin de org não tem acesso a operações master.
- Master só pode virar admin de uma org específica via **impersonação explícita** (com auditoria reforçada).
