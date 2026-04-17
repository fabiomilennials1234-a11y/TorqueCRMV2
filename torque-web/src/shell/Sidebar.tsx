import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Columns3,
  MessagesSquare,
  CheckSquare,
  Workflow,
  Megaphone,
  Sparkles,
  BarChart3,
  Users,
  Package,
  Settings2,
  LifeBuoy,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Tooltip } from "@/ui/tooltip";
import { cn } from "@/lib/utils";
import { useTheme } from "@/providers/ThemeProvider";
import { OrgSwitcher } from "./OrgSwitcher";

import torqueLogoDark from "@/assets/torque-logo.png";
import torqueLogoLight from "@/assets/torque-logo-dark.png";

type NavItem = {
  to: string;
  label: string;
  icon: LucideIcon;
  shortcut?: string;
  badge?: string;
};

const vendasNav: NavItem[] = [
  { to: "/", label: "Visao geral", icon: LayoutDashboard, shortcut: "G D" },
  { to: "/pipeline", label: "Funis", icon: Columns3, shortcut: "G P", badge: "24" },
  { to: "/inbox", label: "Conversas", icon: MessagesSquare, shortcut: "G I", badge: "9" },
  { to: "/follow-ups", label: "Follow-ups", icon: CheckSquare, shortcut: "G F" },
];

const automacaoNav: NavItem[] = [
  { to: "/workflows", label: "Fluxos", icon: Workflow, shortcut: "G W" },
  { to: "/campaigns", label: "Campanhas", icon: Megaphone, shortcut: "G C" },
];

const inteligenciaNav: NavItem[] = [
  { to: "/copilot", label: "Agentes IA", icon: Sparkles, shortcut: "G A" },
  { to: "/analytics", label: "Analytics", icon: BarChart3, shortcut: "G N" },
];

const equipeNav: NavItem[] = [
  { to: "/team", label: "Time", icon: Users, shortcut: "G T" },
  { to: "/products", label: "Produtos", icon: Package },
];

const footerNav: NavItem[] = [
  { to: "/settings", label: "Configuracoes", icon: Settings2, shortcut: "G S" },
  { to: "/help", label: "Ajuda", icon: LifeBuoy },
];

export function Sidebar() {
  const { theme } = useTheme();

  return (
    <aside
      className={cn(
        "flex h-full w-[232px] shrink-0 flex-col",
        "bg-surface/80 backdrop-blur-[8px]",
        "shadow-[inset_-1px_0_0_0_hsl(var(--hairline)/0.5)]",
      )}
    >
      <div className="flex items-center gap-2.5 px-4 pt-4 pb-3">
        <img
          src={theme === "dark" ? torqueLogoDark : torqueLogoLight}
          alt="Torque"
          className="h-7 w-auto"
        />
        <span className="ml-auto text-2xs font-metric text-ink-dim tabular-nums">
          v1.0
        </span>
      </div>

      <div className="px-3 pb-3">
        <OrgSwitcher />
      </div>

      <nav className="flex-1 overflow-y-auto px-2">
        <NavGroup label="Vendas">
          {vendasNav.map((item) => (
            <SidebarLink key={item.to} item={item} />
          ))}
        </NavGroup>
        <NavGroup label="Automacao">
          {automacaoNav.map((item) => (
            <SidebarLink key={item.to} item={item} />
          ))}
        </NavGroup>
        <NavGroup label="Inteligencia">
          {inteligenciaNav.map((item) => (
            <SidebarLink key={item.to} item={item} />
          ))}
        </NavGroup>
        <NavGroup label="Equipe">
          {equipeNav.map((item) => (
            <SidebarLink key={item.to} item={item} />
          ))}
        </NavGroup>
      </nav>

      <div className="shadow-[inset_0_1px_0_0_hsl(var(--hairline)/0.5)] px-2 py-3">
        {footerNav.map((item) => (
          <SidebarLink key={item.to} item={item} />
        ))}
      </div>

      <div className="px-4 pb-4 pt-2">
        <div className="rounded-md bg-elevated/50 p-3 shadow-hairline">
          <div className="flex items-center justify-between">
            <span className="text-2xs uppercase tracking-[0.12em] text-ink-dim">
              Plano Growth
            </span>
            <span className="font-metric text-xs text-ink-muted">64%</span>
          </div>
          <div className="mt-2 h-1 overflow-hidden rounded-full bg-hairline">
            <div
              className="h-full rounded-full bg-accent"
              style={{ width: "64%" }}
            />
          </div>
          <p className="mt-2 text-2xs text-ink-dim">
            3.200 / 5.000 leads este mes
          </p>
        </div>
      </div>
    </aside>
  );
}

function NavGroup({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mb-6">
      <div className="px-2 pb-1.5 pt-2 text-2xs font-medium uppercase tracking-[0.14em] text-ink-dim">
        {label}
      </div>
      <ul className="space-y-1">{children}</ul>
    </div>
  );
}

function SidebarLink({ item }: { item: NavItem }) {
  const Icon = item.icon;
  return (
    <li>
      <Tooltip content={item.label} shortcut={item.shortcut} side="right">
        <NavLink
          to={item.to}
          end={item.to === "/"}
          className={({ isActive }) =>
            cn(
              "group relative flex h-9 items-center gap-2.5 rounded-sm px-2 text-[0.8125rem]",
              "transition-colors duration-150",
              isActive
                ? "bg-elevated text-ink shadow-hairline"
                : "text-ink-muted hover:bg-elevated/50 hover:text-ink",
            )
          }
        >
          {({ isActive }) => (
            <>
              {isActive && (
                <span className="absolute left-0 top-1/2 h-4 w-[2px] -translate-y-1/2 rounded-full bg-accent" />
              )}
              <Icon
                className={cn(
                  "h-4 w-4 shrink-0",
                  isActive ? "text-accent" : "text-ink-dim group-hover:text-ink-muted",
                )}
                strokeWidth={1.75}
              />
              <span className="flex-1 truncate">{item.label}</span>
              {item.badge && (
                <span className="font-metric text-2xs tabular-nums text-ink-dim">
                  {item.badge}
                </span>
              )}
            </>
          )}
        </NavLink>
      </Tooltip>
    </li>
  );
}
