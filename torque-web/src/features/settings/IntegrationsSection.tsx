/**
 * S49 / F.1 + S50 / F.2 — Integrations section.
 *
 * Rich UI surface for the four wired providers (Google, TinyERP, Meta,
 * SZ.Chat) + the two "built-in" rows (WhatsApp Evolution, Asaas,
 * n8n) that Torque carries as ambient infrastructure. Every action
 * goes through a Sheet-based flow so the layout stays stable while the
 * user is connecting — no disruptive redirects except the Google OAuth
 * full-page handoff (which is required by Google's redirect_uri
 * contract).
 */

import { useId, useState } from 'react'
import { AlertCircle, Check, Loader2, RefreshCw } from 'lucide-react'

import { Badge } from '@/ui/badge'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Sheet, SheetContent, SheetTrigger } from '@/ui/sheet'
import {
  googleConnectURL,
  useConnectMeta,
  useConnectSZChat,
  useConnectTinyERP,
  useDisconnectGoogle,
  useDisconnectMeta,
  useDisconnectSZChat,
  useDisconnectTinyERP,
  useIntegrations,
  useSyncTinyERPProducts,
  type IntegrationCredential,
} from '@/hooks/useIntegrations'
import { cn } from '@/lib/utils'

export function IntegrationsSection() {
  const { data: credentials, isLoading } = useIntegrations()
  const byProvider = (p: string): IntegrationCredential | undefined =>
    credentials?.find((c) => c.provider === p)

  if (isLoading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2">
        {[0, 1, 2, 3].map((k) => (
          <div key={k} className="h-28 animate-pulse rounded-lg bg-elevated/40 shadow-elev-1" />
        ))}
      </div>
    )
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <GoogleCard cred={byProvider('google')} />
      <TinyERPCard cred={byProvider('tinyerp')} />
      <MetaCard cred={byProvider('meta')} />
      <SZChatCard cred={byProvider('szchat')} />
      {/* Ambient rows — managed elsewhere. Kept for operator context. */}
      <AmbientCard
        logo="WA"
        name="WhatsApp · Evolution API"
        desc="Canal conversacional principal (configurado por canal)"
      />
      <AmbientCard logo="A" name="Asaas" desc="Cobrança e assinaturas (F14)" />
      <AmbientCard logo="n8" name="n8n (auto-hospedado)" desc="Orquestrador externo" />
    </div>
  )
}

// ---------------- Google ---------------------------------------------

function GoogleCard({ cred }: { cred: IntegrationCredential | undefined }) {
  const disconnect = useDisconnectGoogle()
  return (
    <ProviderCard
      logo="G"
      name="Google Calendar"
      desc="Agenda de reuniões sincronizadas (OAuth 2.0)"
      cred={cred}
      connectAction={
        <Button
          variant="ghost"
          size="xs"
          onClick={() => {
            window.location.href = googleConnectURL()
          }}
        >
          Conectar
        </Button>
      }
      disconnectAction={
        <Button
          variant="ghost"
          size="xs"
          disabled={disconnect.isPending}
          onClick={() => disconnect.mutate(undefined)}
        >
          Desconectar
        </Button>
      }
    />
  )
}

// ---------------- TinyERP --------------------------------------------

function TinyERPCard({ cred }: { cred: IntegrationCredential | undefined }) {
  const disconnect = useDisconnectTinyERP()
  const sync = useSyncTinyERPProducts()
  const [syncResult, setSyncResult] = useState<string | null>(null)

  return (
    <ProviderCard
      logo="T"
      name="TinyERP"
      desc="Catálogo de produtos + pedidos"
      cred={cred}
      note={syncResult ?? undefined}
      connectAction={<TinyERPConnectButton />}
      disconnectAction={
        <>
          <Button
            variant="ghost"
            size="xs"
            disabled={sync.isPending}
            onClick={async () => {
              setSyncResult(null)
              try {
                const r = await sync.mutateAsync()
                setSyncResult(
                  `${r.inserted} novo(s) · ${r.updated} atualizado(s) · ${r.skipped} ignorado(s)`
                )
              } catch {
                setSyncResult('Falha ao sincronizar — ver detalhe do erro')
              }
            }}
          >
            {sync.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <RefreshCw className="h-3 w-3" />
            )}
            Sincronizar
          </Button>
          <Button
            variant="ghost"
            size="xs"
            disabled={disconnect.isPending}
            onClick={() => disconnect.mutate(undefined)}
          >
            Desconectar
          </Button>
        </>
      }
    />
  )
}

