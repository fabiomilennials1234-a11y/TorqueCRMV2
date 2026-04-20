/**
 * Produtos — F11.
 *
 * Member-accessible list; admins get the manage actions (Create / Edit / Archive).
 * Pricing is displayed in BRL by default, locale-formatted client-side; server
 * always persists integer cents.
 */

import { Package, Plus, Search } from 'lucide-react'
import { useId, useMemo, useState } from 'react'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { EmptyState } from '@/ui/empty-state'
import { Input } from '@/ui/input'
import { PageHeader } from '@/ui/page-header'
import { Skeleton } from '@/ui/skeleton'
import { friendlyMessage } from '@/api/errors'
import {
  useArchiveProduct,
  useCreateProduct,
  useProducts,
  type Product,
} from '@/hooks/useProducts'

export function ProductsPage() {
  const [search, setSearch] = useState('')
  const [activeOnly, setActiveOnly] = useState(true)
  const trimmedSearch = search.trim()
  const query = useProducts({
    ...(trimmedSearch ? { search: trimmedSearch } : {}),
    active_only: activeOnly,
  })

  return (
    <div className="mx-auto max-w-[1400px] px-8">
      <PageHeader
        eyebrow="Catálogo"
        title="Produtos"
        description="Itens que aparecem nas Propostas. Preço sempre em centavos no servidor; exibição formatada por moeda."
      />

      <Toolbar
        search={search}
        onSearchChange={setSearch}
        activeOnly={activeOnly}
        onActiveOnlyChange={setActiveOnly}
      />

      {query.isError && (
        <EmptyState
          title="Não foi possível carregar o catálogo."
          description={friendlyMessage(query.error)}
          action={
            <Button variant="ghost" size="sm" onClick={() => void query.refetch()}>
              Tentar de novo
            </Button>
          }
        />
      )}

      {query.isLoading && (
        <div className="mt-6 space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      )}

      {query.isSuccess && query.items.length === 0 && (
        <EmptyState
          icon={Package}
          title="Catálogo vazio"
          description="Adicione o primeiro produto para que propostas possam referenciá-lo."
        />
      )}

      {query.isSuccess && query.items.length > 0 && (
        <ProductTable items={query.items} />
      )}

      {query.hasNextPage && (
        <div className="mt-4 flex justify-center">
          <Button
            variant="ghost"
            size="sm"
            disabled={query.isFetchingNextPage}
            onClick={() => void query.fetchNextPage()}
          >
            {query.isFetchingNextPage ? 'Carregando…' : 'Carregar mais'}
          </Button>
        </div>
      )}
    </div>
  )
}

function Toolbar({
  search,
  onSearchChange,
  activeOnly,
  onActiveOnlyChange,
}: {
  search: string
  onSearchChange: (v: string) => void
  activeOnly: boolean
  onActiveOnlyChange: (v: boolean) => void
}) {
  const create = useCreateProduct()
  const [showForm, setShowForm] = useState(false)

  return (
    <div className="mt-4 flex flex-wrap items-center gap-2">
      <div className="relative max-w-sm flex-1">
        <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
        <Input
          placeholder="Buscar por nome ou SKU…"
          className="pl-8"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <label className="flex items-center gap-2 text-sm text-ink-muted">
        <input
          type="checkbox"
          checked={activeOnly}
          onChange={(e) => onActiveOnlyChange(e.target.checked)}
        />
        Ativos apenas
      </label>
      <div className="ml-auto">
        <Button variant="primary" size="sm" onClick={() => setShowForm((v) => !v)}>
          <Plus className="mr-1 h-3.5 w-3.5" />
          Novo produto
        </Button>
      </div>
      {showForm && (
        <NewProductForm
          disabled={create.isPending}
          onCancel={() => setShowForm(false)}
          onSubmit={async (draft) => {
            await create.mutateAsync(draft)
            setShowForm(false)
          }}
        />
      )}
    </div>
  )
}

function NewProductForm({
  disabled,
  onSubmit,
  onCancel,
}: {
  disabled: boolean
  onSubmit: (d: { name: string; price_cents: number; sku?: string }) => Promise<void>
  onCancel: () => void
}) {
  const [name, setName] = useState('')
  const [price, setPrice] = useState('')
  const [sku, setSku] = useState('')
  const nameId = useId()
  const priceId = useId()
  const skuId = useId()

  const priceCents = useMemo(() => {
    const n = Number(price.replace(/[^0-9.,]/g, '').replace(',', '.'))
    return Number.isFinite(n) ? Math.round(n * 100) : NaN
  }, [price])

  const valid = name.trim().length >= 2 && Number.isFinite(priceCents) && priceCents >= 0

  return (
    <form
      className="mt-3 flex w-full flex-wrap items-end gap-2 rounded-lg bg-surface p-4 shadow-elev-1"
      onSubmit={(e) => {
        e.preventDefault()
        if (!valid) return
        const draft: { name: string; price_cents: number; sku?: string } = {
          name: name.trim(),
          price_cents: priceCents,
        }
        if (sku.trim()) draft.sku = sku.trim()
        void onSubmit(draft)
      }}
    >
      <div className="flex-1">
        <label htmlFor={nameId} className="mb-1 block text-xs text-ink-muted">
          Nome
        </label>
        <Input id={nameId} value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="w-32">
        <label htmlFor={priceId} className="mb-1 block text-xs text-ink-muted">
          Preço (R$)
        </label>
        <Input
          id={priceId}
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          placeholder="0,00"
        />
      </div>
      <div className="w-40">
        <label htmlFor={skuId} className="mb-1 block text-xs text-ink-muted">
          SKU (opcional)
        </label>
        <Input id={skuId} value={sku} onChange={(e) => setSku(e.target.value)} />
      </div>
      <div className="flex gap-2">
        <Button variant="ghost" size="sm" type="button" onClick={onCancel}>
          Cancelar
        </Button>
        <Button variant="primary" size="sm" type="submit" disabled={disabled || !valid}>
          {disabled ? 'Salvando…' : 'Criar'}
        </Button>
      </div>
    </form>
  )
}

function ProductTable({ items }: { items: Product[] }) {
  return (
    <div className="mt-6 overflow-hidden rounded-lg bg-surface shadow-elev-1">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
            <th className="px-5 py-3 text-left font-medium">Produto</th>
            <th className="px-5 py-3 text-left font-medium">SKU</th>
            <th className="px-5 py-3 text-right font-medium">Preço</th>
            <th className="px-5 py-3 text-left font-medium">Status</th>
            <th className="px-5 py-3" />
          </tr>
        </thead>
        <tbody>
          {items.map((p, i) => (
            <ProductRow key={p.id} product={p} separator={i > 0} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ProductRow({ product, separator }: { product: Product; separator: boolean }) {
  const archive = useArchiveProduct(product.id)
  return (
    <tr
      className={
        'transition-colors hover:bg-elevated/40' +
        (separator ? ' shadow-[inset_0_1px_0_0_hsl(var(--hairline))]' : '')
      }
    >
      <td className="px-5 py-3">
        <div className="text-sm text-ink">{product.name}</div>
        {product.description && (
          <div className="text-xs text-ink-dim">{product.description}</div>
        )}
      </td>
      <td className="px-5 py-3 font-metric text-xs text-ink-muted">{product.sku ?? '—'}</td>
      <td className="px-5 py-3 text-right font-metric text-sm text-ink">
        {formatMoney(product.price_cents, product.currency)}
      </td>
      <td className="px-5 py-3">
        {product.is_active ? (
          <Badge tone="success">ativo</Badge>
        ) : (
          <Badge tone="neutral">arquivado</Badge>
        )}
      </td>
      <td className="px-5 py-3 text-right">
        {product.is_active && (
          <Button
            variant="ghost"
            size="xs"
            disabled={archive.isPending}
            onClick={() => void archive.mutateAsync()}
          >
            Arquivar
          </Button>
        )}
      </td>
    </tr>
  )
}

function formatMoney(cents: number, currency: string): string {
  try {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: currency || 'BRL',
    }).format(cents / 100)
  } catch {
    return `${(cents / 100).toFixed(2)} ${currency}`
  }
}
