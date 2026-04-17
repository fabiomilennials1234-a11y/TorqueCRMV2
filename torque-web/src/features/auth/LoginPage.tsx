import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowRight, Lock, Mail } from "lucide-react";
import { Button } from "@/ui/button";
import { Input } from "@/ui/input";
import { Kbd } from "@/ui/kbd";
import { useTheme } from "@/providers/ThemeProvider";

import torqueIcon from "@/assets/torque-icon.png";
import torqueLogoDark from "@/assets/torque-logo.png";
import torqueLogoLight from "@/assets/torque-logo-dark.png";
import torqueHexagons from "@/assets/torque-hexagons.png";

export function LoginPage() {
  const nav = useNavigate();
  const { theme } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  return (
    <div className="relative flex min-h-screen bg-bg text-ink">
      {/* Left — form */}
      <div className="relative flex flex-1 flex-col justify-between px-10 py-10 lg:px-16">
        <div className="flex items-center gap-2.5">
          <img
            src={theme === "dark" ? torqueLogoDark : torqueLogoLight}
            alt="Torque"
            className="h-7 w-auto"
          />
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            nav("/");
          }}
          className="mx-auto w-full max-w-[380px]"
        >
          <div className="mb-10">
            <div className="mb-4 flex justify-center">
              <img
                src={torqueIcon}
                alt=""
                className="h-14 w-14"
              />
            </div>
            <h1 className="font-display text-[2.4rem] leading-[1.02] tracking-tightest text-center">
              Bem-vindo de volta.
            </h1>
            <p className="mt-3 text-[0.9375rem] leading-relaxed text-ink-muted text-center">
              Entre para ver seus leads, conversas e operacao em tempo real.
            </p>
          </div>

          <div className="space-y-3">
            <div>
              <label className="mb-1.5 block text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
                Email corporativo
              </label>
              <div className="relative">
                <Mail className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="voce@empresa.com.br"
                  className="pl-9"
                  autoFocus
                />
              </div>
            </div>

            <div>
              <label className="mb-1.5 flex items-center justify-between text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
                Senha
                <a href="#" className="normal-case tracking-normal text-accent hover:underline">
                  Esqueci
                </a>
              </label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-dim" />
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="pl-9"
                />
              </div>
            </div>
          </div>

          <Button
            variant="primary"
            size="lg"
            className="mt-6 w-full justify-between"
            type="submit"
          >
            <span>Entrar na operacao</span>
            <ArrowRight className="h-4 w-4" />
          </Button>

          <div className="relative my-6 flex items-center">
            <span className="flex-1 h-px bg-hairline" />
            <span className="mx-3 text-2xs uppercase tracking-[0.14em] text-ink-dim">
              ou
            </span>
            <span className="flex-1 h-px bg-hairline" />
          </div>

          <Button variant="outline" size="lg" className="w-full" type="button">
            Entrar com SSO do time
          </Button>

          <p className="mt-8 text-center text-xs text-ink-dim">
            Protegido por 2FA · SSO · Trilha de auditoria completa.
          </p>
        </form>

        <div className="flex items-center justify-between text-2xs text-ink-dim">
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
      <div className="relative hidden lg:flex flex-1 items-stretch">
        <div className="relative flex-1 overflow-hidden bg-surface">
          {/* Hexagons pattern as subtle background */}
          <div
            className="absolute inset-0 opacity-[0.06] dark:opacity-[0.04]"
            style={{
              backgroundImage: `url(${torqueHexagons})`,
              backgroundSize: "400px",
              backgroundRepeat: "repeat",
              backgroundPosition: "center",
            }}
          />
          <div className="absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t from-bg via-bg/60 to-transparent" />
          <div className="absolute inset-0 vignette" />

          <div className="relative flex h-full flex-col justify-between p-12">
            {/* Meter — precision signature */}
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-full shadow-hairline relative">
                  <div className="absolute inset-2 rounded-full shadow-[inset_0_0_0_1px_hsl(var(--accent)/0.6)]" />
                  <div
                    className="absolute left-1/2 top-1/2 h-4 w-px origin-bottom bg-accent"
                    style={{ transform: "translate(-50%,-100%) rotate(45deg)" }}
                  />
                </div>
                <div className="font-metric text-xs text-ink-muted tabular-nums">
                  TQ—2026.04 <span className="text-ink-dim">/</span> BR-SP
                </div>
              </div>
              <div className="font-metric text-2xs tracking-widest text-ink-dim">
                STATUS <span className="text-success">● OPERACIONAL</span>
              </div>
            </div>

            <div>
              <div className="mb-6 inline-flex items-center gap-2 text-2xs font-medium uppercase tracking-[0.16em] text-ink-muted">
                <span className="h-px w-8 bg-accent" />
                Decisao #018 do vault
              </div>
              <blockquote className="font-display text-[2rem] leading-[1.15] tracking-tightest text-ink">
                <span className="text-accent-gradient">Se parece template, reprovou.</span>{" "}
                Cada tela e uma decisao — nada e copia de kit pronto sem intencao.
              </blockquote>
              <div className="mt-6 text-sm text-ink-muted">
                — Principios do sistema <span className="text-ink-dim">· 18</span>
              </div>
            </div>

            {/* Three axes */}
            <div className="grid grid-cols-3 gap-8 pt-8 shadow-[inset_0_1px_0_0_hsl(var(--hairline)/0.5)]">
              {[
                { k: "30", u: "organizacoes ativas" },
                { k: "12ms", u: "p50 ingestao webhook" },
                { k: "9", u: "agentes especializados" },
              ].map((m) => (
                <div key={m.u}>
                  <div className="font-metric text-[1.5rem] leading-none text-ink tabular-nums">
                    {m.k}
                  </div>
                  <div className="mt-1 text-2xs uppercase tracking-[0.12em] text-ink-dim">
                    {m.u}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
