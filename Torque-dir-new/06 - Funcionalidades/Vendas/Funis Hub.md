---
tipo: feature
dominio: vendas
---

# Funis Hub

## Propósito

Página central que lista **todos os pipelines** (estruturais e customizados) da organização em um só lugar, com visão geral de cada um (entries ativos, conversão, tempo médio). Entry point de navegação para o admin e para quem precisa olhar múltiplos funis.

## Atores e Permissões

- **Admin**: vê todos os pipes.
- **Membros**: veem apenas pipes em que têm permissão de visualização.

Ações: `pipeline.view:{id}`, `pipeline.create_custom`.

## Dados Envolvidos

Agrega dados de:
- Lista de pipelines da org.
- Contagens por pipeline (entries ativos, em cada stage).
- Conversão derivada (win rate, velocity).
- Configurações (ativo, oculto, etc.).

## Fluxos do Usuário

### Listar
1. Menu → `Funis Hub`.
2. Lista de cards, um por pipeline. Cada card mostra:
   - Ícone e cor.
   - Nome.
   - Tipo (estrutural / custom).
   - Total de entries ativos.
   - Top 3 stages com contagem.
   - Win rate recente.
   - Botão "Abrir" → vai para kanban daquele pipe.
   - Admin: ícone de configurações.

### Criar Pipeline Customizado
- Botão "+ Novo" no hub.
- Redireciona ao wizard de criação (ver [[Pipelines Customizados]]).

### Reordenar
- Admin arrasta cards para reordenar no menu lateral.
- Ordem é persistida por organização.

### Buscar
- Campo de busca filtra pipes por nome.

### Comparar Funis
- Seleção múltipla → clique "Comparar".
- Abre view lado a lado com métricas comparativas.

## Automações e Eventos

- Hub reage a eventos `PipelineCreated`, `PipelineUpdated`, `PipelineDeleted` para atualizar em tempo real.
- Contagens agregadas atualizadas com debounce.

## Integrações

- **Pipelines estruturais**: aparecem sempre (se feature do plano permite).
- **Pipelines customizados**: aparecem conforme criados.
- **Analytics**: métricas exibidas no hub vêm de views agregadas.
- **Permissões**: filtro de pipes visíveis conforme engine.

## Edge Cases

- **Org sem pipe customizado**: mostra só os estruturais + card de CTA "Crie seu primeiro pipe customizado" para admin.
- **Org com plano básico que não tem customs**: card de upsell em vez de botão criar.
- **Pipe inativo (is_active=false)**: não aparece para membros; admin vê com badge "Inativo".

## Validações

Nenhuma validação nova — herda dos pipes individuais.

## Métricas

- Hub em si não tem métricas próprias; agrega as dos pipes.
- Visita ao hub registrada em analytics de uso (para entender frequência de navegação).

## Design

- Layout em grid responsivo.
- Dark-first.
- Cards com hover que revela mais detalhe (próximas N stages, último movimento).
- Empty state gracioso se org tem só estruturais: foco em guiá-los para usar.
