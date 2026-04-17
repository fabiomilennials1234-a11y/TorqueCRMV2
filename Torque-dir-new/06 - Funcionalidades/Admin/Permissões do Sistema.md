---
tipo: feature
dominio: admin
---

# Permissões do Sistema

## Propósito

Interface onde o admin **configura quem pode fazer o quê** dentro da organização. Inclui gestão de member overrides (conceder/negar ações específicas) e visão geral da matriz de permissões aplicada.

Ver [[03 - Identidade e Permissões/Modelo de Permissões]] para modelo completo. Este doc foca na **tela de gestão**.

## Atores e Permissões

- **Admin**: configura.
- **Membros**: veem próprias permissões (read-only).

Ações: `team.set_permissions`, `team.view_own_permissions`.

## Telas

### 1. Matriz Geral (view de admin)

Tabela: membros × features. Cada célula mostra resumo ("Admin", "Padrão SDR", "Customizado").

- Click na célula → detalhe de permissões do membro × feature.

### 2. Detalhe de Membro

Abrindo um membro, vê lista completa de ações com três estados por ação:
- **Herdado**: permissão vem de role+specialty (exibido como "✔ Default SDR" ou "✘ Default").
- **Concedido**: override explícito liberando.
- **Negado**: override explícito bloqueando.

Checkboxes três-estados (default / allow / deny).

Filtro por feature (Lead, Pipeline, Workflow...).

Busca de ação por nome.

### 3. Presets

Admin pode aplicar preset:
- Reset para default da specialty.
- "SDR Sênior" (inclui ver todos os leads, reassign).
- "Closer Júnior" (bloqueia desconto alto).
- Custom: admin salva combinação como preset reusável.

### 4. Auditoria de Permissões

Lista de mudanças recentes: quem mudou o quê, para quem, quando, motivo.

## Fluxos do Usuário

### Conceder Acesso Específico
1. Configurações → Permissões → seleciona membro.
2. Filtra "lead.view:all".
3. Muda de "Default ✘" para "✔ Concedido".
4. Opcional: motivo.
5. Salvar.
6. Membro vê mudança no próximo login (permissions_version incrementa).

### Bloquear Ação Específica
1. Similar, muda para "✘ Negado".
2. Sobrescreve default permitido.

### Reset
- Click "Reset" volta ao default da specialty do membro.

### Aplicar Preset
- Seleciona preset → preview das mudanças → confirma.

### Ver minhas permissões
- Membro acessa perfil → "Minhas Permissões".
- Lista read-only das ações permitidas/negadas.
- Útil para entender por que vê ou não vê algo.

## Regras de Negócio

1. Apenas admin altera; auditoria obrigatória.
2. Overrides persistem após mudança de specialty (diferentemente de presets).
3. Feature não disponível no plano: override não funciona (feature gating vence).
4. Mudança em permissão invalida cache de token (permissions_version).
5. Remover membro remove também seus overrides.

## Automações

### Emite
- `PermissionGranted(member_id, action, by)`, `PermissionRevoked(member_id, action, by)`.
- `PermissionsPresetApplied(member_id, preset_name, by)`.

### Reage
- Engine de permissão consulta overrides em cada decisão.

## Integrações

- **Engine de permissões**.
- **Audit log** para todas as mudanças.
- **Feature Gating**.
- **Quotas**.

## Edge Cases

- **Override em ação agora removida**: persiste no banco mas ignorado pela engine.
- **Membro downgraded** (ex.: admin → membro): overrides permanecem; se contradizem novas limitações, engine resolve.
- **Bulk edit**: admin aplica preset em múltiplos membros de uma vez (com confirmação).
- **Conflito entre overrides**: último vence (ou regra explícita de deny sempre vence — convenção).

## Validações

- Action válida (existe no catálogo).
- Membro existe e pertence à org.
- Admin tem permissão para editar.

## Métricas

- Número de overrides por org.
- Overrides mais comuns (sinal de default ruim).
- Frequência de mudança.

## Segurança

- Mudanças auditadas.
- MFA opcionalmente requerido para ações críticas (ex.: conceder `master.*` — impossível por definição; mas ações broad como `lead.view:all`).
- Admin removido: overrides por ele continuam válidos (não-repudio).

## UX

- Tabela densa mas filtrável.
- Cores claras: verde concedido, vermelho negado, cinza default.
- Dica ao hover explicando o que cada ação faz.
- Preview "o que muda" antes de aplicar preset.
