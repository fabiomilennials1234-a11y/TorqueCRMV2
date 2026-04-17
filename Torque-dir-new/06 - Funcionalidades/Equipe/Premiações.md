---
tipo: feature
dominio: equipe
---

# Premiações

## Propósito

Sistema de gamificação / recompensa que **associa conquistas a recompensas** concretas (bonus, troféu virtual, visibilidade). Alimenta motivação do time. Vinculado a metas e marcos.

## Atores e Permissões

- **Admin**: define catálogo, distribui.
- **Membros**: visualizam catálogo, resgatam (quando aplicável).

Ações: `reward.view`, `reward.edit_rules`.

## Dados Envolvidos

### Reward (catálogo)
- `id`, `organization_id`.
- `name`.
- `description`.
- `type`: `badge` (só virtual) | `bonus` (valor em moeda) | `item` (produto físico / voucher) | `recognition` (publicação no mural).
- `value` (se bonus).
- `icon_url`.
- `trigger_type`: `goal_achieved` | `rank_top_x` | `streak` | `manual`.
- `trigger_config`: JSON conforme type.
- `scarcity`: `unlimited` | `limited_count` (N por período) | `first_n_only`.
- `is_active`.

### Reward Grant (quando dado)
- `id`, `organization_id`.
- `reward_id`.
- `member_id`.
- `granted_at`.
- `granted_by`: `system` | member_id (admin manual).
- `context`: JSON (qual meta, qual ranking, etc.).
- `status`: `granted` | `acknowledged` (member viu) | `redeemed` (usou).

## Regras de Negócio

1. Reward manual: admin pode conceder a qualquer member em qualquer momento.
2. Reward automático: dispara em evento específico.
3. Scarcity: `limited_count` → primeiros N do período o recebem, restantes não.
4. Bonus: valor é adicionado ao commission ou registrado separadamente para folha.
5. Badge: apenas visual; aparece em perfil do membro.
6. Reconhecimento: publicação em mural/dashboard da org ("João atingiu 150% da meta!").

## Fluxos do Usuário

### Criar Reward (admin)
1. `Premiações → Catálogo → Novo`.
2. Form: tipo, nome, descrição, trigger, scarcity.
3. Preview.
4. Ativar.

### Distribuir Manualmente
1. Admin vê member → botão "Dar reward".
2. Escolhe item do catálogo.
3. Opcional: mensagem personalizada.
4. Grant criado.
5. Membro notificado.

### Ver "Minhas Premiações"
1. Membro acessa perfil → aba Premiações.
2. Lista de badges/bonus ganhos.
3. Cada com contexto ("Atingiu meta X em Março 2026").

### Mural de Reconhecimento
- Dashboard pode ter widget "Conquistas da semana" mostrando rewards públicas.
- Engajamento coletivo.

## Automações e Eventos

### Emite
- `RewardGranted(member_id, reward_id, context)`.

### Reage
- `GoalAchieved` → avalia rewards com trigger `goal_achieved`.
- Cron fim-de-mês → avalia `rank_top_x` (top 3 em métrica, por exemplo).
- `StreakExtended` (ex.: 5 dias seguidos batendo follow-ups) → trigger `streak`.

## Integrações

- **Metas**: trigger primário.
- **Comissões**: rewards tipo `bonus` podem alimentar folha.
- **Analytics**: ranking alimenta rewards.
- **Notificações**: push ao ganhar.
- **Mural / Dashboard**: exibição pública.

## Edge Cases

- **Empate em ranking**: regra de desempate (ex.: quem atingiu primeiro).
- **Reward com scarcity esgotada**: novos members não recebem; sistema avisa.
- **Member removido com rewards pendentes**: bonus pode ser pago ainda (conforme contrato); badges ficam em perfil inativo.
- **Alterar reward depois de grants**: grants existentes preservam config antiga.
- **Reward duplicado** (membro recebe o mesmo 2x): por config — alguns são 1-per-lifetime, outros repetíveis.

## Validações

- Nome único no catálogo.
- Value ≥ 0.
- Trigger config coerente.

## Métricas

- Rewards concedidos por período.
- Top membros por número de rewards.
- Correlação reward × engajamento / produtividade.
- Custo total de bonus.
- Rewards mais/menos concedidos.

## UX

- Animação ao ganhar (confete, som, destaque).
- Badge visual na foto do perfil.
- Lista ordenada por valor/raridade.
- Catálogo visível para motivação (membros veem o que podem ganhar).
