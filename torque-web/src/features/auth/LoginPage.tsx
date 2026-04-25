import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Lock, Mail } from 'lucide-react'
import { friendlyMessage } from '@/api/errors'
import { Button } from '@/ui/button'
import { Input } from '@/ui/input'
import { Kbd } from '@/ui/kbd'
import { useLogin } from '@/hooks/useLogin'
import { useAuth } from '@/providers/AuthProvider'
import { useTheme } from '@/providers/ThemeProvider'
import { useUiMode } from '@/providers/UiModeProvider'

import torqueIcon from '@/assets/torque-icon.png'
import torqueLogoDark from '@/assets/torque-logo.png'
import torqueLogoLight from '@/assets/torque-logo-dark.png'
import torqueHexagons from '@/assets/torque-hexagons.png'

export function LoginPage() {
  const nav = useNavigate()
  const { theme } = useTheme()
  const { mode } = useUiMode()
  const { refresh } = useAuth()
  const login = useLogin()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const errorMessage = login.error ? friendlyMessage(login.error) : null

  return (
    <div className="bg-bg text-ink relative flex min-h-screen">
      {/* Left — form */}
      <div className="relative flex flex-1 flex-col justify-between px-10 py-10 lg:px-16">
        <div className="flex items-center gap-2.5">
          <img
            src={theme === 'dark' ? torqueLogoDark : torqueLogoLight}
            alt="Torque"
            className="h-7 w-auto"
          />
        </div>

        <form
          onSubmit={async (e) => {
            e.preventDefault()
            if (login.isPending) return
            try {
              await login.mutateAsync({ email, password })
              // Wait for the authoritative /auth/me refresh so the shell reads
              // real permissions + ui_mode before we navigate.
              const session = await refresh()
              const effectiveMode = session ? mode : mode
              const target = effectiveMode === 'salesperson' ? '/cockpit' : '/'
              void nav(target, { replace: true })
            } catch {
              // Error is surfaced inline via login.error (silent: true in the hook).
            }
          }}
          className="mx-auto w-full max-w-[380px]"
        >
          <div className="mb-10">
            <div className="mb-4 flex justify-center">
              <img src={torqueIcon} alt="" className="h-14 w-14" />
            </div>
            <h1 className="font-display tracking-tightest text-center text-[2.4rem] leading-[1.02]">
              Bem-vindo de volta.
            </h1>
            <p className="text-ink-muted mt-3 text-center text-[0.9375rem] leading-relaxed">
              Entre para ver seus leads, conversas e operacao em tempo real.
            </p>
          </div>

          <div className="space-y-3">
            <div>
              <label
                htmlFor="login-email"
                className="text-2xs text-ink-dim mb-1.5 block font-medium tracking-[0.14em] uppercase"
              >
                Email corporativo
              </label>
              <div className="relative">
                <Mail className="text-ink-dim absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2" />
                <Input
                  id="login-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="voce@empresa.com.br"
                  className="pl-9"
                  // Autofocus deliberate: primary task on this page is entering
                  // credentials; users expect cursor parked here.
                  // eslint-disable-next-line jsx-a11y/no-autofocus
                  autoFocus
                />
              </div>
            </div>

            <div>
              <label
                htmlFor="login-password"
                className="text-2xs text-ink-dim mb-1.5 flex items-center justify-between font-medium tracking-[0.14em] uppercase"
              >
                Senha
                <button
                  type="button"
                  className="text-accent tracking-normal normal-case hover:underline"
                >
                  Esqueci
                </button>
              </label>
              <div className="relative">
                <Lock className="text-ink-dim absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2" />
                <Input
                  id="login-password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="pl-9"
                />
              </div>
            </div>
          </div>

          {errorMessage && (
            <div
              role="alert"
              className="border-danger/30 bg-danger/10 text-danger mt-4 rounded-md border px-3 py-2 text-sm"
            >
              {errorMessage}
            </div>
          )}

          <Button
            variant="primary"
            size="lg"
            className="mt-6 w-full justify-between"
            type="submit"
            disabled={login.isPending}
          >
            <span>{login.isPending ? 'Entrando…' : 'Entrar na operacao'}</span>
            <ArrowRight className="h-4 w-4" />
          </Button>

          <div className="relative my-6 flex items-center">
            <span className="bg-hairline h-px flex-1" />
            <span className="text-2xs text-ink-dim mx-3 tracking-[0.14em] uppercase">ou</span>
            <span className="bg-hairline h-px flex-1" />
          </div>

          <Button variant="outline" size="lg" className="w-full" type="button">
            Entrar com SSO do time
          </Button>

          <p className="text-ink-dim mt-8 text-center text-xs">
            Protegido por 2FA · SSO · Trilha de auditoria completa.
          </p>
        </form>

        <div className="text-2xs text-ink-dim flex items-center justify-between">
          <div>© 2026 Milennials — Torque CRM v1.0</div>
          <div className="flex items-center gap-2">
            <span>Pressione</span>
            <Kbd>⌘</Kbd>
            <Kbd>K</Kbd>
            <span>para comandos</span>
          </div>
        </div>
      </div>

      {/* Right — editorial panel */}
      <div className="relative hidden flex-1 items-stretch lg:flex">
        <div className="bg-surface relative flex-1 overflow-hidden">
          {/* Hexagons pattern as subtle background */}
          <div
            className="absolute inset-0 opacity-[0.06] dark:opacity-[0.04]"
            style={{
              backgroundImage: `url(${torqueHexagons})`,
              backgroundSize: '400px',
              backgroundRepeat: 'repeat',
              backgroundPosition: 'center',
            }}
          />
          <div className="from-bg via-bg/60 absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t to-transparent" />
          <div className="vignette absolute inset-0" />

          <div className="relative flex h-full flex-col justify-between p-12">
            {/* Meter — precision signature */}
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="shadow-hairline relative h-10 w-10 rounded-full">
                  <div className="absolute inset-2 rounded-full shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.6)]" />
                  <div
                    className="bg-accent absolute top-1/2 left-1/2 h-4 w-px origin-bottom"
                    style={{ transform: 'translate(-50%,-100%) rotate(45deg)' }}
                  />
                </div>
                <div className="font-metric text-ink-muted text-xs tabular-nums">
                  TQ—2026.04 <span className="text-ink-dim">/</span> BR-SP
                </div>
              </div>
              <div className="font-metric text-2xs text-ink-dim tracking-widest">
                STATUS <span className="text-success">● OPERACIONAL</span>
              </div>
            </div>

            <div>
              <div className="text-2xs text-ink-muted mb-6 inline-flex items-center gap-2 font-medium tracking-[0.16em] uppercase">
                <span className="bg-accent h-px w-8" />
                Decisao #018 do vault
              </div>
              <blockquote className="font-display tracking-tightest text-ink text-[2rem] leading-[1.15]">
                <span className="text-accent-gradient">Se parece template, reprovou.</span> Cada
                tela e uma decisao — nada e copia de kit pronto sem intencao.
              </blockquote>
              <div className="text-ink-muted mt-6 text-sm">
                — Principios do sistema <span className="text-ink-dim">· 18</span>
              </div>
            </div>

            {/* Three axes */}
            <div className="grid grid-cols-3 gap-8 pt-8 shadow-[inset_0_1px_0_0_hsl(var(--hairline)/0.5)]">
              {[
                { k: '30', u: 'organizacoes ativas' },
                { k: '12ms', u: 'p50 ingestao webhook' },
                { k: '9', u: 'agentes especializados' },
              ].map((m) => (
                <div key={m.u}>
                  <div className="font-metric text-ink text-[1.5rem] leading-none tabular-nums">
                    {m.k}
                  </div>
                  <div className="text-2xs text-ink-dim mt-1 tracking-[0.12em] uppercase">
                    {m.u}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