function TinyERPConnectButton() {
  const [open, setOpen] = useState(false)
  const [key, setKey] = useState('')
  const connect = useConnectTinyERP()
  const [err, setErr] = useState<string | null>(null)
  const keyId = useId()

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="xs">
          Conectar
        </Button>
      </SheetTrigger>
      <SheetContent
        title="Conectar TinyERP"
        description="Cole a API key do tenant. A chave fica criptografada no banco (AES-256-GCM) e nunca entra no frontend."
      >
        <form
          className="flex flex-col gap-4 p-6"
          onSubmit={async (e) => {
            e.preventDefault()
            setErr(null)
            try {
              await connect.mutateAsync(key)
              setOpen(false)
              setKey('')
            } catch (e: unknown) {
              setErr(String((e as Error)?.message ?? 'Falha ao conectar'))
            }
          }}
        >
          <label htmlFor={keyId} className="flex flex-col gap-1 text-sm text-ink-muted">
            API key TinyERP
            <Input
              id={keyId}
              type="password"
              autoComplete="off"
              spellCheck={false}
              value={key}
              onChange={(e) => setKey(e.target.value)}
              placeholder="32+ caracteres"
              required
              minLength={10}
            />
          </label>
          {err && <p className="text-xs text-danger">{err}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="submit" disabled={connect.isPending || key.length < 10}>
              {connect.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
              Conectar
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ---------------- Meta Ads -------------------------------------------

function MetaCard({ cred }: { cred: IntegrationCredential | undefined }) {
  const disconnect = useDisconnectMeta()
  return (
    <ProviderCard
      logo="M"
      name="Meta Ads"
      desc="Ingestão de leads + insights de campanha (Graph API)"
      cred={cred}
      connectAction={<MetaConnectButton />}
      disconnectAction={
        <Button
          variant="ghost"
          size="xs"
          disabled={disconnect.isPending}
          onClick={() => disconnect.mutate(undefined)}
        >
          Desconectar
        </Button>
      }
    />
  )
}

function MetaConnectButton() {
  const [open, setOpen] = useState(false)
  const [token, setToken] = useState('')
  const [acct, setAcct] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const connect = useConnectMeta()
  const tokenId = useId()
  const acctId = useId()

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="xs">
          Conectar
        </Button>
      </SheetTrigger>
      <SheetContent
        title="Conectar Meta Ads"
        description="Cole o token de System User (long-lived). O ID da conta (act_…) pode ficar em branco — nesse caso o backend usa o ID armazenado na credencial."
      >
        <form
          className="flex flex-col gap-4 p-6"
          onSubmit={async (e) => {
            e.preventDefault()
            setErr(null)
            try {
              const input: Parameters<typeof connect.mutateAsync>[0] = {
                access_token: token,
              }
              const trimmedAcct = acct.trim()
              if (trimmedAcct) input.account_id = trimmedAcct
              await connect.mutateAsync(input)
              setOpen(false)
              setToken('')
              setAcct('')
            } catch (e: unknown) {
              setErr(String((e as Error)?.message ?? 'Falha ao conectar'))
            }
          }}
        >
          <label htmlFor={tokenId} className="flex flex-col gap-1 text-sm text-ink-muted">
            Access token
            <Input
              id={tokenId}
              type="password"
              autoComplete="off"
              spellCheck={false}
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="EAAB..."
              required
              minLength={20}
            />
          </label>
          <label htmlFor={acctId} className="flex flex-col gap-1 text-sm text-ink-muted">
            Account ID (opcional)
            <Input
              id={acctId}
              value={acct}
              onChange={(e) => setAcct(e.target.value)}
              placeholder="act_1234567890"
            />
          </label>
          {err && <p className="text-xs text-danger">{err}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="submit" disabled={connect.isPending || token.length < 20}>
              {connect.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
              Conectar
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ---------------- SZ.Chat --------------------------------------------

function SZChatCard({ cred }: { cred: IntegrationCredential | undefined }) {
  const disconnect = useDisconnectSZChat()
  return (
    <ProviderCard
      logo="SZ"
      name="SZ.Chat"
      desc="Gateway de mensageria alternativo ao Evolution API"
      cred={cred}
      connectAction={<SZChatConnectButton />}
      disconnectAction={
        <Button
          variant="ghost"
          size="xs"
          disabled={disconnect.isPending}
          onClick={() => disconnect.mutate(undefined)}
        >
          Desconectar
        </Button>
      }
    />
  )
}

function SZChatConnectButton() {
  const [open, setOpen] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [channel, setChannel] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const connect = useConnectSZChat()
  const apiKeyId = useId()
  const channelId = useId()

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="xs">
          Conectar
        </Button>
      </SheetTrigger>
      <SheetContent
        title="Conectar SZ.Chat"
        description="A API key é criptografada no banco. O channel ID é opcional — se ausente, o backend usa o que vier na cred (se armazenado)."
      >
        <form
          className="flex flex-col gap-4 p-6"
          onSubmit={async (e) => {
            e.preventDefault()
            setErr(null)
            try {
              const input: Parameters<typeof connect.mutateAsync>[0] = {
                api_key: apiKey,
              }
              const trimmedChannel = channel.trim()
              if (trimmedChannel) input.channel_id = trimmedChannel
              await connect.mutateAsync(input)
              setOpen(false)
              setApiKey('')
              setChannel('')
            } catch (e: unknown) {
              setErr(String((e as Error)?.message ?? 'Falha ao conectar'))
            }
          }}
        >
          <label htmlFor={apiKeyId} className="flex flex-col gap-1 text-sm text-ink-muted">
            API key
            <Input
              id={apiKeyId}
              type="password"
              autoComplete="off"
              spellCheck={false}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              required
              minLength={16}
            />
          </label>
          <label htmlFor={channelId} className="flex flex-col gap-1 text-sm text-ink-muted">
            Channel ID (opcional)
            <Input
              id={channelId}
              value={channel}
              onChange={(e) => setChannel(e.target.value)}
              placeholder="p. ex. wa_main"
            />
          </label>
          {err && <p className="text-xs text-danger">{err}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="submit" disabled={connect.isPending || apiKey.length < 16}>
              {connect.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
              Conectar
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  )
}

// ---------------- Shared card + helpers ------------------------------

interface ProviderCardProps {
  logo: string
  name: string
  desc: string
  cred: IntegrationCredential | undefined
  note?: string | undefined
  connectAction: React.ReactNode
  disconnectAction: React.ReactNode
}

function ProviderCard({
  logo,
  name,
  desc,
  cred,
  note,
  connectAction,
  disconnectAction,
}: ProviderCardProps) {
  const connected = cred?.connected ?? false
  const lastErr = cred?.last_error_text
  return (
    <div className="flex items-start gap-3 rounded-lg bg-surface p-4 shadow-elev-1 transition-shadow hover:shadow-elev-2">
      <div className="font-metric flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-elevated text-xs text-ink-muted shadow-hairline">
        {logo}
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium text-ink">{name}</div>
        <div className="text-xs text-ink-muted">{desc}</div>
        {cred?.external_account_id && (
          <div className="text-ink-subtle mt-1 truncate font-mono text-[11px]">
            {cred.external_account_id}
          </div>
        )}
        <div className="mt-2 flex flex-wrap items-center gap-2">
          {connected ? (
            <Badge tone="success">
              <Check className="h-2.5 w-2.5" />
              conectado
            </Badge>
          ) : (
            <Badge tone="neutral">desconectado</Badge>
          )}
          {connected && cred?.last_success_at && (
            <span className="text-ink-subtle text-[11px]">
              sincronizado {relativeTime(cred.last_success_at)}
            </span>
          )}
          {connected ? disconnectAction : connectAction}
        </div>
        {lastErr && cred?.last_error_at && (
          <div
            className={cn(
              'mt-2 flex items-start gap-1.5 rounded-sm bg-warning/5 px-2 py-1.5 text-[11px]',
              'shadow-[inset_0_0_0_1px_hsl(var(--warning)/0.25)]'
            )}
          >
            <AlertCircle className="mt-0.5 h-3 w-3 shrink-0 text-warning" />
            <div className="min-w-0 flex-1">
              <div className="truncate font-medium text-warning">{lastErr}</div>
              <div className="text-ink-subtle">{relativeTime(cred.last_error_at)}</div>
            </div>
          </div>
        )}
        {note && <div className="mt-2 text-[11px] text-ink-muted">{note}</div>}
      </div>
    </div>
  )
}

function AmbientCard({ logo, name, desc }: { logo: string; name: string; desc: string }) {
  return (
    <div className="flex items-start gap-3 rounded-lg bg-surface p-4 shadow-elev-1">
      <div className="font-metric flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-elevated text-xs text-ink-muted shadow-hairline">
        {logo}
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium text-ink">{name}</div>
        <div className="text-xs text-ink-muted">{desc}</div>
        <div className="mt-2">
          <Badge tone="neutral">gerenciado</Badge>
        </div>
      </div>
    </div>
  )
}

/**
 * relativeTime converts an ISO timestamp into PT-BR relative phrasing.
 * Deliberately lightweight — react-intl's FormattedRelativeTime pulls a
 * larger polyfill than the 3-bucket heuristic below needs.
 */
export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime()
  const now = Date.now()
  if (Number.isNaN(then)) return ''
  const diffSec = Math.round((now - then) / 1000)
  if (diffSec < 60) return 'agora'
  const diffMin = Math.round(diffSec / 60)
  if (diffMin < 60) return `há ${diffMin} min`
  const diffHr = Math.round(diffMin / 60)
  if (diffHr < 24) return `há ${diffHr} h`
  const diffDay = Math.round(diffHr / 24)
  if (diffDay < 30) return `há ${diffDay} d`
  return new Date(iso).toLocaleDateString('pt-BR')
}
