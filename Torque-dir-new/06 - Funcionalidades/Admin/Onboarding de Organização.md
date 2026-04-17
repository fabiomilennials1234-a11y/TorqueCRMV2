---
tipo: feature
dominio: admin
---

# Onboarding de Organização

## Propósito

Wizard guiado para o **primeiro admin** configurar a organização após provisionamento. Objetivo: levar a org ao "primeiro valor" em < 15 minutos — ou seja, ter um canal conectado, um pipe com leads, e um fluxo mínimo para receber.

## Atores e Permissões

- **Admin novo** (recém-provisionado).
- **Master**: pode reset/retomar wizard.

Ações: relativas à configuração (organization.edit, integration.connect, etc.).

## Estrutura do Wizard

### Etapa 1 — Boas-vindas
- Vídeo curto ou ilustração.
- Explica o que vamos configurar.
- CTA: "Começar".

### Etapa 2 — Dados da Empresa
- Nome (se não veio do provisionamento).
- Logo.
- Cor de destaque.
- Fuso horário.
- Janela de negócio.

### Etapa 3 — Primeiro Canal
- Escolhe entre canais disponíveis (WhatsApp, Messenger).
- Para WhatsApp: fluxo de conexão (QR code tipicamente).
- Valida conexão.
- Dica: "Pule se ainda não tem número, pode configurar depois".

### Etapa 4 — Primeiro Pipe
- Pipe WhatsApp (estrutural) vem com stages default.
- Admin pode: aceitar defaults, ou editar stages/nomes.
- Pequeno tutorial de drag-drop no kanban vazio.

### Etapa 5 — Convide seu Time
- Form para convidar 2-3 membros com specialty.
- "Pular" disponível.

### Etapa 6 — Primeiro Workflow (opcional)
- Template pronto: "Mensagem de boas-vindas ao novo lead".
- Admin revisa, pode editar, ativa.

### Etapa 7 — Agente IA (opcional)
- Template pronto: "Qualificador básico".
- Admin preenche business_context mínimo.
- Ativa em playground (não produção ainda).

### Etapa 8 — Primeiro Lead
- Opções:
  - "Adicionar manualmente" — form simples.
  - "Conectar integração de entrada" — instruções de webhook.
  - "Importar CSV" — upload.

### Etapa 9 — Tour Guiado
- Walkthrough da UI: onde estão as features principais.
- Tooltips dinâmicas.

### Etapa 10 — Conclusão
- Checklist de sucesso (✔ canal conectado, ✔ time convidado, ✔ primeiro lead).
- Próximos passos sugeridos.
- Marca `onboarded_at`.

## Regras de Negócio

1. Wizard persiste progresso em `onboarding_state` da org.
2. Admin pode sair e voltar — retoma de onde parou.
3. Etapas opcionais podem ser puladas.
4. Após `onboarded_at`, wizard não aparece mais — mas admin pode acessar por menu → "Setup".
5. Master pode resetar onboarding (ex.: cliente quer refazer tour).

## Fluxos do Usuário

Ver estrutura acima.

## Automações

### Emite
- `OnboardingStepCompleted(step)`, `OnboardingCompleted`.

### Reage
- Sistema sugere próximos passos em dashboard baseado em `onboarding_state`.

## Integrações

- Todas as features configuráveis no wizard.
- Tutorial em-app usa sistema de tour (componentes destacando UI).

## Edge Cases

- **Admin que já fez onboarding recebe trainee**: trainee tem próprio tutorial mais curto.
- **Org provisionada cold** (sem usuário ainda): onboarding começa no primeiro login.
- **Master impersonando**: vê onboarding do cliente mas não o completa pelo cliente.
- **Erro em etapa** (ex.: não conseguiu conectar canal): pode pular, configurar depois.

## Validações

- Campos obrigatórios por etapa.
- Canal conectado validado antes de prosseguir.
- Convites válidos.

## Métricas

- Taxa de completude (quantos termimam vs abandonam).
- Taxa de abandono por etapa (identifica fricção).
- Tempo médio até conclusão.
- Correlação onboarding completo × retenção (clientes que completam ficam mais).

## Segurança

- Mesmos padrões — validação, auth, audit.

## UX

- Progresso visível (passo N de M).
- Linguagem amigável.
- Skip disponíveis.
- Persistência óbvia (não perde dado ao sair).
- Celebra conclusão (confete, "Parabéns, sua operação está no ar!").
